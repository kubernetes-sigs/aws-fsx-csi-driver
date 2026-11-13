/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	fsx "sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
)

var controllerRoleName = flag.String("controller-role-name", "", "IAM role the controller uses through EKS Pod Identity; the isolation tests are skipped when empty")

const (
	// The tag key the Pod Identity policies check on file systems.
	isolationTagKey = "eks:eks-cluster-name"
	// The cluster name the out-of-band file system is tagged with.
	otherClusterName = "fsx-e2e-other-cluster"

	controllerNamespace  = "kube-system"
	controllerDeployment = "fsx-csi-controller"
	controllerContainer  = "fsx-plugin"
	controllerSA         = "fsx-csi-controller-sa"

	fsCreateTimeout = 30 * time.Minute
	poll            = 15 * time.Second
	// Time for IAM to apply a changed inline policy before the controller restarts.
	iamPropagation = 30 * time.Second
)

// The tests walk one cluster through the rollout in docs/cluster-tag-isolation.md:
// a controller without isolation creates a file system, backfill tags it, and
// enforce keeps working on it while refusing a file system tagged for another
// cluster, both in the controller and in IAM. The controller uses EKS Pod Identity
// with the docs/iam/cluster-tag-*.json policies, swapped between steps. The specs
// depend on each other and run in order.
var _ = Describe("[fsx-csi-isolation] Cluster tag isolation", Ordered, func() {
	var (
		ctx              = context.Background()
		cs               clientset.Interface
		c                *cloud
		ns               string
		scName           string
		subnetId         string
		securityGroupIds []string

		legacyPVC, enforcedPVC string
		legacyFS, enforcedFS   string
		foreignFS              string
		foreignPV, foreignPVC  string
	)

	policy := func(name string) string {
		return filepath.Join(framework.TestContext.RepoRoot, "docs", "iam", name)
	}
	setPolicies := func(files ...string) {
		args := append([]string{"set-policies", *controllerRoleName}, files...)
		runScript("hack/e2e/pod-identity.sh", args...)
		time.Sleep(iamPropagation)
	}
	setIsolation := func(isolation string, tagged bool) {
		name, key := "", ""
		if tagged {
			name, key = *clusterName, isolationTagKey
		}
		runScript("hack/e2e/helm-upgrade.sh",
			"--set", "controller.clusterName="+name,
			"--set", "controller.clusterTagKey="+key,
			"--set", "controller.clusterTagIsolation="+isolation)
	}

	BeforeAll(func() {
		if *controllerRoleName == "" {
			Skip("-controller-role-name is not set")
		}
		var err error
		cs, err = framework.LoadClientset()
		framework.ExpectNoError(err)
		c = NewCloud(*region)
		instance, err := c.getNodeInstance(*clusterName)
		framework.ExpectNoError(err)
		subnetId = *instance.SubnetId
		securityGroupIds = getSecurityGroupIds(instance)

		nsObj, err := cs.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "fsx-isolation-"},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		ns = nsObj.Name

		reclaim := v1.PersistentVolumeReclaimDelete
		sc, err := cs.StorageV1().StorageClasses().Create(ctx, &storagev1.StorageClass{
			ObjectMeta:  metav1.ObjectMeta{Name: ns},
			Provisioner: "fsx.csi.aws.com",
			Parameters: map[string]string{
				"subnetId":         subnetId,
				"securityGroupIds": strings.Join(securityGroupIds, ","),
			},
			ReclaimPolicy: &reclaim,
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		scName = sc.Name
	})

	AfterAll(func() {
		if cs == nil {
			return
		}
		if foreignFS != "" {
			if err := c.DeleteFileSystem(ctx, foreignFS); err != nil && !errors.Is(err, fsx.ErrNotFound) {
				framework.Logf("deleting %s: %v", foreignFS, err)
			}
		}
		if foreignPV != "" {
			_ = cs.CoreV1().PersistentVolumes().Delete(ctx, foreignPV, metav1.DeleteOptions{})
		}
		_ = cs.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
		_ = cs.StorageV1().StorageClasses().Delete(ctx, scName, metav1.DeleteOptions{})
	})

	It("creates file systems without a cluster tag before isolation", func() {
		// Start on the shared policy, which covers every file system this driver created.
		setPolicies(policy("shared-policy.json"))
		setIsolation("disable", false)

		// The file system of another cluster, created out of band with the test
		// credentials. Create it now so it is available by the enforce step.
		fs, err := c.CreateFileSystem(ctx, "fsx-e2e-foreign-"+ns, &fsx.FileSystemOptions{
			CapacityGiB:      1200,
			SubnetId:         subnetId,
			SecurityGroupIds: securityGroupIds,
			ExtraTags:        []string{isolationTagKey + "=" + otherClusterName},
		})
		framework.ExpectNoError(err)
		foreignFS = fs.FileSystemId

		legacyPVC = createPVC(ctx, cs, ns, "legacy", scName)
		legacyFS = waitForBoundFileSystem(ctx, cs, ns, legacyPVC)

		tags := describe(ctx, c, legacyFS).Tags
		Expect(tags).To(HaveKey(fsx.VolumeNameTagKey))
		Expect(tags).To(HaveKeyWithValue(fsx.SharedTagKey, fsx.SharedTagValue))
		Expect(tags).NotTo(HaveKey(isolationTagKey))

		framework.ExpectNoError(c.WaitForFileSystemAvailable(ctx, foreignFS))
	})

	It("backfills the cluster tag on existing file systems", func() {
		// As in docs/cluster-tag-isolation.md: add the tag-based policies alongside the
		// shared policy, which stays attached until enforce is set.
		setPolicies(policy("shared-policy.json"), policy("cluster-tag-policy.json"), policy("cluster-tag-backfill-policy.json"))
		setIsolation("backfill", true)

		// The controller sweeps at startup and then hourly. Restart it if the first
		// sweep ran before IAM applied the new policy.
		start := time.Now()
		Eventually(func(g Gomega) {
			if time.Since(start) > 5*time.Minute {
				start = time.Now()
				restartController(ctx, cs)
			}
			g.Expect(describe(ctx, c, legacyFS).Tags).To(HaveKeyWithValue(isolationTagKey, *clusterName))
		}).WithTimeout(20 * time.Minute).WithPolling(poll).Should(Succeed())

		Eventually(func() string {
			return controllerLogs(ctx, cs)
		}).WithTimeout(5 * time.Minute).WithPolling(poll).Should(ContainSubstring("cluster tag backfill complete"))
	})

	It("creates file systems with the cluster tag when enforcing", func() {
		setIsolation("enforce", true)
		// Then detach the shared policy and the backfill policy. The remaining specs
		// run with only cluster-tag-policy.json.
		setPolicies(policy("cluster-tag-policy.json"))

		enforcedPVC = createPVC(ctx, cs, ns, "enforced", scName)
		enforcedFS = waitForBoundFileSystem(ctx, cs, ns, enforcedPVC)
		Expect(describe(ctx, c, enforcedFS).Tags).To(HaveKeyWithValue(isolationTagKey, *clusterName))
	})

	It("refuses to delete another cluster's file system when enforcing", func() {
		foreign := describe(ctx, c, foreignFS)
		foreignPV, foreignPVC = createStaticVolume(ctx, cs, ns, "foreign", foreign)

		framework.ExpectNoError(cs.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, foreignPVC, metav1.DeleteOptions{}))
		Eventually(func() string {
			return volumeFailedDeleteMessage(ctx, cs, foreignPV)
		}).WithTimeout(5 * time.Minute).WithPolling(poll).Should(ContainSubstring("FailedPrecondition"))

		Expect(describe(ctx, c, foreignFS).Tags).To(HaveKeyWithValue(isolationTagKey, otherClusterName))

		// Keep the file system for the next spec and let AfterAll delete it.
		_, err := cs.CoreV1().PersistentVolumes().Patch(ctx, foreignPV, types.MergePatchType,
			[]byte(`{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}`), metav1.PatchOptions{})
		framework.ExpectNoError(err)
	})

	It("denies the controller role access to another cluster's file system in IAM", func() {
		// Calls FSx directly with the controller's Pod Identity role, bypassing the
		// controller's own check. Describing this cluster's file system proves the
		// credentials work; deleting or retagging the other cluster's must be denied.
		script := fmt.Sprintf(`set -u
aws fsx describe-file-systems --file-system-ids %[1]s --query 'FileSystems[0].FileSystemId' --output text && echo DESCRIBE_OK
aws fsx tag-resource --resource-arn %[2]s --tags Key=%[3]s,Value=%[4]s 2>&1 | grep -q AccessDenied && echo TAG_DENIED
aws fsx delete-file-system --file-system-id %[5]s 2>&1 | grep -q AccessDenied && echo DELETE_DENIED
true`, legacyFS, describe(ctx, c, foreignFS).ResourceARN, isolationTagKey, *clusterName, foreignFS)
		out := runWithControllerRole(ctx, cs, script)
		Expect(out).To(ContainSubstring("DESCRIBE_OK"))
		Expect(out).To(ContainSubstring("TAG_DENIED"))
		Expect(out).To(ContainSubstring("DELETE_DENIED"))

		Expect(describe(ctx, c, foreignFS).Tags).To(HaveKeyWithValue(isolationTagKey, otherClusterName))
	})

	It("deletes this cluster's file systems when enforcing", func() {
		for _, pvc := range []string{legacyPVC, enforcedPVC} {
			claim, err := cs.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvc, metav1.GetOptions{})
			framework.ExpectNoError(err)
			framework.ExpectNoError(cs.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, pvc, metav1.DeleteOptions{}))
			framework.ExpectNoError(e2epv.WaitForPersistentVolumeDeleted(ctx, cs, claim.Spec.VolumeName, poll, 10*time.Minute))
		}
	})
})

// runScript runs a script from the repository root with the environment run.sh exported.
func runScript(script string, args ...string) {
	cmd := exec.Command(filepath.Join(framework.TestContext.RepoRoot, script), args...)
	out, err := cmd.CombinedOutput()
	framework.Logf("%s %s:\n%s", script, strings.Join(args, " "), out)
	framework.ExpectNoError(err)
}

func describe(ctx context.Context, c *cloud, id string) *fsx.FileSystem {
	fs, err := c.DescribeFileSystem(ctx, id)
	framework.ExpectNoError(err)
	return fs
}

func createPVC(ctx context.Context, cs clientset.Interface, ns, name, scName string) string {
	pvc, err := cs.CoreV1().PersistentVolumeClaims(ns).Create(ctx, &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			StorageClassName: &scName,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse("1200Gi")},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return pvc.Name
}

// waitForBoundFileSystem waits for the claim to bind and returns its file system ID.
func waitForBoundFileSystem(ctx context.Context, cs clientset.Interface, ns, pvc string) string {
	framework.ExpectNoError(e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, cs, ns, pvc, poll, fsCreateTimeout))
	claim, err := cs.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvc, metav1.GetOptions{})
	framework.ExpectNoError(err)
	pv, err := cs.CoreV1().PersistentVolumes().Get(ctx, claim.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	return pv.Spec.CSI.VolumeHandle
}

// createStaticVolume creates a PersistentVolume with reclaim policy Delete for an
// existing file system, and a claim bound to it.
func createStaticVolume(ctx context.Context, cs clientset.Interface, ns, name string, fs *fsx.FileSystem) (string, string) {
	pv, err := cs.CoreV1().PersistentVolumes().Create(ctx, &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: ns + "-" + name},
		Spec: v1.PersistentVolumeSpec{
			AccessModes:                   []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Capacity:                      v1.ResourceList{v1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", fs.CapacityGiB))},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:           "fsx.csi.aws.com",
					VolumeHandle:     fs.FileSystemId,
					VolumeAttributes: map[string]string{"dnsname": fs.DnsName, "mountname": fs.MountName},
				},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	empty := ""
	pvc, err := cs.CoreV1().PersistentVolumeClaims(ns).Create(ctx, &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			StorageClassName: &empty,
			VolumeName:       pv.Name,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: pv.Spec.Capacity[v1.ResourceStorage]},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	framework.ExpectNoError(e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, cs, ns, pvc.Name, poll, 2*time.Minute))
	return pv.Name, pvc.Name
}

func volumeFailedDeleteMessage(ctx context.Context, cs clientset.Interface, pv string) string {
	events, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=PersistentVolume,involvedObject.name=" + pv + ",reason=VolumeFailedDelete",
	})
	framework.ExpectNoError(err)
	var msgs []string
	for _, e := range events.Items {
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, "\n")
}

func controllerPods(ctx context.Context, cs clientset.Interface) []v1.Pod {
	pods, err := cs.CoreV1().Pods(controllerNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app=" + controllerDeployment})
	framework.ExpectNoError(err)
	return pods.Items
}

// controllerLogs returns the fsx-plugin logs of every controller pod.
func controllerLogs(ctx context.Context, cs clientset.Interface) string {
	var buf bytes.Buffer
	for _, pod := range controllerPods(ctx, cs) {
		out, err := cs.CoreV1().Pods(controllerNamespace).GetLogs(pod.Name, &v1.PodLogOptions{Container: controllerContainer}).DoRaw(ctx)
		if err != nil {
			framework.Logf("logs of %s: %v", pod.Name, err)
			continue
		}
		buf.Write(out)
	}
	return buf.String()
}

func restartController(ctx context.Context, cs clientset.Interface) {
	framework.Logf("restarting the controller to run another backfill sweep")
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"fsx-e2e/restartedAt":%q}}}}}`, time.Now().Format(time.RFC3339))
	_, err := cs.AppsV1().Deployments(controllerNamespace).Patch(ctx, controllerDeployment, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	framework.ExpectNoError(err)
}

// runWithControllerRole runs a shell script in an aws-cli pod that uses the
// controller's service account, so EKS Pod Identity gives it the controller's
// role, and returns its output.
func runWithControllerRole(ctx context.Context, cs clientset.Interface, script string) string {
	pod, err := cs.CoreV1().Pods(controllerNamespace).Create(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "fsx-isolation-iam-"},
		Spec: v1.PodSpec{
			ServiceAccountName: controllerSA,
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{{
				Name:    "aws",
				Image:   "public.ecr.aws/aws-cli/aws-cli:latest",
				Command: []string{"/bin/sh", "-c", script},
				Env:     []v1.EnvVar{{Name: "AWS_REGION", Value: *region}},
			}},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	defer func() {
		_ = cs.CoreV1().Pods(controllerNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}()

	Eventually(func() (v1.PodPhase, error) {
		p, err := cs.CoreV1().Pods(controllerNamespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return p.Status.Phase, nil
	}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Equal(v1.PodSucceeded))

	out, err := cs.CoreV1().Pods(controllerNamespace).GetLogs(pod.Name, &v1.PodLogOptions{}).DoRaw(ctx)
	framework.ExpectNoError(err)
	framework.Logf("aws-cli pod output:\n%s", out)
	return string(out)
}
