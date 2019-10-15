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
	"strings"

	fsx "github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud"
	fsxcsidriver "github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/driver"
	. "github.com/onsi/ginkgo"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/storage/testpatterns"
	"k8s.io/kubernetes/test/e2e/storage/testsuites"
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
	driverName       string
	subnetId         string
	securityGroupIds []string
}

var _ testsuites.TestDriver = &fsxDriver{}
var _ testsuites.PreprovisionedPVTestDriver = &fsxDriver{}

//var _ testsuites.DynamicPVTestDriver = &fsxDriver{}

func InitFSxCSIDriver() testsuites.TestDriver {
	return &fsxDriver{
		driverName: fsxcsidriver.DriverName,
	}
}

func (e *fsxDriver) GetDriverInfo() *testsuites.DriverInfo {
	return &testsuites.DriverInfo{
		Name:                 e.driverName,
		SupportedFsType:      sets.NewString(""),
		SupportedMountOption: sets.NewString("flock", "ro"),
		Capabilities: map[testsuites.Capability]bool{
			testsuites.CapPersistence: true,
			testsuites.CapExec:        true,
			testsuites.CapMultiPODs:   true,
			testsuites.CapRWX:         true,
		},
	}
}

func (e *fsxDriver) SkipUnsupportedTest(testpatterns.TestPattern) {}

func (e *fsxDriver) PrepareTest(f *framework.Framework) (*testsuites.PerTestConfig, func()) {
	By("PrepareTest")
	cancelPodLogs := testsuites.StartPodLogs(f)

	return &testsuites.PerTestConfig{
			Driver:    e,
			Prefix:    "fsx",
			Framework: f,
		}, func() {
			By("Cleaning up FSx CSI driver")
			cancelPodLogs()
		}
}

func (e *fsxDriver) CreateVolume(config *testsuites.PerTestConfig, volType testpatterns.TestVolType) testsuites.TestVolume {
	c := NewCloud(*region)
	instance, err := c.getNodeInstance(*clusterName)
	if err != nil {
		Fail(fmt.Sprintf("failed to get node instance %v", err))
	}
	securityGroupIds := getSecurityGroupIds(instance)
	subnetId := *instance.SubnetId

	ctx := context.Background()
	options := &fsx.FileSystemOptions{
		CapacityGiB:      3600,
		SubnetId:         subnetId,
		SecurityGroupIds: securityGroupIds,
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

func (e *fsxDriver) GetPersistentVolumeSource(readOnly bool, fsType string, volume testsuites.TestVolume) (*v1.PersistentVolumeSource, *v1.VolumeNodeAffinity) {
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

func (e *fsxDriver) GetDynamicProvisionStorageClass(config *testsuites.PerTestConfig, fsType string) *storagev1.StorageClass {
	c := NewCloud(*region)
	instance, err := c.getNodeInstance(*clusterName)
	if err != nil {
		Fail(fmt.Sprintf("failed to get node instance %v", err))
	}
	securityGroupIds := getSecurityGroupIds(instance)
	subnetId := *instance.SubnetId

	provisioner := e.driverName
	parameters := map[string]string{
		"subnetId":         subnetId,
		"securityGroupIds": strings.Join(securityGroupIds, ","),
	}

	ns := config.Framework.Namespace.Name
	suffix := fmt.Sprintf("%s-sc", e.driverName)

	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind: "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Name must be unique, so let's base it on namespace name and use GenerateName
			Name: names.SimpleNameGenerator.GenerateName(ns + "-" + suffix),
		},
		Provisioner: provisioner,
		Parameters:  parameters,
	}
}

//func (e *fsxDriver) GetClaimSize() string {
//	return "3600Gi"
//}

// List of testSuites to be executed in below loop
var csiTestSuites = []func() testsuites.TestSuite{
	testsuites.InitVolumesTestSuite,
	testsuites.InitVolumeIOTestSuite,
	testsuites.InitVolumeModeTestSuite,
	testsuites.InitSubPathTestSuite,
	testsuites.InitProvisioningTestSuite,
	//testsuites.InitSnapshottableTestSuite,
	testsuites.InitMultiVolumeTestSuite,
}

var _ = Describe("FSx CSI Driver Conformance", func() {
	driver := InitFSxCSIDriver()
	Context(testsuites.GetDriverNameWithFeatureTags(driver), func() {
		testsuites.DefineTestSuite(driver, csiTestSuites)
	})
})
