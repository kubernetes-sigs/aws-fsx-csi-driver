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

package e2e

import (
	"fmt"
	"log"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
	"sigs.k8s.io/aws-fsx-csi-driver/tests/e2e/driver"
	"sigs.k8s.io/aws-fsx-csi-driver/tests/e2e/testsuites"
)

var _ = Describe("[fsx-csi-e2e] INTELLIGENT_TIERING Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("fsx")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged

	var (
		cs               clientset.Interface
		ns               *v1.Namespace
		dvr              driver.PVTestDriver
		cloud            *cloud
		subnetId         string
		securityGroupIds []string
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		dvr = driver.InitFSxCSIDriver()
		cloud = NewCloud(*region)
		instance, err := cloud.getNodeInstance(*clusterName)
		if err != nil {
			Fail(fmt.Sprintf("failed to get node instance %v", err))
		}
		securityGroupIds = getSecurityGroupIds(instance)
		subnetId = *instance.SubnetId
	})

	It("should create an INTELLIGENT_TIERING volume with default configuration", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":          subnetId,
							"securityGroupIds":  strings.Join(securityGroupIds, ","),
							"storageType":       "INTELLIGENT_TIERING",
							"deploymentType":    "PERSISTENT_2",
							"throughputCapacity": "4000",
							"skipFinalBackup":   "true", // Speed up test cleanup
						},
						ClaimSize: "1200Gi", // This will be ignored for INTELLIGENT_TIERING
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: dvr,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create an INTELLIGENT_TIERING volume with custom metadata IOPS", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":          subnetId,
							"securityGroupIds":  strings.Join(securityGroupIds, ","),
							"storageType":       "INTELLIGENT_TIERING",
							"deploymentType":    "PERSISTENT_2",
							"throughputCapacity": "4000",
							"metadataIops":      "12000",
							"skipFinalBackup":   "true", // Speed up test cleanup
						},
						ClaimSize: "1200Gi", // This will be ignored for INTELLIGENT_TIERING
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: dvr,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create an INTELLIGENT_TIERING volume with NO_CACHE mode", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":                 subnetId,
							"securityGroupIds":         strings.Join(securityGroupIds, ","),
							"storageType":              "INTELLIGENT_TIERING",
							"deploymentType":           "PERSISTENT_2",
							"throughputCapacity":        "4000",
							"dataReadCacheSizingMode":  "NO_CACHE",
							"skipFinalBackup":          "true", // Speed up test cleanup
						},
						ClaimSize: "1200Gi", // This will be ignored for INTELLIGENT_TIERING
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: dvr,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create an INTELLIGENT_TIERING volume with USER_PROVISIONED cache", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":                 subnetId,
							"securityGroupIds":         strings.Join(securityGroupIds, ","),
							"storageType":              "INTELLIGENT_TIERING",
							"deploymentType":           "PERSISTENT_2",
							"throughputCapacity":        "4000",
							"dataReadCacheSizingMode":  "USER_PROVISIONED",
							"dataReadCacheSizeGiB":     "1000",
							"skipFinalBackup":          "true", // Speed up test cleanup
						},
						ClaimSize: "1200Gi", // This will be ignored for INTELLIGENT_TIERING
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: dvr,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create an INTELLIGENT_TIERING volume and verify backup on deletion", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		
		// This test verifies that when skipFinalBackup is NOT set (or set to false),
		// a backup is created when the filesystem is deleted
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":          subnetId,
							"securityGroupIds":  strings.Join(securityGroupIds, ","),
							"storageType":       "INTELLIGENT_TIERING",
							"deploymentType":    "PERSISTENT_2",
							"throughputCapacity": "4000",
							// NOTE: skipFinalBackup is NOT set, so a backup should be created on deletion
						},
						ClaimSize: "1200Gi", // This will be ignored for INTELLIGENT_TIERING
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTestWithBackupVerification{
			CSIDriver: dvr,
			Pods:      pods,
			Cloud:     cloud,
		}
		test.Run(cs, ns)
	})
})
