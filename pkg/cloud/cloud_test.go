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
		volumeName             = "volumeName"
		fileSystemId           = "fs-1234"
		volumeSizeGiB    int64 = 3600
		subnetId               = "subnet-056da83524edbe641"
		securityGroupIds       = []string{"sg-086f61ea73388fb6b", "sg-0145e55e976000c9e"}
		dnsname                = "test.fsx.us-west-2.amazoawd.com"
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing subnet ID",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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
		volumeSizeGiB int64 = 3600
		dnsname             = "test.fsx.us-west-2.amazoawd.com"
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
				}

				output := &fsx.DescribeFileSystemsOutput{
					FileSystems: []*fsx.FileSystem{
						{
							FileSystemId:    aws.String(fileSystemId),
							StorageCapacity: aws.Int64(volumeSizeGiB),
							DNSName:         aws.String(dnsname),
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
					metadata: &metadata{"instanceID", "region", "az"},
					fsx:      mockFSx,
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
