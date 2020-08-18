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
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud/mocks"
)

func TestCreateFileSystem(t *testing.T) {
	var (
		volumeName                     = "volumeName"
		fileSystemId                   = "fs-1234"
		volumeSizeGiB            int64 = 1200
		subnetId                       = "subnet-056da83524edbe641"
		securityGroupIds               = []string{"sg-086f61ea73388fb6b", "sg-0145e55e976000c9e"}
		dnsname                        = "test.fsx.us-west-2.amazoawd.com"
		s3ImportPath                   = "s3://fsx-s3-data-repository"
		s3ExportPath                   = "s3://fsx-s3-data-repository/export"
		deploymentType                 = fsx.LustreDeploymentTypeScratch2
		mountName                      = "fsx"
		kmsKeyId                       = "arn:aws:kms:us-east-1:215474938041:key/48313a27-7d88-4b51-98a4-fdf5bc80dbbe"
		perUnitStorageThroughput int64 = 200
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
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							MountName: aws.String(mountName),
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
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							MountName: aws.String(mountName),
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
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							MountName: aws.String(mountName),
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
						DNSName:         aws.String(dnsname),
						LustreConfiguration: &fsx.LustreFileSystemConfiguration{
							MountName: aws.String(mountName),
							DeploymentType: aws.String(fsx.LustreDeploymentTypePersistent1),
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
					S3ImportPath:     s3ImportPath,
					S3ExportPath:     s3ExportPath,
				}

				dataRepositoryConfiguration := &fsx.DataRepositoryConfiguration{}
				dataRepositoryConfiguration.SetImportPath(s3ImportPath)
				dataRepositoryConfiguration.SetExportPath(s3ExportPath)

				lustreFileSystemConfiguration := &fsx.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					MountName:                   aws.String(mountName),
				}

				output := &fsx.CreateFileSystemOutput{
					FileSystem: &fsx.FileSystem{
						FileSystemId:        aws.String(fileSystemId),
						StorageCapacity:     aws.Int64(volumeSizeGiB),
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
		fileSystemId        = "fs-1234"
		volumeSizeGiB int64 = 1200
		dnsname             = "test.fsx.us-west-2.amazoawd.com"
		s3ImportPath        = "s3://fsx-s3-data-repository"
		s3ExportPath        = "s3://fsx-s3-data-repository/export"
		mountName           = "fsx"
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
							DNSName:         aws.String(dnsname),
							LustreConfiguration: &fsx.LustreFileSystemConfiguration{
								MountName: aws.String(mountName),
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
				dataRepositoryConfiguration.SetImportPath(s3ImportPath)
				dataRepositoryConfiguration.SetExportPath(s3ExportPath)

				lustreFileSystemConfiguration := &fsx.LustreFileSystemConfiguration{
					DataRepositoryConfiguration: dataRepositoryConfiguration,
					MountName:                   aws.String(mountName),
				}

				output := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:        aws.String(fileSystemId),
							StorageCapacity:     aws.Int64(volumeSizeGiB),
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
