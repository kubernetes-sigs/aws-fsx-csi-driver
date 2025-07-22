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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/test/e2e/framework"
	storageframework "k8s.io/kubernetes/test/e2e/storage/framework"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	fsx "sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	fsxcsidriver "sigs.k8s.io/aws-fsx-csi-driver/pkg/driver"
)

type fsxVolume struct {
	c            *cloud
	fileSystemId string
	dnsName      string
}

func (v *fsxVolume) DeleteVolume() {
	ctx := context.Background()
	err := v.c.DeleteFileSystem(ctx, v.fileSystemId)
	if err != nil {
		Fail(fmt.Sprintf("failed to delete filesystem %s", err))
	}
}

type fsxDriver struct {
	driverName string
}

var _ storageframework.TestDriver = &fsxDriver{}
var _ storageframework.PreprovisionedPVTestDriver = &fsxDriver{}

func InitFSxCSIDriver() storageframework.TestDriver {
	return &fsxDriver{
		driverName: fsxcsidriver.DriverName,
	}
}

func (e *fsxDriver) GetDriverInfo() *storageframework.DriverInfo {
	return &storageframework.DriverInfo{
		Name:                 e.driverName,
		SupportedFsType:      sets.NewString(""),
		SupportedMountOption: sets.NewString("flock", "ro"),
		Capabilities: map[storageframework.Capability]bool{
			storageframework.CapPersistence: true,
			storageframework.CapExec:        true,
			storageframework.CapMultiPODs:   true,
			storageframework.CapRWX:         true,
		},
	}
}

func (e *fsxDriver) SkipUnsupportedTest(storageframework.TestPattern) {}

func (e *fsxDriver) PrepareTest(f *framework.Framework) (*storageframework.PerTestConfig, func()) {
	By("PrepareTest")
	cancelPodLogs := utils.StartPodLogs(f, f.Namespace)

	return &storageframework.PerTestConfig{
			Driver:    e,
			Prefix:    "fsx",
			Framework: f,
		}, func() {
			By("Cleaning up FSx CSI driver")
			cancelPodLogs()
		}
}

func (e *fsxDriver) CreateVolume(config *storageframework.PerTestConfig, volType storageframework.TestVolType) storageframework.TestVolume {
	c := NewCloud(*region)
	instance, err := c.getNodeInstance(*clusterName)
	if err != nil {
		Fail(fmt.Sprintf("failed to get node instance %v", err))
	}
	securityGroupIds := getSecurityGroupIds(instance)
	subnetId := *instance.SubnetId

	ctx := context.Background()
	options := &fsx.FileSystemOptions{
		CapacityGiB:            3600,
		SubnetId:               subnetId,
		SecurityGroupIds:       securityGroupIds,
		FileSystemTypeVersion:  "2.15",
	}
	ns := config.Framework.Namespace.Name
	volumeName := fmt.Sprintf("fsx-e2e-test-volume-%s", ns)
	fs, err := c.CreateFileSystem(ctx, volumeName, options)
	if err != nil {
		Fail(fmt.Sprintf("failed to created filesystem %s", err))
	}

	err = c.WaitForFileSystemAvailable(ctx, fs.FileSystemId)
	if err != nil {
		Fail(fmt.Sprintf("failed to wait on filesystem %s", err))
	}

	return &fsxVolume{
		c:            c,
		fileSystemId: fs.FileSystemId,
		dnsName:      fs.DnsName,
	}
}

func (e *fsxDriver) GetPersistentVolumeSource(readOnly bool, fsType string, volume storageframework.TestVolume) (*v1.PersistentVolumeSource, *v1.VolumeNodeAffinity) {
	v := volume.(*fsxVolume)
	pvSource := v1.PersistentVolumeSource{
		CSI: &v1.CSIPersistentVolumeSource{
			Driver:       e.driverName,
			VolumeHandle: v.fileSystemId,
			VolumeAttributes: map[string]string{
				"dnsname": v.dnsName,
			},
		},
	}
	return &pvSource, nil
}

// TOOD: uncomment for testing dynamic provisioning

//var _ testsuites.DynamicPVTestDriver = &fsxDriver{}
//func (e *fsxDriver) GetDynamicProvisionStorageClass(config *testsuites.PerTestConfig, fsType string) *storagev1.StorageClass {
//	c := NewCloud(*region)
//	instance, err := c.getNodeInstance(*clusterName)
//	if err != nil {
//		Fail(fmt.Sprintf("failed to get node instance %v", err))
//	}
//	securityGroupIds := getSecurityGroupIds(instance)
//	subnetId := *instance.SubnetId
//
//	provisioner := e.driverName
//	parameters := map[string]string{
//		"subnetId":         subnetId,
//		"securityGroupIds": strings.Join(securityGroupIds, ","),
//	}
//
//	ns := config.Framework.Namespace.Name
//	suffix := fmt.Sprintf("%s-sc", e.driverName)
//
//	return &storagev1.StorageClass{
//		TypeMeta: metav1.TypeMeta{
//			Kind: "StorageClass",
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			// Name must be unique, so let's base it on namespace name and use GenerateName
//			Name: names.SimpleNameGenerator.GenerateName(ns + "-" + suffix),
//		},
//		Provisioner: provisioner,
//		Parameters:  parameters,
//	}
//}

// List of testSuites to be executed in below loop
var csiTestSuites = []func() storageframework.TestSuite{
	testsuites.InitVolumesTestSuite,
	testsuites.InitVolumeIOTestSuite,
	testsuites.InitVolumeModeTestSuite,
	testsuites.InitSubPathTestSuite,
	testsuites.InitProvisioningTestSuite,
	//testsuites.InitSnapshottableTestSuite,
	testsuites.InitVolumeExpandTestSuite,
	testsuites.InitMultiVolumeTestSuite,
}

var _ = Describe("FSx CSI Driver Conformance", func() {
	driver := InitFSxCSIDriver()
	Context(storageframework.GetDriverNameWithFeatureTags(driver), func() {
		storageframework.DefineTestSuites(driver, csiTestSuites)
	})
})
