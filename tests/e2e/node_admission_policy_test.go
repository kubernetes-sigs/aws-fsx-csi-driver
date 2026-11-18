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
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

const (
	agentNotReadyTaintKey = "fsx.csi.aws.com/agent-not-ready"
	testTaintKey          = "fsx-node-vap"
	nodeNamespace         = "kube-system"
	nodeServiceAccount    = "fsx-csi-node-sa"
	nodeNameExtraKey      = "authentication.kubernetes.io/node-name"

	otherNodeDenied  = "may only update its own Node object"
	onlyTaintsDenied = "may only change the taints of the Node its pod runs on"
	taintDenied      = "may only remove the fsx.csi.aws.com/agent-not-ready taint"
)

// The chart's ValidatingAdmissionPolicy limits the node pods' service account to
// removing the agent-not-ready taint from the Node each pod runs on. The spec uses a real
// token of that service account, bound to a running node pod, like the node
// pod's own token, and ends with the node pod's own startup taint removal.
var _ = Describe("[fsx-csi-e2e] Node ValidatingAdmissionPolicy", func() {
	f := framework.NewDefaultFramework("fsx-node-vap")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged

	It("limits the node pods' service account to removing the agent-not-ready taint from their own Node", func(ctx context.Context) {
		cs := f.ClientSet
		serverVersion, err := cs.Discovery().ServerVersion()
		framework.ExpectNoError(err)
		if version.MustParseGeneric(serverVersion.GitVersion).LessThan(version.MajorMinor(1, 30)) {
			Skip("the policy needs Kubernetes 1.30 or later")
		}

		// A running node pod, a token of its service account bound to it, and another node.
		pod := runningNodePod(ctx, cs, "")
		ownNode := pod.Spec.NodeName
		nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		framework.ExpectNoError(err)
		otherNode := ""
		for _, n := range nodes.Items {
			if n.Name != ownNode {
				otherNode = n.Name
				break
			}
		}
		Expect(otherNode).NotTo(BeEmpty(), "the cluster needs a second node")

		expiration := int64(600)
		token, err := cs.CoreV1().ServiceAccounts(nodeNamespace).CreateToken(ctx, nodeServiceAccount, &authenticationv1.TokenRequest{
			Spec: authenticationv1.TokenRequestSpec{
				ExpirationSeconds: &expiration,
				BoundObjectRef:    &authenticationv1.BoundObjectReference{Kind: "Pod", APIVersion: "v1", Name: pod.Name, UID: pod.UID},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		nodeClient := clientWithToken(f.ClientConfig(), token.Status.Token)

		// The API server reads the node from the token, as the policy expects.
		review, err := nodeClient.AuthenticationV1().SelfSubjectReviews().Create(ctx, &authenticationv1.SelfSubjectReview{}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		Expect(review.Status.UserInfo.Username).To(Equal(fmt.Sprintf("system:serviceaccount:%s:%s", nodeNamespace, nodeServiceAccount)))
		Expect(review.Status.UserInfo.Extra[nodeNameExtraKey]).To(ConsistOf(authenticationv1.ExtraValue{ownNode}))

		// Taint both nodes for a few seconds, with NoSchedule so running pods stay.
		for _, taint := range []v1.Taint{{Key: agentNotReadyTaintKey, Effect: v1.TaintEffectNoSchedule}, {Key: testTaintKey, Effect: v1.TaintEffectNoSchedule}} {
			addTaint(ctx, cs, ownNode, taint)
		}
		addTaint(ctx, cs, otherNode, v1.Taint{Key: agentNotReadyTaintKey, Effect: v1.TaintEffectNoSchedule})
		DeferCleanup(func(ctx context.Context) {
			for _, node := range []string{ownNode, otherNode} {
				for _, key := range []string{agentNotReadyTaintKey, testTaintKey} {
					if hasTaint(ctx, cs, node, key) {
						framework.ExpectNoError(removeTaint(ctx, cs, node, key))
					}
				}
			}
		})

		By("denying the taint removal on another node")
		Expect(removeTaint(ctx, nodeClient, otherNode, agentNotReadyTaintKey)).To(MatchError(ContainSubstring(otherNodeDenied)))

		By("denying label, annotation and unschedulable changes on its own node")
		for _, patch := range []string{
			`{"metadata":{"labels":{"fsx-node-vap":"denied"}}}`,
			`{"metadata":{"annotations":{"fsx-node-vap":"denied"}}}`,
			`{"spec":{"unschedulable":true}}`,
		} {
			_, err = nodeClient.CoreV1().Nodes().Patch(ctx, ownNode, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			Expect(err).To(MatchError(ContainSubstring(onlyTaintsDenied)), patch)
		}

		By("denying adding or removing another taint on its own node")
		err = patchTaints(ctx, nodeClient, ownNode, func(taints []v1.Taint) []v1.Taint {
			return append(taints, v1.Taint{Key: testTaintKey + "-added", Effect: v1.TaintEffectNoSchedule})
		})
		Expect(err).To(MatchError(ContainSubstring(taintDenied)))
		Expect(removeTaint(ctx, nodeClient, ownNode, testTaintKey)).To(MatchError(ContainSubstring(taintDenied)))

		By("denying Node status updates and Node deletion")
		// RBAC grants the node pods' service account only get and patch on nodes, so these are
		// forbidden before admission; the policy also denies them.
		_, err = nodeClient.CoreV1().Nodes().Patch(ctx, ownNode, types.MergePatchType,
			[]byte(`{"status":{"phase":"Running"}}`), metav1.PatchOptions{}, "status")
		Expect(apierrors.IsForbidden(err)).To(BeTrue(), "status patch: %v", err)
		err = nodeClient.CoreV1().Nodes().Delete(ctx, otherNode, metav1.DeleteOptions{})
		Expect(apierrors.IsForbidden(err)).To(BeTrue(), "delete: %v", err)

		By("allowing reordering the taints of its own node")
		framework.ExpectNoError(patchTaints(ctx, nodeClient, ownNode, func(taints []v1.Taint) []v1.Taint {
			reordered := slices.Clone(taints)
			slices.Reverse(reordered)
			return reordered
		}))

		By("allowing the agent-not-ready taint removal on its own node")
		framework.ExpectNoError(removeTaint(ctx, nodeClient, ownNode, agentNotReadyTaintKey))
		Expect(hasTaint(ctx, cs, ownNode, agentNotReadyTaintKey)).To(BeFalse())
		Expect(hasTaint(ctx, cs, ownNode, testTaintKey)).To(BeTrue())

		By("letting a restarted node pod remove the agent-not-ready taint from its node")
		// Last: deleting the node pod invalidates the token above.
		addTaint(ctx, cs, ownNode, v1.Taint{Key: agentNotReadyTaintKey, Effect: v1.TaintEffectNoSchedule})
		framework.ExpectNoError(cs.CoreV1().Pods(nodeNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}))
		Eventually(func(ctx context.Context) bool {
			return hasTaint(ctx, cs, ownNode, agentNotReadyTaintKey)
		}).WithContext(ctx).WithTimeout(3 * time.Minute).WithPolling(2 * time.Second).Should(BeFalse())
		Expect(runningNodePod(ctx, cs, ownNode).Name).NotTo(Equal(pod.Name))
	})
})

// runningNodePod returns a running node pod, on the given node when it is not empty.
func runningNodePod(ctx context.Context, cs clientset.Interface, node string) *v1.Pod {
	var pod *v1.Pod
	Eventually(func(ctx context.Context) *v1.Pod {
		pods, err := cs.CoreV1().Pods(nodeNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app=fsx-csi-node"})
		framework.ExpectNoError(err)
		for i := range pods.Items {
			p := &pods.Items[i]
			if p.Status.Phase == v1.PodRunning && p.DeletionTimestamp == nil && p.Spec.NodeName != "" && (node == "" || p.Spec.NodeName == node) {
				pod = p
				return p
			}
		}
		return nil
	}).WithContext(ctx).WithTimeout(3*time.Minute).WithPolling(2*time.Second).ShouldNot(BeNil(), "no running fsx-csi-node pod")
	return pod
}

// clientWithToken returns a client for the same API server that authenticates
// only with the given bearer token.
func clientWithToken(config *rest.Config, token string) clientset.Interface {
	c := rest.AnonymousClientConfig(config)
	c.BearerToken = token
	client, err := clientset.NewForConfig(c)
	framework.ExpectNoError(err)
	return client
}

func hasTaint(ctx context.Context, cs clientset.Interface, node, key string) bool {
	n, err := cs.CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{})
	framework.ExpectNoError(err)
	for _, t := range n.Spec.Taints {
		if t.Key == key {
			return true
		}
	}
	return false
}

func addTaint(ctx context.Context, cs clientset.Interface, node string, taint v1.Taint) {
	framework.ExpectNoError(patchTaints(ctx, cs, node, func(taints []v1.Taint) []v1.Taint {
		return append(taints, taint)
	}))
}

func removeTaint(ctx context.Context, cs clientset.Interface, node, key string) error {
	return patchTaints(ctx, cs, node, func(taints []v1.Taint) []v1.Taint {
		var keep []v1.Taint
		for _, t := range taints {
			if t.Key != key {
				keep = append(keep, t)
			}
		}
		return keep
	})
}

// patchTaints changes a node's taints with the same JSON patch the node pod
// sends in removeNotReadyTaint: test the current taints, then replace them. A
// node without taints gets an add instead, since the test needs the path. The
// patch carries the resourceVersion it read, so it does not overwrite
// concurrent taint changes, and is retried on conflict.
func patchTaints(ctx context.Context, cs clientset.Interface, node string, change func([]v1.Taint) []v1.Taint) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		var n *v1.Node
		n, err = cs.CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{})
		if err != nil {
			return err
		}
		resourceVersion := map[string]interface{}{"op": "add", "path": "/metadata/resourceVersion", "value": n.ResourceVersion}
		ops := []map[string]interface{}{
			resourceVersion,
			{"op": "add", "path": "/spec/taints", "value": change(n.Spec.Taints)},
		}
		if len(n.Spec.Taints) > 0 {
			ops = []map[string]interface{}{
				resourceVersion,
				{"op": "test", "path": "/spec/taints", "value": n.Spec.Taints},
				{"op": "replace", "path": "/spec/taints", "value": change(n.Spec.Taints)},
			}
		}
		patch, merr := json.Marshal(ops)
		if merr != nil {
			return merr
		}
		_, err = cs.CoreV1().Nodes().Patch(ctx, node, types.JSONPatchType, patch, metav1.PatchOptions{})
		// Retry when the node changed in between, for example by the node lifecycle controller.
		if err == nil || !(apierrors.IsConflict(err) || strings.Contains(err.Error(), "testing value")) {
			return err
		}
	}
	return err
}
