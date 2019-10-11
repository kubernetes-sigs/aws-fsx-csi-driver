/*
Copyright 2018 The Kubernetes Authors.

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

	"github.com/kubernetes-sigs/aws-fsx-csi-driver/tests/e2e/driver"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[fsx-csi-e2e] Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("fsx")

	var (
		cs               clientset.Interface
		ns               *v1.Namespace
		dvr              driver.PVTestDriver
		subnetId         string
		securityGroupIds []string
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		dvr = driver.InitFSxCSIDriver()
		instance, err := getNodeInstance(*clusterName)
		if err != nil {
			Fail(fmt.Sprintf("could ge node instance %v", err))
		}
		securityGroupIds = getSecurityGroupIds(instance)
		subnetId = *instance.SubnetId
	})

	It("should create a volume on demand with subnetId and security groups", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":         subnetId,
							"securityGroupIds": strings.Join(securityGroupIds, ","),
						},
						ClaimSize: "3600Gi",
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

	It("should create a volume on demand with flock mount option", func() {
		log.Printf("Using subnet ID %s security group ID %s", subnetId, securityGroupIds)
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						Parameters: map[string]string{
							"subnetId":         subnetId,
							"securityGroupIds": strings.Join(securityGroupIds, ","),
						},
						MountOptions: []string{"flock"},
						ClaimSize:    "1200Gi",
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
})
