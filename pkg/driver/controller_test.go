/*
Copyright 2019 The Kubernetes Authors.

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

package driver

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/fsx"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/driver/mocks"
)

func TestCreateVolume(t *testing.T) {

	var (
		endpoint               = "endpoint"
		volumeName             = "volumeName"
		fileSystemId           = "fs-1234"
		sharedVolumeId         = "shared/fs-1234/volumeName"
		volumeSizeGiB    int64 = 1200
		subnetId               = "subnet-056da83524edbe641"
		securityGroupIds       = "sg-086f61ea73388fb6b,sg-0145e55e976000c9e"
		dnsName                = "test.fsx.us-west-2.amazoawd.com"
		mountName              = "random"
		stdVolCap              = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		}
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:         subnetId,
						volumeParamsSecurityGroupIds: securityGroupIds,
					},
				}

				ctx := context.Background()
				fs := &cloud.FileSystem{
					FileSystemId: fileSystemId,
					CapacityGiB:  volumeSizeGiB,
					DnsName:      dnsName,
					MountName:    mountName,
				}
				mockCloud.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Eq(volumeName), gomock.Any()).Return(fs, nil)
				mockCloud.EXPECT().WaitForFileSystemAvailable(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if resp.Volume.VolumeId != fileSystemId {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, fileSystemId)
				}

				if resp.Volume.CapacityBytes == 0 {
					t.Fatalf("resp.Volume.CapacityGiB is zero")
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with deploymentType SCRATCH_2",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:         subnetId,
						volumeParamsSecurityGroupIds: securityGroupIds,
						volumeParamsDeploymentType:   fsx.LustreDeploymentTypeScratch2,
						volumeParamsStorageType:      fsx.StorageTypeSsd,
					},
				}

				ctx := context.Background()
				fs := &cloud.FileSystem{
					FileSystemId: fileSystemId,
					CapacityGiB:  volumeSizeGiB,
					DnsName:      dnsName,
					MountName:    mountName,
				}
				mockCloud.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Eq(volumeName), gomock.Any()).Return(fs, nil)
				mockCloud.EXPECT().WaitForFileSystemAvailable(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if resp.Volume.VolumeId != fileSystemId {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, fileSystemId)
				}

				if resp.Volume.CapacityBytes == 0 {
					t.Fatalf("resp.Volume.CapacityGiB is zero")
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with deploymentType PERSISTENT_1",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:                      subnetId,
						volumeParamsSecurityGroupIds:              securityGroupIds,
						volumeParamsDeploymentType:                fsx.LustreDeploymentTypePersistent1,
						volumeParamsKmsKeyId:                      "arn:aws:kms:us-east-1:215474938041:key/48313a27-7d88-4b51-98a4-fdf5bc80dbbe",
						volumeParamsPerUnitStorageThroughput:      "200",
						volumeParamsStorageType:                   fsx.StorageTypeSsd,
						volumeParamsAutomaticBackupRetentionDays:  "1",
						volumeParamsDailyAutomaticBackupStartTime: "00:00",
						volumeParamsCopyTagsToBackups:             "true",
					},
				}

				ctx := context.Background()
				fs := &cloud.FileSystem{
					FileSystemId: fileSystemId,
					CapacityGiB:  volumeSizeGiB,
					DnsName:      dnsName,
					MountName:    mountName,
				}
				mockCloud.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Eq(volumeName), gomock.Any()).Return(fs, nil)
				mockCloud.EXPECT().WaitForFileSystemAvailable(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if resp.Volume.VolumeId != fileSystemId {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, fileSystemId)
				}

				if resp.Volume.CapacityBytes == 0 {
					t.Fatalf("resp.Volume.CapacityGiB is zero")
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: existing fs",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsFileSystemId: fileSystemId,
					},
				}

				ctx := context.Background()
				fs := &cloud.FileSystem{
					FileSystemId: fileSystemId,
					CapacityGiB:  volumeSizeGiB,
					DnsName:      dnsName,
					MountName:    mountName,
				}
				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(fs, nil)
				mockCloud.EXPECT().WaitForFileSystemAvailable(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if resp.Volume.VolumeId != sharedVolumeId {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, fileSystemId)
				}

				if resp.Volume.CapacityBytes == 0 {
					t.Fatalf("resp.Volume.CapacityGiB is zero")
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				subPath := "volumeName"
				subpath, exists := resp.Volume.VolumeContext[volumeContextSubPath]
				if !exists {
					t.Fatal("subpath is missing")
				}

				if subpath != subPath {
					t.Fatalf("subpath mismatches. actual: %v expected: %v", subpath, subPath)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: volume name missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:         subnetId,
						volumeParamsSecurityGroupIds: securityGroupIds,
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: perUnitStorageThroughput has to be a integer",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:                 subnetId,
						volumeParamsSecurityGroupIds:         securityGroupIds,
						volumeParamsPerUnitStorageThroughput: "notInteger",
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: volume capacity missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					Parameters: map[string]string{
						volumeParamsSubnetId:         subnetId,
						volumeParamsSecurityGroupIds: securityGroupIds,
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: CreateFileSystem return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsSubnetId:         subnetId,
						volumeParamsSecurityGroupIds: securityGroupIds,
					},
				}

				ctx := context.Background()
				mockCloud.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Eq(volumeName), gomock.Any()).Return(nil, cloud.ErrFsExistsDiffSize)

				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DescribeFileSystem return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.CreateVolumeRequest{
					Name: volumeName,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsFileSystemId: fileSystemId,
					},
				}

				ctx := context.Background()
				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil, cloud.ErrFsExistsDiffSize)

				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDeleteVolume(t *testing.T) {
	var (
		endpoint       = "endpoint"
		fileSystemId   = "fs-1234"
		sharedVolumeId = "shared/fs-1234/volumeName"
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.DeleteVolumeRequest{
					VolumeId: fileSystemId,
				}

				ctx := context.Background()

				mockCloud.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil)
				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: existing filesystem",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.DeleteVolumeRequest{
					VolumeId: sharedVolumeId,
				}

				ctx := context.Background()

				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: volume ID is missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.DeleteVolumeRequest{}

				ctx := context.Background()
				_, err := driver.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatal("DeleteVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: DeleteFileSystem returns ErrNotFound",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.DeleteVolumeRequest{
					VolumeId: fileSystemId,
				}

				ctx := context.Background()
				mockCloud.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(cloud.ErrNotFound)
				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DeleteFileSystem returns other error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				req := &csi.DeleteVolumeRequest{
					VolumeId: fileSystemId,
				}

				ctx := context.Background()
				mockCloud.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(errors.New("DeleteFileSystem failed"))
				_, err := driver.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatal("DeleteVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerGetCapabilities(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockCloud := mocks.NewMockCloud(mockCtl)
	endpoint := "endpoint"

	driver := &Driver{
		endpoint: endpoint,
		cloud:    mockCloud,
	}

	ctx := context.Background()
	_, err := driver.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("ControllerGetCapabilities is failed: %v", err)
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {

	var (
		endpoint     = "endpoint"
		fileSystemId = "fs-12345"
		stdVolCap    = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		}
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()

				fs := &cloud.FileSystem{}
				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(fs, nil)

				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId: fileSystemId,
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
				}

				resp, err := driver.ValidateVolumeCapabilities(ctx, req)
				if err != nil {
					t.Fatalf("ControllerGetCapabilities is failed: %v", err)
				}
				if resp.Confirmed == nil {
					t.Fatal("capability is not supported")
				}
				mockCtl.Finish()
			},
		},
		{
			name: "fail: volume ID is missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
				}

				_, err := driver.ValidateVolumeCapabilities(ctx, req)
				if err == nil {
					t.Fatal("ControllerGetCapabilities is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: volueme capability is missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId: fileSystemId,
				}

				_, err := driver.ValidateVolumeCapabilities(ctx, req)
				if err == nil {
					t.Fatal("ControllerGetCapabilities is not failed")
				}

				mockCtl.Finish()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
