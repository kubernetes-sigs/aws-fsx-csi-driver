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
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/fsx"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/mocks"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/util"
)

func TestCreateVolume(t *testing.T) {

	var (
		endpoint               = "endpoint"
		volumeName             = "volumeName"
		fileSystemId           = "fs-1234"
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
		extraTags = "key1=value1,key2=value2"
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
						volumeParamsSubnetId:                   subnetId,
						volumeParamsSecurityGroupIds:           securityGroupIds,
						volumeParamsDeploymentType:             fsx.LustreDeploymentTypeScratch2,
						volumeParamsStorageType:                fsx.StorageTypeSsd,
						volumeParamsWeeklyMaintenanceStartTime: "7:08:00",
						volumeParamsFileSystemTypeVersion:      "2.12",
						volumeParamsExtraTags:                  extraTags,
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
						volumeParamsDataCompressionType:           "LZ4",
						volumeParamsWeeklyMaintenanceStartTime:    "7:08:00",
						volumeParamsFileSystemTypeVersion:         "2.12",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDeleteVolume(t *testing.T) {
	var (
		endpoint     = "endpoint"
		fileSystemId = "fs-1234"
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

func TestExpandVolume(t *testing.T) {
	var (
		endpoint             = "endpoint"
		fileSystemId         = "fs-1234"
		initialSizeGiB int64 = 1200
		finalSizeGiB   int64 = 2400
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
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
						LimitBytes:    util.GiBToBytes(3600),
					},
				}

				initialFs := &cloud.FileSystem{
					FileSystemId:             fileSystemId,
					CapacityGiB:              initialSizeGiB,
					DnsName:                  "test.us-east-1.fsx.amazonaws.com",
					MountName:                "random",
					StorageType:              "SSD",
					DeploymentType:           "SCRATCH_2",
					PerUnitStorageThroughput: 0,
				}

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(initialFs, nil)
				mockCloud.EXPECT().ResizeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId), gomock.Eq(finalSizeGiB)).Return(finalSizeGiB, nil)
				mockCloud.EXPECT().WaitForFileSystemResize(gomock.Eq(ctx), gomock.Eq(fileSystemId), gomock.Eq(finalSizeGiB)).Return(nil)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err != nil {
					t.Fatalf("ControllerExpandVolume failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: requested capacity does not exceed current capacity",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(initialSizeGiB),
						LimitBytes:    util.GiBToBytes(3600),
					},
				}

				initialFs := &cloud.FileSystem{
					FileSystemId:             fileSystemId,
					CapacityGiB:              initialSizeGiB,
					DnsName:                  "test.us-east-1.fsx.amazonaws.com",
					MountName:                "random",
					StorageType:              "SSD",
					DeploymentType:           "SCRATCH_2",
					PerUnitStorageThroughput: 0,
				}

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(initialFs, nil)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err != nil {
					t.Fatalf("ControllerExpandVolume failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: volume ID not provided",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
					},
				}
				expandError := status.Error(codes.InvalidArgument, "Volume ID not provided")

				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: capacity range not provided",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
				}
				expandError := status.Error(codes.InvalidArgument, "Capacity range not provided")

				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: DescribeFileSystems not successful, filesystem not found",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
					},
				}
				expandError := status.Errorf(codes.NotFound, "Filesystem not found: %v", cloud.ErrNotFound)

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil, cloud.ErrNotFound)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: DescribeFileSystems not successful, other error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
					},
				}
				expandError := status.Errorf(codes.Internal, "DescribeFileSystem failed: %v", cloud.ErrMultiFileSystems)

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(nil, cloud.ErrMultiFileSystems)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: capacity limit exceeds required bytes not successful",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
						LimitBytes:    util.GiBToBytes(2000),
					},
				}
				expandError := status.Errorf(codes.OutOfRange, "Requested storage capacity of %d bytes exceeds capacity limit of %d bytes.",
					util.GiBToBytes(finalSizeGiB),
					expandRequest.GetCapacityRange().GetLimitBytes())

				initialFs := &cloud.FileSystem{
					FileSystemId:             fileSystemId,
					CapacityGiB:              initialSizeGiB,
					DnsName:                  "test.us-east-1.fsx.amazonaws.com",
					MountName:                "random",
					StorageType:              "SSD",
					DeploymentType:           "SCRATCH_2",
					PerUnitStorageThroughput: 0,
				}

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(initialFs, nil)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: ResizeFileSystem failed",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
					},
				}
				resizeError := fmt.Errorf("UpdateFileSystem failed: %v", awserr.New(fsx.ErrCodeBadRequest, "test", nil))
				expandError := status.Errorf(codes.Internal, "resize failed: %v", resizeError)

				initialFs := &cloud.FileSystem{
					FileSystemId:             fileSystemId,
					CapacityGiB:              initialSizeGiB,
					DnsName:                  "test.us-east-1.fsx.amazonaws.com",
					MountName:                "random",
					StorageType:              "SSD",
					DeploymentType:           "SCRATCH_2",
					PerUnitStorageThroughput: 0,
				}

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(initialFs, nil)
				mockCloud.EXPECT().ResizeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId), gomock.Eq(finalSizeGiB)).Return(initialSizeGiB, resizeError)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: WaitForFileSystemResize failed",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockCloud := mocks.NewMockCloud(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					cloud:    mockCloud,
				}

				ctx := context.Background()
				expandRequest := &csi.ControllerExpandVolumeRequest{
					VolumeId: fileSystemId,
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: util.GiBToBytes(1440),
					},
				}
				waitError := fmt.Errorf("update failed for filesystem %s: test", fileSystemId)
				expandError := status.Errorf(codes.Internal, "filesystem is not resized: %v", waitError)

				initialFs := &cloud.FileSystem{
					FileSystemId:             fileSystemId,
					CapacityGiB:              initialSizeGiB,
					DnsName:                  "test.us-east-1.fsx.amazonaws.com",
					MountName:                "random",
					StorageType:              "SSD",
					DeploymentType:           "SCRATCH_2",
					PerUnitStorageThroughput: 0,
				}

				mockCloud.EXPECT().DescribeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId)).Return(initialFs, nil)
				mockCloud.EXPECT().ResizeFileSystem(gomock.Eq(ctx), gomock.Eq(fileSystemId), gomock.Eq(finalSizeGiB)).Return(finalSizeGiB, nil)
				mockCloud.EXPECT().WaitForFileSystemResize(gomock.Eq(ctx), gomock.Eq(fileSystemId), gomock.Eq(finalSizeGiB)).Return(waitError)
				_, err := driver.ControllerExpandVolume(ctx, expandRequest)
				if err == nil {
					t.Fatalf("ControllerExpandVolume did not return error, expected [%v]", expandError)
				}
				if err.Error() != expandError.Error() {
					t.Fatalf("ControllerExpandVolume returned error [%v], expected [%v]", err, expandError)
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
			name: "fail: volume capability is missing",
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
