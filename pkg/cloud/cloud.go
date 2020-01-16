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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/fsx"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

const (
	// DefaultVolumeSize represents the default size used
	// this is the minimum FSx for Lustre FS size
	DefaultVolumeSize = 1200
)

// Tags
const (
	// VolumeNameTagKey is the key value that refers to the volume's name.
	VolumeNameTagKey = "CSIVolumeName"
)

var (
	// ErrMultiDisks is an error that is returned when multiple
	// disks are found with the same volume name.
	ErrMultiFileSystems = errors.New("Multiple filesystems with same ID")

	// ErrFsExistsDiffSize is an error that is returned if a filesystem
	// exists with a given ID, but a different capacity is requested.
	ErrFsExistsDiffSize = errors.New("There is already a disk with same ID and different size")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("Resource was not found")
)

// FileSystem represents a FSx for Lustre filesystem
type FileSystem struct {
	FileSystemId string
	CapacityGiB  int64
	DnsName      string
	MountName    string
}

// FileSystemOptions represents the options to create FSx for Lustre filesystem
type FileSystemOptions struct {
	CapacityGiB      int64
	SubnetId         string
	SecurityGroupIds []string
	S3ImportPath     string
	S3ExportPath     string
	DeploymentType   string
	KmsKeyId         string
}

// FSx abstracts FSx client to facilitate its mocking.
// See https://docs.aws.amazon.com/sdk-for-go/api/service/fsx/ for details
type FSx interface {
	CreateFileSystemWithContext(aws.Context, *fsx.CreateFileSystemInput, ...request.Option) (*fsx.CreateFileSystemOutput, error)
	DeleteFileSystemWithContext(aws.Context, *fsx.DeleteFileSystemInput, ...request.Option) (*fsx.DeleteFileSystemOutput, error)
	DescribeFileSystemsWithContext(aws.Context, *fsx.DescribeFileSystemsInput, ...request.Option) (*fsx.DescribeFileSystemsOutput, error)
}

type Cloud interface {
	CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error)
	DeleteFileSystem(ctx context.Context, fileSystemId string) (err error)
	DescribeFileSystem(ctx context.Context, fileSystemId string) (fs *FileSystem, err error)
	WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error
}

type cloud struct {
	fsx FSx
}

// NewCloud returns a new instance of AWS cloud
// It panics if session is invalid
func NewCloud(region string) Cloud {
	awsConfig := &aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	return &cloud{
		fsx: fsx.New(session.Must(session.NewSession(awsConfig))),
	}
}

func (c *cloud) CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error) {
	if len(fileSystemOptions.SubnetId) == 0 {
		return nil, fmt.Errorf("SubnetId is required")
	}

	lustreConfiguration := &fsx.CreateFileSystemLustreConfiguration{}

	if fileSystemOptions.S3ImportPath != "" {
		lustreConfiguration.SetImportPath(fileSystemOptions.S3ImportPath)
	}

	if fileSystemOptions.S3ExportPath != "" {
		lustreConfiguration.SetExportPath(fileSystemOptions.S3ExportPath)
	}

	if fileSystemOptions.DeploymentType != "" {
		lustreConfiguration.SetDeploymentType(fileSystemOptions.DeploymentType)
	}

	input := &fsx.CreateFileSystemInput{
		ClientRequestToken:  aws.String(volumeName),
		FileSystemType:      aws.String("LUSTRE"),
		LustreConfiguration: lustreConfiguration,
		StorageCapacity:     aws.Int64(fileSystemOptions.CapacityGiB),
		SubnetIds:           []*string{aws.String(fileSystemOptions.SubnetId)},
		SecurityGroupIds:    aws.StringSlice(fileSystemOptions.SecurityGroupIds),
		Tags: []*fsx.Tag{
			{
				Key:   aws.String(VolumeNameTagKey),
				Value: aws.String(volumeName),
			},
		},
	}

	if fileSystemOptions.KmsKeyId != "" {
		input.KmsKeyId = aws.String(fileSystemOptions.KmsKeyId)
	}

	output, err := c.fsx.CreateFileSystemWithContext(ctx, input)
	if err != nil {
		if isIncompatibleParameter(err) {
			return nil, ErrFsExistsDiffSize
		}
		return nil, fmt.Errorf("CreateFileSystem failed: %v", err)
	}

	mountName := "fsx"
	if output.FileSystem.LustreConfiguration.MountName != nil {
		mountName = *output.FileSystem.LustreConfiguration.MountName
	}

	return &FileSystem{
		FileSystemId: *output.FileSystem.FileSystemId,
		CapacityGiB:  *output.FileSystem.StorageCapacity,
		DnsName:      *output.FileSystem.DNSName,
		MountName:    mountName,
	}, nil
}

func (c *cloud) DeleteFileSystem(ctx context.Context, fileSystemId string) (err error) {
	input := &fsx.DeleteFileSystemInput{
		FileSystemId: aws.String(fileSystemId),
	}
	if _, err = c.fsx.DeleteFileSystemWithContext(ctx, input); err != nil {
		if isFileSystemNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("DeleteFileSystem failed: %v", err)
	}
	return nil
}

func (c *cloud) DescribeFileSystem(ctx context.Context, fileSystemId string) (*FileSystem, error) {
	fs, err := c.getFileSystem(ctx, fileSystemId)
	if err != nil {
		return nil, err
	}

	mountName := "fsx"
	if fs.LustreConfiguration.MountName != nil {
		mountName = *fs.LustreConfiguration.MountName
	}

	return &FileSystem{
		FileSystemId: *fs.FileSystemId,
		CapacityGiB:  *fs.StorageCapacity,
		DnsName:      *fs.DNSName,
		MountName:    mountName,
	}, nil
}

func (c *cloud) WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error {
	var (
		// interval to check if filesystem is ready
		// needs to be shorter than the provisioner timeout
		checkInterval = 15 * time.Second
		// FSx for lustre filesystem creation time is around 5 mins
		checkTimeout = 7 * time.Minute
	)
	err := wait.Poll(checkInterval, checkTimeout, func() (done bool, err error) {
		fs, err := c.getFileSystem(ctx, fileSystemId)
		if err != nil {
			return true, err
		}
		klog.V(4).Infof("WaitForFileSystemAvailable filesystem status is: %v", *fs.Lifecycle)
		switch *fs.Lifecycle {
		case "AVAILABLE":
			return true, nil
		case "CREATING":
			return false, nil
		default:
			return true, fmt.Errorf("unexpected state for filesystem %s: %q", fileSystemId, *fs.Lifecycle)
		}
	})

	return err

}

func (c *cloud) getFileSystem(ctx context.Context, fileSystemId string) (*fsx.FileSystem, error) {
	input := &fsx.DescribeFileSystemsInput{
		FileSystemIds: []*string{aws.String(fileSystemId)},
	}

	output, err := c.fsx.DescribeFileSystemsWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(output.FileSystems) > 1 {
		return nil, ErrMultiFileSystems
	}

	if len(output.FileSystems) == 0 {
		return nil, ErrNotFound
	}

	return output.FileSystems[0], nil
}

func isFileSystemNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeFileSystemNotFound {
			return true
		}
	}
	return false
}

func isIncompatibleParameter(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeIncompatibleParameterError {
			return true
		}
	}
	return false
}
