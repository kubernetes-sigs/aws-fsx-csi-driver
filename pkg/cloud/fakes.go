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
	"fmt"
	"math/rand"
	"time"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type FakeCloudProvider struct {
	m           *metadata
	fileSystems map[string]*FileSystem
}

func NewFakeCloudProvider() *FakeCloudProvider {
	return &FakeCloudProvider{
		m:           &metadata{"instanceID", "region", "az"},
		fileSystems: make(map[string]*FileSystem),
	}
}

func (c *FakeCloudProvider) GetMetadata() MetadataService {
	return c.m
}

func (c *FakeCloudProvider) CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error) {
	fs, exists := c.fileSystems[volumeName]
	if exists {
		if fs.CapacityGiB == fileSystemOptions.CapacityGiB {
			return fs, nil
		} else {
			return nil, ErrFsExistsDiffSize
		}
	}

	fs = &FileSystem{
		FileSystemId: fmt.Sprintf("fs-%d", random.Uint64()),
		CapacityGiB:  fileSystemOptions.CapacityGiB,
		DnsName:      "test.us-east-1.fsx.amazonaws.com",
		MountName:    "random",
	}
	c.fileSystems[volumeName] = fs
	return fs, nil
}

func (c *FakeCloudProvider) DeleteFileSystem(ctx context.Context, volumeID string) (err error) {
	delete(c.fileSystems, volumeID)
	for name, fs := range c.fileSystems {
		if fs.FileSystemId == volumeID {
			delete(c.fileSystems, name)
		}
	}
	return nil
}

func (c *FakeCloudProvider) DescribeFileSystem(ctx context.Context, volumeID string) (fs *FileSystem, err error) {
	for _, fs := range c.fileSystems {
		if fs.FileSystemId == volumeID {
			return fs, nil
		}
	}
	return nil, ErrNotFound
}

func (c *FakeCloudProvider) WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error {
	return nil
}
