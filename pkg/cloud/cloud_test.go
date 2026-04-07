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

package cloud

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud/mocks"
)

func TestCreateFileSystem(t *testing.T) {
	var (
		volumeName                          = "volumeName"
		fileSystemId                        = "fs-1234"
		volumeSizeGiB                 int32 = 1200
		p2VolumeSizeGiB               int32 = 4800
		subnetId                            = "subnet-056da83524edbe641"
		securityGroupIds                    = []string{"sg-086f61ea73388fb6b", "sg-0145e55e976000c9e"}
		dnsname                             = "test.fsx.us-west-2.amazoawd.com"
		autoImportPolicy                    = types.AutoImportPolicyTypeNewChanged
		s3ImportPath                        = "s3://fsx-s3-data-repository"
		s3ExportPath                        = "s3://fsx-s3-data-repository/export"
		deploymentType                      = types.LustreDeploymentTypeScratch2
		mountName                           = "fsx"
		kmsKeyId                            = "arn:aws:kms:us-east-1:215474938041:key/48313a27-7d88-4b51-98a4-fdf5bc80dbbe"
		perUnitStorageThroughput      int32 = 200
		p2PerUnitStorageThroughput    int32 = 1000
		DailyAutomaticBackupStartTime       = "00:00:00"
		AutomaticBackupRetentionDays  int32 = 1
		CopyTagsToBackups                   = true
		efaEnabled                          = true
		metadataModeAutomatic               = "AUTOMATIC"
		metadataModeUserProvisioned         = "USER_PROVISIONED"
		metadataIops                  int32 = 6000
		dataCompressionTypeNone             = types.DataCompressionTypeNone
		dataCompressionTypeLZ4              = types.DataCompressionTypeLz4
		weeklyMaintenanceStartTime          = "7:09:00"
		fileSystemTypeVersion               = "2.12"
		fileSystemTypeVersion2_15           = "2.15"
		extraTags                           = []string{"key1=value1", "key2=value2"}
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal without deploymentType",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:                volumeSizeGiB,
					SubnetId:                   subnetId,
					SecurityGroupIds:           securityGroupIds,
					FileSystemTypeVersion:      fileSystemTypeVersion,
					WeeklyMaintenanceStartTime: weeklyMaintenanceStartTime,
					ExtraTags:                  extraTags,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:          aws.String(fileSystemId),
						FileSystemTypeVersion: aws.String(fileSystemTypeVersion),
						StorageCapacity:       aws.Int32(volumeSizeGiB),
						StorageType:           types.StorageTypeSsd,
						DNSName:               aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:             types.LustreDeploymentTypeScratch1,
							MountName:                  aws.String(mountName),
							WeeklyMaintenanceStartTime: aws.String(weeklyMaintenanceStartTime),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with deploymentType",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   string(deploymentType),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int32(volumeSizeGiB),
						StorageType:     types.StorageTypeSsd,
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType: deploymentType,
							MountName:      aws.String(mountName),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with deploymentType and storageTypeSsd",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   string(deploymentType),
					StorageType:      string(types.StorageTypeSsd),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int32(volumeSizeGiB),
						StorageType:     types.StorageTypeSsd,
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType: deploymentType,
							MountName:      aws.String(mountName),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with deploymentType and storageTypeHdd",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   string(types.LustreDeploymentTypePersistent1),
					StorageType:      string(types.StorageTypeHdd),
					DriveCacheType:   string(types.DriveCacheTypeNone),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int32(volumeSizeGiB),
						StorageType:     types.StorageTypeHdd,
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType: types.LustreDeploymentTypePersistent1,
							MountName:      aws.String(mountName),
							DriveCacheType: types.DriveCacheTypeNone,
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: incompatible deploymentType and storageTypeHdd",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   string(deploymentType),
					StorageType:      string(types.StorageTypeHdd),
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatalf("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: S3 data repository",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					AutoImportPolicy: string(autoImportPolicy),
					S3ImportPath:     s3ImportPath,
					S3ExportPath:     s3ExportPath,
				}

				dataRepositoryConfiguration := &types.DataRepositoryConfiguration{}
				dataRepositoryConfiguration.AutoImportPolicy = types.AutoImportPolicyType(autoImportPolicy)
				dataRepositoryConfiguration.ImportPath = aws.String(s3ImportPath)
				dataRepositoryConfiguration.ExportPath = aws.String(s3ExportPath)

				lustreFileSystemConfiguration := &types.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					DeploymentType:              deploymentType,
					MountName:                   aws.String(mountName),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:        aws.String(fileSystemId),
						StorageCapacity:     aws.Int32(volumeSizeGiB),
						StorageType:         types.StorageTypeSsd,
						DNSName:             aws.String(dnsname),
						LustreConfiguration: lustreFileSystemConfiguration,
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: invalid import and export path config - only s3ExportPath",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					S3ExportPath:     s3ExportPath,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatalf("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: invalid import and export path config - different bucket",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					AutoImportPolicy: string(autoImportPolicy),
					S3ImportPath:     "s3://bucket1/import",
					S3ExportPath:     "s3://bucket2/export",
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatalf("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: the kmsKeyId can can only be specified for PERSISTENT_1 deployment type",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   string(deploymentType),
					KmsKeyId:         kmsKeyId,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: the perUnitStorageThroughput can only be specified for PERSISTENT_1 deployment type",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:              volumeSizeGiB,
					SubnetId:                 subnetId,
					SecurityGroupIds:         securityGroupIds,
					DeploymentType:           string(deploymentType),
					PerUnitStorageThroughput: perUnitStorageThroughput,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing subnet ID",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SecurityGroupIds: securityGroupIds,
				}

				ctx := context.Background()
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: CreateFileSystem return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: Create PERSISTENT file system with scheduled backup",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:                   volumeSizeGiB,
					SubnetId:                      subnetId,
					SecurityGroupIds:              securityGroupIds,
					AutomaticBackupRetentionDays:  AutomaticBackupRetentionDays,
					DailyAutomaticBackupStartTime: DailyAutomaticBackupStartTime,
					CopyTagsToBackups:             CopyTagsToBackups,
					DeploymentType:                string(types.LustreDeploymentTypePersistent1),
				}

				lustreFileSystemConfiguration := &types.LustreFileSystemConfiguration{
					DeploymentType:                types.LustreDeploymentTypePersistent1,
					MountName:                     aws.String(mountName),
					AutomaticBackupRetentionDays:  aws.Int32(AutomaticBackupRetentionDays),
					DailyAutomaticBackupStartTime: aws.String(DailyAutomaticBackupStartTime),
					CopyTagsToBackups:             aws.Bool(CopyTagsToBackups),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:        aws.String(fileSystemId),
						StorageCapacity:     aws.Int32(volumeSizeGiB),
						StorageType:         types.StorageTypeSsd,
						DNSName:             aws.String(dnsname),
						LustreConfiguration: lustreFileSystemConfiguration,
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with NONE DataCompressionType",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: string(dataCompressionTypeNone),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int32(volumeSizeGiB),
						StorageType:     types.StorageTypeSsd,
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:      types.LustreDeploymentTypeScratch1,
							MountName:           aws.String(mountName),
							DataCompressionType: types.DataCompressionTypeNone,
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with LZ4 DataCompressionType",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: string(dataCompressionTypeLZ4),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int32(volumeSizeGiB),
						StorageType:     types.StorageTypeSsd,
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:      types.LustreDeploymentTypeScratch1,
							MountName:           aws.String(mountName),
							DataCompressionType: types.DataCompressionTypeLz4,
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: invalid DataCompressionType",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: "ZFS",
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: metadataConfigurationMode AUTOMATIC",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:               p2VolumeSizeGiB,
					SubnetId:                  subnetId,
					SecurityGroupIds:          securityGroupIds,
					DeploymentType:            string(types.LustreDeploymentTypePersistent2),
					FileSystemTypeVersion:     fileSystemTypeVersion2_15,
					PerUnitStorageThroughput:  p2PerUnitStorageThroughput,
					MetadataConfigurationMode: metadataModeAutomatic,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:          aws.String(fileSystemId),
						StorageCapacity:       aws.Int32(p2VolumeSizeGiB),
						StorageType:           types.StorageTypeSsd,
						DNSName:               aws.String(dnsname),
						FileSystemTypeVersion: aws.String(fileSystemTypeVersion2_15),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:           types.LustreDeploymentTypePersistent2,
							PerUnitStorageThroughput: aws.Int32(p2PerUnitStorageThroughput),
							MountName:                aws.String(mountName),
							MetadataConfiguration: &types.FileSystemLustreMetadataConfiguration{
								Mode: types.MetadataConfigurationModeAutomatic,
							},
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != p2VolumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, p2VolumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: metadataConfigurationMode USER_PROVISIONED",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:               p2VolumeSizeGiB,
					SubnetId:                  subnetId,
					SecurityGroupIds:          securityGroupIds,
					DeploymentType:            string(types.LustreDeploymentTypePersistent2),
					FileSystemTypeVersion:     fileSystemTypeVersion2_15,
					PerUnitStorageThroughput:  p2PerUnitStorageThroughput,
					MetadataConfigurationMode: metadataModeUserProvisioned,
					MetadataIops:              metadataIops,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:          aws.String(fileSystemId),
						StorageCapacity:       aws.Int32(p2VolumeSizeGiB),
						StorageType:           types.StorageTypeSsd,
						DNSName:               aws.String(dnsname),
						FileSystemTypeVersion: aws.String(fileSystemTypeVersion2_15),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:           types.LustreDeploymentTypePersistent2,
							MountName:                aws.String(mountName),
							PerUnitStorageThroughput: aws.Int32(p2PerUnitStorageThroughput),
							MetadataConfiguration: &types.FileSystemLustreMetadataConfiguration{
								Mode: types.MetadataConfigurationModeUserProvisioned,
								Iops: aws.Int32(metadataIops),
							},
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != p2VolumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, p2VolumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: efaEnabled true",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:               p2VolumeSizeGiB,
					SubnetId:                  subnetId,
					SecurityGroupIds:          securityGroupIds,
					DeploymentType:            string(types.LustreDeploymentTypePersistent2),
					FileSystemTypeVersion:     fileSystemTypeVersion2_15,
					PerUnitStorageThroughput:  p2PerUnitStorageThroughput,
					EfaEnabled:                efaEnabled,
					MetadataConfigurationMode: metadataModeAutomatic,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId:          aws.String(fileSystemId),
						StorageCapacity:       aws.Int32(p2VolumeSizeGiB),
						StorageType:           types.StorageTypeSsd,
						DNSName:               aws.String(dnsname),
						FileSystemTypeVersion: aws.String(fileSystemTypeVersion2_15),
						LustreConfiguration: &types.LustreFileSystemConfiguration{
							DeploymentType:           types.LustreDeploymentTypePersistent2,
							MountName:                aws.String(mountName),
							PerUnitStorageThroughput: aws.Int32(p2PerUnitStorageThroughput),
							EfaEnabled:               aws.Bool(efaEnabled),
							MetadataConfiguration: &types.FileSystemLustreMetadataConfiguration{
								Mode: types.MetadataConfigurationModeAutomatic,
							},
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileSystem(ctx, volumeName, req)
				if err != nil {
					t.Fatalf("CreateFileSystem is failed: %v", err)
				}

				if resp == nil {
					t.Fatal("resp is nil")
				}

				if resp.FileSystemId != fileSystemId {
					t.Fatalf("FileSystemId mismatches. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
				}

				if resp.CapacityGiB != p2VolumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, p2VolumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.MountName != mountName {
					t.Fatalf("MountName mismatches. actual: %v expected: %v", resp.MountName, mountName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: invalid metadataConfiguration no iops provided",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				req := &FileSystemOptions{
					CapacityGiB:               p2VolumeSizeGiB,
					SubnetId:                  subnetId,
					SecurityGroupIds:          securityGroupIds,
					DeploymentType:            string(types.LustreDeploymentTypePersistent2),
					FileSystemTypeVersion:     fileSystemTypeVersion2_15,
					PerUnitStorageThroughput:  p2PerUnitStorageThroughput,
					MetadataConfigurationMode: metadataModeUserProvisioned,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystem failed"))
				_, err := c.CreateFileSystem(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDeleteFileSystem(t *testing.T) {
	var (
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
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				ctx := context.Background()
				
				// Mock DescribeFileSystems to return a non-INTELLIGENT_TIERING filesystem without skipFinalBackup tag
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId: aws.String(fileSystemId),
							StorageType:  types.StorageTypeSsd,
							DNSName:      aws.String("test.fsx.us-west-2.amazonaws.com"),
							Lifecycle:    types.FileSystemLifecycleAvailable,
							Tags:         []types.Tag{},
						},
					},
				}
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(describeOutput, nil)
				
				output := &fsx.DeleteFileSystemOutput{}
				mockFSx.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Any()).DoAndReturn(
					func(ctx context.Context, input *fsx.DeleteFileSystemInput, opts ...func(*fsx.Options)) (*fsx.DeleteFileSystemOutput, error) {
						// Should not set LustreConfiguration when skipFinalBackup is false
						if input.LustreConfiguration != nil {
							t.Error("LustreConfiguration should not be set when skipFinalBackup is false")
						}
						return output, nil
					},
				).Return(output, nil)
				err := c.DeleteFileSystem(ctx, fileSystemId)
				if err != nil {
					t.Fatalf("DeleteFileSystem is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DeleteFileSystemWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				ctx := context.Background()
				
				// Mock DescribeFileSystems to return a non-INTELLIGENT_TIERING filesystem
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId: aws.String(fileSystemId),
							StorageType:  types.StorageTypeSsd,
							DNSName:      aws.String("test.fsx.us-west-2.amazonaws.com"),
							Lifecycle:    types.FileSystemLifecycleAvailable,
							Tags:         []types.Tag{},
						},
					},
				}
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(describeOutput, nil)
				
				mockFSx.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DeleteFileSystemWithContext failed"))
				err := c.DeleteFileSystem(ctx, fileSystemId)
				if err == nil {
					t.Fatal("DeleteFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: skipFinalBackup from tag",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				ctx := context.Background()
				
				// Mock DescribeFileSystems to return filesystem with skipFinalBackup tag set to true
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId: aws.String(fileSystemId),
							StorageType:  types.StorageTypeIntelligentTiering,
							DNSName:      aws.String("test.fsx.us-west-2.amazonaws.com"),
							Lifecycle:    types.FileSystemLifecycleAvailable,
							Tags: []types.Tag{
								{
									Key:   aws.String(SkipFinalBackupTagKey),
									Value: aws.String("true"),
								},
							},
						},
					},
				}
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(describeOutput, nil)
				
				// Mock DeleteFileSystem and verify SkipFinalBackup is set based on tag
				deleteOutput := &fsx.DeleteFileSystemOutput{}
				mockFSx.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Any()).DoAndReturn(
					func(ctx context.Context, input *fsx.DeleteFileSystemInput, opts ...func(*fsx.Options)) (*fsx.DeleteFileSystemOutput, error) {
						if input.LustreConfiguration == nil {
							t.Error("LustreConfiguration should be set when tag indicates skipFinalBackup")
						} else if input.LustreConfiguration.SkipFinalBackup == nil || !*input.LustreConfiguration.SkipFinalBackup {
							t.Error("SkipFinalBackup should be true when tag is set to true")
						}
						return deleteOutput, nil
					},
				).Return(deleteOutput, nil)
				
				err := c.DeleteFileSystem(ctx, fileSystemId)
				if err != nil {
					t.Fatalf("DeleteFileSystem failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: DescribeFileSystems fails but deletion proceeds",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx:         mockFSx,
					volumeCache: make(map[string]*FileSystem),
				}

				ctx := context.Background()
				
				// Mock DescribeFileSystems to fail
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DescribeFileSystems failed"))
				
				// Mock DeleteFileSystem - should still be called without SkipFinalBackup
				deleteOutput := &fsx.DeleteFileSystemOutput{}
				mockFSx.EXPECT().DeleteFileSystem(gomock.Eq(ctx), gomock.Any()).DoAndReturn(
					func(ctx context.Context, input *fsx.DeleteFileSystemInput, opts ...func(*fsx.Options)) (*fsx.DeleteFileSystemOutput, error) {
						// When describe fails, we should not set LustreConfiguration
						if input.LustreConfiguration != nil {
							t.Error("LustreConfiguration should not be set when DescribeFileSystems fails")
						}
						return deleteOutput, nil
					},
				).Return(deleteOutput, nil)
				
				err := c.DeleteFileSystem(ctx, fileSystemId)
				if err != nil {
					t.Fatalf("DeleteFileSystem failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDescribeFileSystem(t *testing.T) {
	var (
		fileSystemId           = "fs-1234"
		volumeSizeGiB    int32 = 1200
		dnsname                = "test.fsx.us-west-2.amazoawd.com"
		autoImportPolicy       = types.AutoImportPolicyTypeNewChanged
		s3ImportPath           = "s3://fsx-s3-data-repository"
		s3ExportPath           = "s3://fsx-s3-data-repository/export"
		mountName              = "fsx"
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				output := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(volumeSizeGiB),
							StorageType:     types.StorageTypeSsd,
							DNSName:         aws.String(dnsname),
							LustreConfiguration: &types.LustreFileSystemConfiguration{
								DeploymentType: types.LustreDeploymentTypeScratch1,
								MountName:      aws.String(mountName),
							},
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				_, err := c.DescribeFileSystem(ctx, fileSystemId)
				if err != nil {
					t.Fatalf("DeleteFileSystem is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: S3 data repository",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				dataRepositoryConfiguration := &types.DataRepositoryConfiguration{}
				dataRepositoryConfiguration.AutoImportPolicy = autoImportPolicy
				dataRepositoryConfiguration.ImportPath = aws.String(s3ImportPath)
				dataRepositoryConfiguration.ExportPath = aws.String(s3ExportPath)

				lustreFileSystemConfiguration := &types.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					DeploymentType:              types.LustreDeploymentTypeScratch1,
					MountName:                   aws.String(mountName),
				}

				output := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:        aws.String(fileSystemId),
							StorageCapacity:     aws.Int32(volumeSizeGiB),
							StorageType:         types.StorageTypeSsd,
							DNSName:             aws.String(dnsname),
							LustreConfiguration: lustreFileSystemConfiguration,
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				_, err := c.DescribeFileSystem(ctx, fileSystemId)
				if err != nil {
					t.Fatalf("DeleteFileSystem is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DescribeFileSystemWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DescribeFileSystems failed"))
				_, err := c.DescribeFileSystem(ctx, fileSystemId)
				if err == nil {
					t.Fatal("DescribeFileSystem is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestResizeFileSystem(t *testing.T) {
	var (
		fileSystemId         = "fs-1234"
		initialSizeGiB int32 = 1200
		finalSizeGiB   int32 = 2400
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateOutput := &fsx.UpdateFileSystemOutput{
					FileSystem: &types.FileSystem{
						FileSystemId: aws.String(fileSystemId),
						AdministrativeActions: []types.AdministrativeAction{
							{
								AdministrativeActionType: types.AdministrativeActionTypeFileSystemUpdate,
								Status:                   types.StatusInProgress,
								TargetFileSystemValues: &types.FileSystem{
									StorageCapacity: aws.Int32(finalSizeGiB),
								},
							},
							{
								AdministrativeActionType: types.AdministrativeActionTypeStorageOptimization,
								Status:                   types.StatusPending,
							},
						},
					},
				}

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(updateOutput, nil)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err != nil {
					t.Fatalf("ResizeFileSystem is failed: %v", err)
				}
				if resp != finalSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, finalSizeGiB)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: matching update in progress",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
							AdministrativeActions: []types.AdministrativeAction{
								{
									AdministrativeActionType: types.AdministrativeActionTypeFileSystemUpdate,
									Status:                   types.StatusInProgress,
									TargetFileSystemValues: &types.FileSystem{
										StorageCapacity: aws.Int32(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: types.AdministrativeActionTypeStorageOptimization,
									Status:                   types.StatusPending,
								},
							},
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateError := &types.BadRequest{
					Message: aws.String("Unable to perform the storage capacity update. There is an update already in progress."),
				}

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err != nil {
					t.Fatalf("ResizeFileSystem is failed: %v", err)
				}
				if resp != finalSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, initialSizeGiB)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: first DescribeFileSystems not successful",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeError := &types.FileSystemNotFound{
					Message: aws.String("test"),
				}
				resizeError := fmt.Errorf("DescribeFileSystems failed: %v", describeError)

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(nil, describeError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("ResizeFileSystem did not return error, expected [%v]", resizeError)
				}
				if err.Error() != resizeError.Error() {
					t.Fatalf("ResizeFileSystem returned error [%v], expected [%v]", err, resizeError)
				}
				if resp != 0 {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, 0)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: UpdateFileSystem not successful, error not update in progress",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateError := &types.BadRequest{
					Message: aws.String("test"),
				}
				resizeError := fmt.Errorf("UpdateFileSystem failed: %v", updateError)

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("ResizeFileSystem did not return error, expected [%v]", resizeError)
				}
				if err.Error() != resizeError.Error() {
					t.Fatalf("ResizeFileSystem returned error [%v], expected [%v]", err, resizeError)
				}
				if resp != initialSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, initialSizeGiB)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: update in progress, second DescribeFileSystems not successful",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
						},
					},
				}
				describeError := &types.BadRequest{
					Message: aws.String("test"),
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateError := &types.BadRequest{
					Message: aws.String("Unable to perform the storage capacity update. There is an update already in progress."),
				}
				resizeError := fmt.Errorf("DescribeFileSystems failed: %v", describeError)

				gomock.InOrder(
					mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil),
					mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(nil, describeError),
				)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("ResizeFileSystem did not return error, expected [%v]", resizeError)
				}
				if err.Error() != resizeError.Error() {
					t.Fatalf("ResizeFileSystem returned error [%v], expected [%v]", err, resizeError)
				}
				if resp != initialSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, initialSizeGiB)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: update in progress, no updates found",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateError := &types.BadRequest{
					Message: aws.String("Unable to perform the storage capacity update. There is an update already in progress."),
				}
				resizeError := fmt.Errorf("there is no update on filesystem %s", fileSystemId)

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("ResizeFileSystem did not return error, expected [%v]", resizeError)
				}
				if err.Error() != resizeError.Error() {
					t.Fatalf("ResizeFileSystem returned error [%v], expected [%v]", err, resizeError)
				}
				if resp != initialSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, initialSizeGiB)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: update in progress, no matching updates found",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
							AdministrativeActions: []types.AdministrativeAction{
								{
									AdministrativeActionType: types.AdministrativeActionTypeFileSystemUpdate,
									Status:                   types.StatusInProgress,
									TargetFileSystemValues: &types.FileSystem{
										StorageCapacity: aws.Int32(4800),
									},
								},
								{
									AdministrativeActionType: types.AdministrativeActionTypeStorageOptimization,
									Status:                   types.StatusPending,
								},
							},
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int32(finalSizeGiB),
				}
				updateError := &types.BadRequest{
					Message: aws.String("Unable to perform the storage capacity update. There is an update already in progress."),
				}
				resizeError := fmt.Errorf("there is no update with storage capacity of %d GiB on filesystem %s", finalSizeGiB, fileSystemId)

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystem(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
				resp, err := c.ResizeFileSystem(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("ResizeFileSystem did not return error, expected [%v]", resizeError)
				}
				if err.Error() != resizeError.Error() {
					t.Fatalf("ResizeFileSystem returned error [%v], expected [%v]", err, resizeError)
				}
				if resp != initialSizeGiB {
					t.Fatalf("ResizeFileSystem returned %d GiB as resized storage capacity, expected %d GiB", resp, initialSizeGiB)
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestWaitForFileSystemResize(t *testing.T) {
	var (
		fileSystemId         = "fs-1234"
		initialSizeGiB int32 = 1200
		finalSizeGiB   int32 = 2400
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: update action completed, optimizing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(finalSizeGiB),
							AdministrativeActions: []types.AdministrativeAction{
								{
									AdministrativeActionType: types.AdministrativeActionTypeFileSystemUpdate,
									Status:                   types.StatusUpdatedOptimizing,
									TargetFileSystemValues: &types.FileSystem{
										StorageCapacity: aws.Int32(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: types.AdministrativeActionTypeStorageOptimization,
									Status:                   types.StatusInProgress,
								},
							},
						},
					},
				}

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				err := c.WaitForFileSystemResize(ctx, fileSystemId, finalSizeGiB)
				if err != nil {
					t.Fatalf("WaitForFileSystemResize is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "failure: update action failed",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []string{fileSystemId},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []types.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int32(initialSizeGiB),
							AdministrativeActions: []types.AdministrativeAction{
								{
									AdministrativeActionType: types.AdministrativeActionTypeFileSystemUpdate,
									Status:                   types.StatusFailed,
									FailureDetails: &types.AdministrativeActionFailureDetails{
										Message: aws.String("test"),
									},
									TargetFileSystemValues: &types.FileSystem{
										StorageCapacity: aws.Int32(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: types.AdministrativeActionTypeStorageOptimization,
									Status:                   types.StatusPending,
								},
							},
						},
					},
				}
				waitError := fmt.Errorf("update failed for filesystem %s: %q", fileSystemId, *describeOutput.FileSystems[0].AdministrativeActions[0].FailureDetails.Message)

				mockFSx.EXPECT().DescribeFileSystems(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				err := c.WaitForFileSystemResize(ctx, fileSystemId, finalSizeGiB)
				if err == nil {
					t.Fatalf("WaitForFileSystemResize did not return error, expected [%v]", err)
				}
				if err.Error() != waitError.Error() {
					t.Fatalf("WaitForFileSystemResize returned error [%v], expected [%v]", err, waitError)
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

// Feature: intelligent-tiering, Property 6: Storage Capacity Exclusion
// For any INTELLIGENT_TIERING filesystem creation request, the driver SHALL NOT include
// StorageCapacity in the AWS API request, regardless of whether a storage capacity was
// specified in the PVC.
// Validates: Requirements 1.3, 5.2
func TestCreateFileSystem_IntelligentTiering_StorageCapacityExclusion(t *testing.T) {
	var (
		volumeName       = "volumeName"
		fileSystemId     = "fs-1234"
		subnetId         = "subnet-056da83524edbe641"
		securityGroupIds = []string{"sg-086f61ea73388fb6b"}
		dnsname          = "test.fsx.us-west-2.amazoawd.com"
		mountName        = "fsx"
	)

	testCases := []struct {
		name              string
		capacityGiB       int32
		expectCapacitySet bool
	}{
		{
			name:              "INTELLIGENT_TIERING with zero capacity",
			capacityGiB:       0,
			expectCapacitySet: false,
		},
		{
			name:              "INTELLIGENT_TIERING with 1200 GiB capacity",
			capacityGiB:       1200,
			expectCapacitySet: false,
		},
		{
			name:              "INTELLIGENT_TIERING with 4800 GiB capacity",
			capacityGiB:       4800,
			expectCapacitySet: false,
		},
		{
			name:              "INTELLIGENT_TIERING with 10000 GiB capacity",
			capacityGiB:       10000,
			expectCapacitySet: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			mockFSx := mocks.NewMockFSx(mockCtl)
			c := &cloud{
				fsx:         mockFSx,
				volumeCache: make(map[string]*FileSystem),
			}

			req := &FileSystemOptions{
				CapacityGiB:            tc.capacityGiB,
				SubnetId:               subnetId,
				SecurityGroupIds:       securityGroupIds,
				StorageType:            "INTELLIGENT_TIERING",
				DeploymentType:         string(types.LustreDeploymentTypePersistent2),
				ThroughputCapacity:     4000,
				DataReadCacheSizingMode: "PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			}

			output := &fsx.CreateFileSystemOutput{
				FileSystem: &types.FileSystem{
					FileSystemId:    aws.String(fileSystemId),
					StorageType:     types.StorageTypeIntelligentTiering,
					StorageCapacity: aws.Int32(1200), // AWS returns a capacity even for INTELLIGENT_TIERING
					DNSName:         aws.String(dnsname),
					LustreConfiguration: &types.LustreFileSystemConfiguration{
						DeploymentType:     types.LustreDeploymentTypePersistent2,
						MountName:          aws.String(mountName),
						ThroughputCapacity: aws.Int32(4000),
					},
				},
			}

			ctx := context.Background()

			// Capture the input to verify StorageCapacity is not set
			mockFSx.EXPECT().CreateFileSystem(gomock.Eq(ctx), gomock.Any()).DoAndReturn(
				func(ctx context.Context, input *fsx.CreateFileSystemInput, opts ...func(*fsx.Options)) (*fsx.CreateFileSystemOutput, error) {
					// Property validation: StorageCapacity must NOT be set for INTELLIGENT_TIERING
					if input.StorageCapacity != nil {
						t.Errorf("StorageCapacity should not be set for INTELLIGENT_TIERING, but got: %d", *input.StorageCapacity)
					}

					// Verify INTELLIGENT_TIERING storage type is set
					if input.StorageType != types.StorageTypeIntelligentTiering {
						t.Errorf("StorageType should be INTELLIGENT_TIERING, but got: %v", input.StorageType)
					}

					return output, nil
				},
			)

			resp, err := c.CreateFileSystem(ctx, volumeName, req)
			if err != nil {
				t.Fatalf("CreateFileSystem failed: %v", err)
			}

			if resp == nil {
				t.Fatal("resp is nil")
			}

			if resp.FileSystemId != fileSystemId {
				t.Fatalf("FileSystemId mismatch. actual: %v expected: %v", resp.FileSystemId, fileSystemId)
			}
		})
	}
}

func TestIsBadRequestUpdateInProgress(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: BadRequest update in progress",
			testFunc: func(t *testing.T) {
				errorInput := &types.BadRequest{
					Message: aws.String("Unable to perform the storage capacity update. There is an update already in progress."),
				}
				if !isBadRequestUpdateInProgress(errorInput) {
					t.Fatalf("isBadRequestUpdateInProgress returned false, expected true")
				}
			},
		},
		{
			name: "failure: AWS error, different type",
			testFunc: func(t *testing.T) {
				errorInput := &types.FileSystemNotFound{
					Message: aws.String("test"),
				}
				if isBadRequestUpdateInProgress(errorInput) {
					t.Fatalf("isBadRequestUpdateInProgress returned true, expected false")
				}
			},
		},
		{
			name: "failure: not AWS error",
			testFunc: func(t *testing.T) {
				errorInput := errors.New("test")
				if isBadRequestUpdateInProgress(errorInput) {
					t.Fatalf("isBadRequestUpdateInProgress returned true, expected false")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
