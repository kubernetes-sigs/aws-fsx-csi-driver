/*
Copyright 2026 The Kubernetes Authors.

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud/mocks"
)

const testARN = "arn:aws:fsx:us-east-1:111122223333:file-system/fs-1"

func TestTagFileSystemSendsOnlyTheGivenTags(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockFSx := mocks.NewMockFSx(mockCtl)
	c := &cloud{fsx: mockFSx, fsIdCache: map[string]string{}}
	mockFSx.EXPECT().TagResource(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, in *fsx.TagResourceInput, _ ...func(*fsx.Options)) (*fsx.TagResourceOutput, error) {
			if aws.ToString(in.ResourceARN) != testARN {
				t.Errorf("ARN %q", aws.ToString(in.ResourceARN))
			}
			// Sorted by key, and nothing else.
			want := [][2]string{{"example.com/cluster", "my-cluster"}, {SharedTagKey, SharedTagValue}}
			if len(in.Tags) != len(want) {
				t.Fatalf("tags %+v", in.Tags)
			}
			for i, tag := range in.Tags {
				if got := [2]string{aws.ToString(tag.Key), aws.ToString(tag.Value)}; got != want[i] {
					t.Errorf("tag %d = %v, want %v", i, got, want[i])
				}
			}
			return &fsx.TagResourceOutput{}, nil
		})
	tags := map[string]string{SharedTagKey: SharedTagValue, "example.com/cluster": "my-cluster"}
	if err := c.TagFileSystem(context.Background(), testARN, tags); err != nil {
		t.Fatal(err)
	}
}

func TestTagFileSystemErrors(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockFSx := mocks.NewMockFSx(mockCtl)
	c := &cloud{fsx: mockFSx, fsIdCache: map[string]string{}}
	if err := c.TagFileSystem(context.Background(), "", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected an error for an empty ARN")
	}
	if err := c.TagFileSystem(context.Background(), testARN, nil); err == nil {
		t.Fatal("expected an error for no tags")
	}
	denied := errors.New("AccessDeniedException")
	mockFSx.EXPECT().TagResource(gomock.Any(), gomock.Any()).Return(nil, denied)
	if err := c.TagFileSystem(context.Background(), testARN, map[string]string{"k": "v"}); !errors.Is(err, denied) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateFileSystemAddsDriverTags(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockFSx := mocks.NewMockFSx(mockCtl)
	c := &cloud{fsx: mockFSx, fsIdCache: map[string]string{}}
	stop := errors.New("stop after the request")
	mockFSx.EXPECT().CreateFileSystem(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, in *fsx.CreateFileSystemInput, _ ...func(*fsx.Options)) (*fsx.CreateFileSystemOutput, error) {
			got := map[string]string{}
			for _, tag := range in.Tags {
				got[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			want := map[string]string{VolumeNameTagKey: "pvc-1", SharedTagKey: SharedTagValue, "team": "a"}
			if len(got) != len(want) || len(in.Tags) != len(want) {
				t.Fatalf("tags %v, want %v", got, want)
			}
			for k, v := range want {
				if got[k] != v {
					t.Errorf("tag %s = %q, want %q", k, got[k], v)
				}
			}
			return nil, stop
		})
	_, err := c.CreateFileSystem(context.Background(), "pvc-1", &FileSystemOptions{CapacityGiB: 1200, SubnetId: "subnet-1", ExtraTags: []string{"team=a"}})
	if err == nil {
		t.Fatal("expected the stubbed error")
	}
}

func TestDescribeFileSystemReturnsTagsAndARN(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockFSx := mocks.NewMockFSx(mockCtl)
	c := &cloud{fsx: mockFSx, fsIdCache: map[string]string{}}
	mockFSx.EXPECT().DescribeFileSystems(gomock.Any(), gomock.Any()).Return(&fsx.DescribeFileSystemsOutput{
		FileSystems: []types.FileSystem{{
			FileSystemId:        aws.String("fs-1"),
			ResourceARN:         aws.String(testARN),
			StorageCapacity:     aws.Int32(1200),
			DNSName:             aws.String("fs-1.fsx.us-east-1.amazonaws.com"),
			LustreConfiguration: &types.LustreFileSystemConfiguration{},
			Tags:                []types.Tag{{Key: aws.String(VolumeNameTagKey), Value: aws.String("pvc-1")}},
		}},
	}, nil)
	fs, err := c.DescribeFileSystem(context.Background(), "fs-1")
	if err != nil {
		t.Fatal(err)
	}
	if fs.ResourceARN != testARN || fs.Tags[VolumeNameTagKey] != "pvc-1" {
		t.Fatalf("got %+v", fs)
	}
}
