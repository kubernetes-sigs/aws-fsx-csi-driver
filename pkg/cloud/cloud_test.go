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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud/mocks"
)

func TestCreateFileSystem(t *testing.T) {
	var (
		volumeName                          = "volumeName"
		fileSystemId                        = "fs-1234"
		volumeSizeGiB                 int64 = 1200
		subnetId                            = "subnet-056da83524edbe641"
		securityGroupIds                    = []string{"sg-086f61ea73388fb6b", "sg-0145e55e976000c9e"}
		dnsname                             = "test.fsx.us-west-2.amazoawd.com"
		autoImportPolicy                    = "NEW_CHANGED"
		s3ImportPath                        = "s3://fsx-s3-data-repository"
		s3ExportPath                        = "s3://fsx-s3-data-repository/export"
		deploymentType                      = fsx.LustreDeploymentTypeScratch2
		mountName                           = "fsx"
		kmsKeyId                            = "arn:aws:kms:us-east-1:215474938041:key/48313a27-7d88-4b51-98a4-fdf5bc80dbbe"
		perUnitStorageThroughput      int64 = 200
		DailyAutomaticBackupStartTime       = "00:00:00"
		AutomaticBackupRetentionDays  int64 = 1
		CopyTagsToBackups                   = true
		dataCompressionTypeNone             = "NONE"
		dataCompressionTypeLZ4              = "LZ4"
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeSsd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType: aws.String(fsx.LustreDeploymentTypeScratch1),
							MountName:      aws.String(mountName),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   deploymentType,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeSsd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType: aws.String(deploymentType),
							MountName:      aws.String(mountName),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   deploymentType,
					StorageType:      fsx.StorageTypeSsd,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeSsd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType: aws.String(deploymentType),
							MountName:      aws.String(mountName),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   fsx.LustreDeploymentTypePersistent1,
					StorageType:      fsx.StorageTypeHdd,
					DriveCacheType:   fsx.DriveCacheTypeNone,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeHdd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType: aws.String(fsx.LustreDeploymentTypePersistent1),
							MountName:      aws.String(mountName),
							DriveCacheType: aws.String(fsx.DriveCacheTypeNone),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   deploymentType,
					StorageType:      fsx.StorageTypeHdd,
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					AutoImportPolicy: autoImportPolicy,
					S3ImportPath:     s3ImportPath,
					S3ExportPath:     s3ExportPath,
				}

				dataRepositoryConfiguration := &fsx.DataRepositoryConfiguration{}
				dataRepositoryConfiguration.SetAutoImportPolicy(autoImportPolicy)
				dataRepositoryConfiguration.SetImportPath(s3ImportPath)
				dataRepositoryConfiguration.SetExportPath(s3ExportPath)

				lustreFileSystemConfiguration := &fsx.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					DeploymentType:              aws.String(deploymentType),
					MountName:                   aws.String(mountName),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:        aws.String(fileSystemId),
						StorageCapacity:     aws.Int64(volumeSizeGiB),
						StorageType:         aws.String(fsx.StorageTypeSsd),
						DNSName:             aws.String(dnsname),
						LustreConfiguration: lustreFileSystemConfiguration,
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					S3ExportPath:     s3ExportPath,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					AutoImportPolicy: autoImportPolicy,
					S3ImportPath:     "s3://bucket1/import",
					S3ExportPath:     "s3://bucket2/export",
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
					DeploymentType:   deploymentType,
					KmsKeyId:         kmsKeyId,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:              volumeSizeGiB,
					SubnetId:                 subnetId,
					SecurityGroupIds:         securityGroupIds,
					DeploymentType:           deploymentType,
					PerUnitStorageThroughput: perUnitStorageThroughput,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
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
			name: "fail: CreateFileSystemWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:                   volumeSizeGiB,
					SubnetId:                      subnetId,
					SecurityGroupIds:              securityGroupIds,
					AutomaticBackupRetentionDays:  AutomaticBackupRetentionDays,
					DailyAutomaticBackupStartTime: DailyAutomaticBackupStartTime,
					CopyTagsToBackups:             CopyTagsToBackups,
					DeploymentType:                fsx.LustreDeploymentTypePersistent1,
				}

				lustreFileSystemConfiguration := &fsx.LustreFileSystemConfiguration{
					DeploymentType:                aws.String(fsx.LustreDeploymentTypePersistent1),
					MountName:                     aws.String(mountName),
					AutomaticBackupRetentionDays:  aws.Int64(AutomaticBackupRetentionDays),
					DailyAutomaticBackupStartTime: aws.String(DailyAutomaticBackupStartTime),
					CopyTagsToBackups:             aws.Bool(CopyTagsToBackups),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:        aws.String(fileSystemId),
						StorageCapacity:     aws.Int64(volumeSizeGiB),
						StorageType:         aws.String(fsx.StorageTypeSsd),
						DNSName:             aws.String(dnsname),
						LustreConfiguration: lustreFileSystemConfiguration,
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: dataCompressionTypeNone,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeSsd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType:      aws.String(fsx.LustreDeploymentTypeScratch1),
							MountName:           aws.String(mountName),
							DataCompressionType: aws.String(dataCompressionTypeNone),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: dataCompressionTypeLZ4,
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:    aws.String(fileSystemId),
						StorageCapacity: aws.Int64(volumeSizeGiB),
						StorageType:     aws.String(fsx.StorageTypeSsd),
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							DeploymentType:      aws.String(fsx.LustreDeploymentTypeScratch1),
							MountName:           aws.String(mountName),
							DataCompressionType: aws.String(dataCompressionTypeLZ4),
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				req := &FileSystemOptions{
					CapacityGiB:         volumeSizeGiB,
					SubnetId:            subnetId,
					SecurityGroupIds:    securityGroupIds,
					DataCompressionType: "ZFS",
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileSystemWithContext failed"))
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
					fsx: mockFSx,
				}

				output := &fsx.DeleteFileSystemOutput{}
				ctx := context.Background()
				mockFSx.EXPECT().DeleteFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
					fsx: mockFSx,
				}

				ctx := context.Background()
				mockFSx.EXPECT().DeleteFileSystemWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DeleteFileSystemWithContext failed"))
				err := c.DeleteFileSystem(ctx, fileSystemId)
				if err == nil {
					t.Fatal("DeleteFileSystem is not failed")
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
		volumeSizeGiB    int64 = 1200
		dnsname                = "test.fsx.us-west-2.amazoawd.com"
		autoImportPolicy       = "NEW_CHANGED"
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
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(volumeSizeGiB),
							StorageType:     aws.String(fsx.StorageTypeSsd),
							DNSName:         aws.String(dnsname),
							LustreConfiguration: &fsx.LustreFileSystemConfiguration{
								DeploymentType: aws.String(fsx.LustreDeploymentTypeScratch1),
								MountName:      aws.String(mountName),
							},
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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

				dataRepositoryConfiguration := &fsx.DataRepositoryConfiguration{}
				dataRepositoryConfiguration.SetAutoImportPolicy(autoImportPolicy)
				dataRepositoryConfiguration.SetImportPath(s3ImportPath)
				dataRepositoryConfiguration.SetExportPath(s3ExportPath)

				lustreFileSystemConfiguration := &fsx.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					DeploymentType:              aws.String(fsx.LustreDeploymentTypeScratch1),
					MountName:                   aws.String(mountName),
				}

				output := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:        aws.String(fileSystemId),
							StorageCapacity:     aws.Int64(volumeSizeGiB),
							StorageType:         aws.String(fsx.StorageTypeSsd),
							DNSName:             aws.String(dnsname),
							LustreConfiguration: lustreFileSystemConfiguration,
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
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
				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DescribeFileSystemsWithContext failed"))
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
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				describeInput := &fsx.DescribeFileSystemsInput{
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateOutput := &fsx.UpdateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId: aws.String(fileSystemId),
						AdministrativeActions: []*fsx.AdministrativeAction{
							{
								AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeFileSystemUpdate),
								Status:                   aws.String(fsx.StatusInProgress),
								TargetFileSystemValues: &fsx.FileSystem{
									StorageCapacity: aws.Int64(finalSizeGiB),
								},
							},
							{
								AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeStorageOptimization),
								Status:                   aws.String(fsx.StatusPending),
							},
						},
					},
				}

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(updateOutput, nil)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
							AdministrativeActions: []*fsx.AdministrativeAction{
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeFileSystemUpdate),
									Status:                   aws.String(fsx.StatusInProgress),
									TargetFileSystemValues: &fsx.FileSystem{
										StorageCapacity: aws.Int64(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeStorageOptimization),
									Status:                   aws.String(fsx.StatusPending),
								},
							},
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateError := awserr.New(fsx.ErrCodeBadRequest, "Unable to perform the storage capacity update. There is an update already in progress.", nil)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeError := awserr.New(fsx.ErrCodeFileSystemNotFound, "test", nil)
				resizeError := fmt.Errorf("DescribeFileSystems failed: %v", describeError)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(nil, describeError)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateError := awserr.New(fsx.ErrCodeBadRequest, "test", nil)
				resizeError := fmt.Errorf("UpdateFileSystem failed: %v", updateError)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
						},
					},
				}
				describeError := awserr.New(fsx.ErrCodeBadRequest, "test", nil)
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateError := awserr.New(fsx.ErrCodeBadRequest, "Unable to perform the storage capacity update. There is an update already in progress.", nil)
				resizeError := fmt.Errorf("DescribeFileSystems failed: %v", describeError)

				gomock.InOrder(
					mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil),
					mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(nil, describeError),
				)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateError := awserr.New(fsx.ErrCodeBadRequest, "Unable to perform the storage capacity update. There is an update already in progress.", nil)
				resizeError := fmt.Errorf("there is no update on filesystem %s", fileSystemId)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
							AdministrativeActions: []*fsx.AdministrativeAction{
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeFileSystemUpdate),
									Status:                   aws.String(fsx.StatusInProgress),
									TargetFileSystemValues: &fsx.FileSystem{
										StorageCapacity: aws.Int64(4800),
									},
								},
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeStorageOptimization),
									Status:                   aws.String(fsx.StatusPending),
								},
							},
						},
					},
				}
				updateInput := &fsx.UpdateFileSystemInput{
					FileSystemId:    aws.String(fileSystemId),
					StorageCapacity: aws.Int64(finalSizeGiB),
				}
				updateError := awserr.New(fsx.ErrCodeBadRequest, "Unable to perform the storage capacity update. There is an update already in progress.", nil)
				resizeError := fmt.Errorf("there is no update with storage capacity of %d GiB on filesystem %s", finalSizeGiB, fileSystemId)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Times(2).Return(describeOutput, nil)
				mockFSx.EXPECT().UpdateFileSystemWithContext(gomock.Eq(ctx), gomock.Eq(updateInput)).Return(nil, updateError)
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
		initialSizeGiB int64 = 1200
		finalSizeGiB   int64 = 2400
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(finalSizeGiB),
							AdministrativeActions: []*fsx.AdministrativeAction{
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeFileSystemUpdate),
									Status:                   aws.String(fsx.StatusUpdatedOptimizing),
									TargetFileSystemValues: &fsx.FileSystem{
										StorageCapacity: aws.Int64(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeStorageOptimization),
									Status:                   aws.String(fsx.StatusInProgress),
								},
							},
						},
					},
				}

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
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
					FileSystemIds: []*string{aws.String(fileSystemId)},
				}
				describeOutput := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(initialSizeGiB),
							AdministrativeActions: []*fsx.AdministrativeAction{
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeFileSystemUpdate),
									Status:                   aws.String(fsx.StatusFailed),
									FailureDetails: &fsx.AdministrativeActionFailureDetails{
										Message: aws.String("test"),
									},
									TargetFileSystemValues: &fsx.FileSystem{
										StorageCapacity: aws.Int64(finalSizeGiB),
									},
								},
								{
									AdministrativeActionType: aws.String(fsx.AdministrativeActionTypeStorageOptimization),
									Status:                   aws.String(fsx.StatusPending),
								},
							},
						},
					},
				}
				waitError := fmt.Errorf("update failed for filesystem %s: %q", fileSystemId, *describeOutput.FileSystems[0].AdministrativeActions[0].FailureDetails.Message)

				mockFSx.EXPECT().DescribeFileSystemsWithContext(gomock.Eq(ctx), gomock.Eq(describeInput)).Return(describeOutput, nil)
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

func TestIsBadRequestUpdateInProgress(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: BadRequest update in progress",
			testFunc: func(t *testing.T) {
				errorInput := awserr.New(fsx.ErrCodeBadRequest, "Unable to perform the storage capacity update. There is an update already in progress.", nil)
				if !isBadRequestUpdateInProgress(errorInput) {
					t.Fatalf("isBadRequestUpdateInProgress returned false, expected true")
				}
			},
		},
		{
			name: "failure: AWS error, different type",
			testFunc: func(t *testing.T) {
				errorInput := awserr.New(fsx.ErrCodeFileSystemNotFound, "test", nil)
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
