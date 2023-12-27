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
	m                 *Metadata
	fileSystems       map[string]*FileSystem
	filesystem2Assocs map[string]map[string]struct{}
	associations      map[string]*DataRepositoryAssociation
}

func NewFakeCloudProvider() *FakeCloudProvider {
	return &FakeCloudProvider{
		m:                 &Metadata{InstanceID: "InstanceID", InstanceType: "Region", Region: "az"},
		fileSystems:       make(map[string]*FileSystem),
		filesystem2Assocs: make(map[string]map[string]struct{}),
		associations:      make(map[string]*DataRepositoryAssociation),
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
		FileSystemId:             fmt.Sprintf("fs-%d", random.Uint64()),
		CapacityGiB:              fileSystemOptions.CapacityGiB,
		DnsName:                  "test.us-east-1.fsx.amazonaws.com",
		MountName:                "random",
		StorageType:              fileSystemOptions.StorageType,
		DeploymentType:           fileSystemOptions.DeploymentType,
		PerUnitStorageThroughput: fileSystemOptions.PerUnitStorageThroughput,
	}
	c.fileSystems[volumeName] = fs
	return fs, nil
}

func (c *FakeCloudProvider) ResizeFileSystem(ctx context.Context, volumeName string, newSizeGiB int64) (int64, error) {
	fs, exists := c.fileSystems[volumeName]
	if !exists {
		return 0, ErrNotFound
	}

	fs.CapacityGiB = newSizeGiB
	c.fileSystems[volumeName] = fs
	return newSizeGiB, nil
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

func (c *FakeCloudProvider) WaitForFileSystemResize(ctx context.Context, fileSystemId string, resizeGiB int64) error {
	return nil
}

func (c *FakeCloudProvider) CreateDataRepositoryAssociation(ctx context.Context, filesystemId string, draOptions *DataRepositoryAssociationOptions) (*DataRepositoryAssociation, error) {
	dra := &DataRepositoryAssociation{
		AssociationId:               fmt.Sprintf("dra-%d", random.Uint64()),
		FileSystemId:                filesystemId,
		BatchImportMetaDataOnCreate: draOptions.BatchImportMetaDataOnCreate,
		DataRepositoryPath:          draOptions.DataRepositoryPath,
		FileSystemPath:              draOptions.FileSystemPath,
		S3:                          draOptions.S3,
	}
	if _, ok := c.filesystem2Assocs[filesystemId]; !ok {
		c.filesystem2Assocs[filesystemId] = make(map[string]struct{})
	}
	c.filesystem2Assocs[filesystemId][dra.AssociationId] = struct{}{}
	c.associations[dra.AssociationId] = dra
	return dra, nil
}

func (c *FakeCloudProvider) DescribeDataRepositoryAssociationsInFileSystem(ctx context.Context, fileSystemId string) ([]*DataRepositoryAssociation, error) {
	assocIdsOnFilesystem, ok := c.filesystem2Assocs[fileSystemId]
	if !ok {
		return []*DataRepositoryAssociation{}, nil
	}
	dras := []*DataRepositoryAssociation{}
	for assocId := range assocIdsOnFilesystem {
		dras = append(dras, c.associations[assocId])
	}
	return dras, nil
}

func (c *FakeCloudProvider) WaitForDataRepositoryAssociationAvailable(ctx context.Context, associationId string) error {
	return nil
}

func (c *FakeCloudProvider) DeleteDataRepositoryAssociation(ctx context.Context, associationId string) error {
	if assoc, ok := c.associations[associationId]; ok {
		if _, ok := c.filesystem2Assocs[assoc.FileSystemId]; ok {
			delete(c.filesystem2Assocs[assoc.FileSystemId], associationId)
		}
		delete(c.associations, associationId)
	}
	return nil
}
