/*
Copyright 2024 The Kubernetes Authors.

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

package testsuites

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	awscloud "sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/tests/e2e/driver"
)

// DynamicallyProvisionedCmdVolumeTestWithBackupVerification will provision required StorageClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to create a new PV
// Testing if the Pod(s) Cmd is run with a 0 exit code
// After deletion, verifies that a backup was created and cleans it up
type DynamicallyProvisionedCmdVolumeTestWithBackupVerification struct {
	CSIDriver driver.DynamicPVTestDriver
	Pods      []PodDetails
	Cloud     awscloud.Cloud
}

func (t *DynamicallyProvisionedCmdVolumeTestWithBackupVerification) Run(client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		tpod, cleanup := pod.SetupWithDynamicVolumes(client, namespace, t.CSIDriver)
		
		// Get the filesystem ID before cleanup
		var fileSystemId string
		if len(tpod.pod.Spec.Volumes) > 0 {
			pvcName := tpod.pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName
			pvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(context.Background(), pvcName, metav1.GetOptions{})
			framework.ExpectNoError(err)
			
			pv, err := client.CoreV1().PersistentVolumes().Get(context.Background(), pvc.Spec.VolumeName, metav1.GetOptions{})
			framework.ExpectNoError(err)
			
			fileSystemId = pv.Spec.CSI.VolumeHandle
			framework.Logf("Filesystem ID: %s", fileSystemId)
		}

		By("deploying the pod")
		tpod.Create()
		By("checking that the pods command exits with no error")
		tpod.WaitForSuccess()
		
		By("cleaning up the pod")
		tpod.Cleanup()
		
		By("cleaning up volumes - this should trigger backup creation")
		for i := range cleanup {
			cleanup[i]()
		}
		
		// Wait a bit for the backup to be initiated
		By("waiting for backup to be created")
		time.Sleep(30 * time.Second)
		
		By(fmt.Sprintf("verifying backup was created for filesystem %s", fileSystemId))
		ctx := context.Background()
		backups, err := t.Cloud.GetBackupsForFileSystem(ctx, fileSystemId)
		framework.ExpectNoError(err)
		
		Expect(len(backups)).To(BeNumerically(">", 0), "Expected at least one backup to be created")
		framework.Logf("Found %d backup(s) for filesystem %s", len(backups), fileSystemId)
		
		// Clean up the backups
		By("cleaning up backups")
		for _, backup := range backups {
			if backup.BackupId != nil {
				framework.Logf("Deleting backup %s", *backup.BackupId)
				err := t.Cloud.DeleteBackup(ctx, *backup.BackupId)
				if err != nil {
					framework.Logf("Warning: failed to delete backup %s: %v", *backup.BackupId, err)
				}
			}
		}
	}
}
