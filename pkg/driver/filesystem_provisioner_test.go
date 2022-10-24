package driver

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/fsx"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/mocks"
)

func TestCreateVolumeByFileSystemProvisioner(t *testing.T) {
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
			name: "success: normal with kubernetes external provisioner's parameter key",
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
						volumeParamsSubnetId:                           subnetId,
						volumeParamsSecurityGroupIds:                   securityGroupIds,
						kubernetesExternalProvisionerKeyPrefix + "any": "any",
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
			name: "fail: invalid parameter is passed",
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
						"invalidParam":               "invalid",
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

func TestDeleteVolumeByFileSystemProvisioner(t *testing.T) {
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
