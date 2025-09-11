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
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

const (
	// DefaultVolumeSize represents the default size used
	// this is the minimum FSx for Lustre FS size
	DefaultVolumeSize = 1200

	// PollCheckInterval specifies the interval to check if filesystem is ready;
	// needs to be shorter than the provisioner timeout
	PollCheckInterval = 30 * time.Second
	// PollCheckTimeout specifies the time limit for polling DescribeFileSystems
	// for a completed create/update operation. FSx for Lustre filesystem
	// creation time is around 5 minutes, and update time varies depending on
	// target file system values
	PollCheckTimeout = 10 * time.Minute
)

// Tags
const (
	// VolumeNameTagKey is the key value that refers to the volume's name.
	VolumeNameTagKey = "CSIVolumeName"
)

// Set during build time via -ldflags
var driverVersion string

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
	FileSystemId             string
	CapacityGiB              int32
	DnsName                  string
	MountName                string
	StorageType              string
	DeploymentType           string
	PerUnitStorageThroughput int32
}

// FileSystemOptions represents the options to create FSx for Lustre filesystem
type FileSystemOptions struct {
	CapacityGiB                   int32
	SubnetId                      string
	SecurityGroupIds              []string
	AutoImportPolicy              string
	S3ImportPath                  string
	S3ExportPath                  string
	DeploymentType                string
	KmsKeyId                      string
	PerUnitStorageThroughput      int32
	StorageType                   string
	DriveCacheType                string
	DailyAutomaticBackupStartTime string
	AutomaticBackupRetentionDays  int32
	CopyTagsToBackups             bool
	DataCompressionType           string
	WeeklyMaintenanceStartTime    string
	FileSystemTypeVersion         string
	ExtraTags                     []string
	EfaEnabled                    bool
	MetadataConfigurationMode     string
	MetadataIops                  int32
}

// FSx abstracts FSx client to facilitate its mocking.
type FSx interface {
	CreateFileSystem(context.Context, *fsx.CreateFileSystemInput, ...func(*fsx.Options)) (*fsx.CreateFileSystemOutput, error)
	UpdateFileSystem(context.Context, *fsx.UpdateFileSystemInput, ...func(*fsx.Options)) (*fsx.UpdateFileSystemOutput, error)
	DeleteFileSystem(context.Context, *fsx.DeleteFileSystemInput, ...func(*fsx.Options)) (*fsx.DeleteFileSystemOutput, error)
	DescribeFileSystems(context.Context, *fsx.DescribeFileSystemsInput, ...func(*fsx.Options)) (*fsx.DescribeFileSystemsOutput, error)
}

type Cloud interface {
	CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error)
	ResizeFileSystem(ctx context.Context, fileSystemId string, newSizeGiB int32) (int32, error)
	DeleteFileSystem(ctx context.Context, fileSystemId string) (err error)
	DescribeFileSystem(ctx context.Context, fileSystemId string) (fs *FileSystem, err error)
	WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error
	WaitForFileSystemResize(ctx context.Context, fileSystemId string, resizeGiB int32) error
	FindFileSystemByVolumeName(ctx context.Context, volumeName string) (*FileSystem, error)
}

type cloud struct {
	region string
	fsx    FSx
}

// NewCloud returns a new instance of AWS cloud
// It panics if config is invalid
func NewCloud(region string) (Cloud, error) {
	os.Setenv("AWS_EXECUTION_ENV", "aws-fsx-csi-driver-"+driverVersion)
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	awsConfig.Region = region
	// Set RetryMaxAttempts to a high value. It will be "overwritten" if context deadline comes sooner.
	awsConfig.RetryMaxAttempts = 8

	svc := fsx.NewFromConfig(awsConfig)
	return &cloud{
		region: region,
		fsx:    svc,
	}, nil
}

func (c *cloud) CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error) {
	if len(fileSystemOptions.SubnetId) == 0 {
		return nil, fmt.Errorf("SubnetId is required")
	}

	lustreConfiguration := &types.CreateFileSystemLustreConfiguration{}

	if fileSystemOptions.AutoImportPolicy != "" {
		lustreConfiguration.AutoImportPolicy = types.AutoImportPolicyType(fileSystemOptions.AutoImportPolicy)
	}

	if fileSystemOptions.S3ImportPath != "" {
		lustreConfiguration.ImportPath = aws.String(fileSystemOptions.S3ImportPath)
	}

	if fileSystemOptions.S3ExportPath != "" {
		lustreConfiguration.ExportPath = aws.String(fileSystemOptions.S3ExportPath)
	}

	if fileSystemOptions.DeploymentType != "" {
		lustreConfiguration.DeploymentType = types.LustreDeploymentType(fileSystemOptions.DeploymentType)
	}

	if fileSystemOptions.DriveCacheType != "" {
		lustreConfiguration.DriveCacheType = types.DriveCacheType(fileSystemOptions.DriveCacheType)
	}

	if fileSystemOptions.PerUnitStorageThroughput != 0 {
		lustreConfiguration.PerUnitStorageThroughput = aws.Int32(fileSystemOptions.PerUnitStorageThroughput)
	}

	if fileSystemOptions.AutomaticBackupRetentionDays != 0 {
		lustreConfiguration.AutomaticBackupRetentionDays = aws.Int32(fileSystemOptions.AutomaticBackupRetentionDays)
		if fileSystemOptions.DailyAutomaticBackupStartTime != "" {
			lustreConfiguration.DailyAutomaticBackupStartTime = aws.String(fileSystemOptions.DailyAutomaticBackupStartTime)
		}
	}

	if fileSystemOptions.CopyTagsToBackups {
		lustreConfiguration.CopyTagsToBackups = aws.Bool(true)
	}

	if fileSystemOptions.DataCompressionType != "" {
		lustreConfiguration.DataCompressionType = types.DataCompressionType(fileSystemOptions.DataCompressionType)
	}

	if fileSystemOptions.WeeklyMaintenanceStartTime != "" {
		lustreConfiguration.WeeklyMaintenanceStartTime = aws.String(fileSystemOptions.WeeklyMaintenanceStartTime)
	}

	if fileSystemOptions.EfaEnabled {
		lustreConfiguration.EfaEnabled = aws.Bool(true)
	}

	if fileSystemOptions.MetadataConfigurationMode != "" {
		metadataConfiguration := &types.CreateFileSystemLustreMetadataConfiguration{}
		metadataConfiguration.Mode = types.MetadataConfigurationMode(fileSystemOptions.MetadataConfigurationMode)
		if fileSystemOptions.MetadataIops != 0 {
			metadataConfiguration.Iops = aws.Int32(fileSystemOptions.MetadataIops)
		}
		lustreConfiguration.MetadataConfiguration = metadataConfiguration
	}
	var tags = []types.Tag{
		{
			Key:   aws.String(VolumeNameTagKey),
			Value: aws.String(volumeName),
		},
	}

	for _, extraTag := range fileSystemOptions.ExtraTags {
		extraTagSplit := strings.Split(extraTag, "=")
		tagKey := extraTagSplit[0]
		tagValue := extraTagSplit[1]

		tags = append(tags, types.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})
	}

	input := &fsx.CreateFileSystemInput{
		ClientRequestToken:  aws.String(volumeName),
		FileSystemType:      "LUSTRE",
		LustreConfiguration: lustreConfiguration,
		StorageCapacity:     aws.Int32(fileSystemOptions.CapacityGiB),
		SubnetIds:           []string{fileSystemOptions.SubnetId},
		SecurityGroupIds:    fileSystemOptions.SecurityGroupIds,
		Tags:                tags,
	}

	if fileSystemOptions.FileSystemTypeVersion != "" {
		input.FileSystemTypeVersion = aws.String(fileSystemOptions.FileSystemTypeVersion)
	}
	if fileSystemOptions.StorageType != "" {
		input.StorageType = types.StorageType(fileSystemOptions.StorageType)
	}
	if fileSystemOptions.KmsKeyId != "" {
		input.KmsKeyId = aws.String(fileSystemOptions.KmsKeyId)
	}

	output, err := c.fsx.CreateFileSystem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("CreateFileSystem failed: %v", err)
	}

	mountName := "fsx"
	if output.FileSystem.LustreConfiguration.MountName != nil {
		mountName = *output.FileSystem.LustreConfiguration.MountName
	}

	perUnitStorageThroughput := int32(0)
	if output.FileSystem.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *output.FileSystem.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileSystem{
		FileSystemId:             *output.FileSystem.FileSystemId,
		CapacityGiB:              *output.FileSystem.StorageCapacity,
		DnsName:                  *output.FileSystem.DNSName,
		MountName:                mountName,
		StorageType:              string(output.FileSystem.StorageType),
		DeploymentType:           string(output.FileSystem.LustreConfiguration.DeploymentType),
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

// ResizeFileSystem makes a request to the FSx API to update the storage capacity of the filesystem.
func (c *cloud) ResizeFileSystem(ctx context.Context, fileSystemId string, newSizeGiB int32) (int32, error) {
	originalFs, err := c.getFileSystem(ctx, fileSystemId)
	if err != nil {
		return 0, fmt.Errorf("DescribeFileSystems failed: %v", err)
	}

	input := &fsx.UpdateFileSystemInput{
		FileSystemId:    aws.String(fileSystemId),
		StorageCapacity: aws.Int32(newSizeGiB),
	}

	_, err = c.fsx.UpdateFileSystem(ctx, input)
	if err != nil {
		if !isBadRequestUpdateInProgress(err) {
			return *originalFs.StorageCapacity, fmt.Errorf("UpdateFileSystem failed: %v", err)
		}

		// If the error is because of an update in progress, check for an existing update with the same target storage
		// capacity as the current request. A previous volume expansion request that experienced a timeout could
		// have already made an equivalent update request to the FSx API.
		_, err = c.getUpdateResizeAdministrativeAction(ctx, fileSystemId, newSizeGiB)
		if err != nil {
			return *originalFs.StorageCapacity, err
		}
	}

	return newSizeGiB, nil
}

func (c *cloud) DeleteFileSystem(ctx context.Context, fileSystemId string) (err error) {
	input := &fsx.DeleteFileSystemInput{
		FileSystemId: aws.String(fileSystemId),
	}
	if _, err = c.fsx.DeleteFileSystem(ctx, input); err != nil {
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

	perUnitStorageThroughput := int32(0)
	if fs.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *fs.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileSystem{
		FileSystemId:             *fs.FileSystemId,
		CapacityGiB:              *fs.StorageCapacity,
		DnsName:                  *fs.DNSName,
		MountName:                mountName,
		StorageType:              string(fs.StorageType),
		DeploymentType:           string(fs.LustreConfiguration.DeploymentType),
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

func (c *cloud) WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error {
	err := wait.Poll(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		fs, err := c.getFileSystem(ctx, fileSystemId)
		if err != nil {
			return true, err
		}
		klog.V(2).InfoS("WaitForFileSystemAvailable", "filesystem", fileSystemId, "status", string(fs.Lifecycle))
		switch string(fs.Lifecycle) {
		case "AVAILABLE":
			return true, nil
		case "CREATING":
			return false, nil
		default:
			return true, fmt.Errorf("unexpected state for filesystem %s: %q", fileSystemId, string(fs.Lifecycle))
		}
	})

	return err
}

// WaitForFileSystemResize polls the FSx API for status of the update operation with the given target storage
// capacity. The polling terminates when the update operation reaches a completed, failed, or unknown state.
func (c *cloud) WaitForFileSystemResize(ctx context.Context, fileSystemId string, resizeGiB int32) error {
	err := wait.PollImmediate(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		updateAction, err := c.getUpdateResizeAdministrativeAction(ctx, fileSystemId, resizeGiB)
		if err != nil {
			return true, err
		}

		klog.V(2).InfoS("WaitForFileSystemResize", "filesystem", fileSystemId, "update status", string(updateAction.Status))
		switch string(updateAction.Status) {
		case "PENDING", "IN_PROGRESS":
			// The resizing workflow has not completed
			return false, nil
		case "UPDATED_OPTIMIZING", "COMPLETED":
			// The resizing workflow has completed and the filesystem is in a usable state
			return true, nil
		default:
			// "FAILURE" is the only remaining AdministrativeAction status
			return true, fmt.Errorf("update failed for filesystem %s: %q", fileSystemId, *updateAction.FailureDetails.Message)
		}
	})

	return err
}

func (c *cloud) getFileSystem(ctx context.Context, fileSystemId string) (*types.FileSystem, error) {
	input := &fsx.DescribeFileSystemsInput{
		FileSystemIds: []string{fileSystemId},
	}

	output, err := c.fsx.DescribeFileSystems(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(output.FileSystems) > 1 {
		return nil, ErrMultiFileSystems
	}

	if len(output.FileSystems) == 0 {
		return nil, ErrNotFound
	}

	return &(output.FileSystems[0]), nil
}

// getUpdateResizeAdministrativeAction retrieves the AdministrativeAction associated with a file system update with the
// given target storage capacity, if one exists.
func (c *cloud) getUpdateResizeAdministrativeAction(ctx context.Context, fileSystemId string, resizeGiB int32) (*types.AdministrativeAction, error) {
	fs, err := c.getFileSystem(ctx, fileSystemId)
	if err != nil {
		return nil, fmt.Errorf("DescribeFileSystems failed: %v", err)
	}

	if len(fs.AdministrativeActions) == 0 {
		return nil, fmt.Errorf("there is no update on filesystem %s", fileSystemId)
	}

	// AdministrativeAction items are ordered by newest to oldest start time, so use the first satisfactory target
	// storage capacity match
	for _, action := range fs.AdministrativeActions {
		if action.AdministrativeActionType == "FILE_SYSTEM_UPDATE" &&
			action.TargetFileSystemValues.StorageCapacity != nil &&
			*action.TargetFileSystemValues.StorageCapacity == resizeGiB {
			return &action, nil
		}
	}

	return nil, fmt.Errorf("there is no update with storage capacity of %d GiB on filesystem %s", resizeGiB, fileSystemId)
}

func isFileSystemNotFound(err error) bool {
	var notFound *types.FileSystemNotFound
	return errors.As(err, &notFound)
}

// isBadRequestUpdateInProgress identifies an error returned from the FSx API as a BadRequest with an "update already
// in progress" message.
func isBadRequestUpdateInProgress(err error) bool {
	var badRequest *types.BadRequest
	return errors.As(err, &badRequest) && strings.Contains(err.Error(), "Unable to perform the storage capacity update. There is an update already in progress.")
}

func (c *cloud) FindFileSystemByVolumeName(ctx context.Context, volumeName string) (*FileSystem, error) {
	var nextToken *string
	const maxResults = 100

	klog.V(4).InfoS("Searching for existing filesystem", "volumeName", volumeName)

	// AWS FSx DescribeFileSystems API doesn't support filtering by tags,
	// so we paginate through all filesystems and filter client-side
	for {
		input := &fsx.DescribeFileSystemsInput{
			MaxResults: aws.Int32(maxResults),
			NextToken:  nextToken,
		}

		output, err := c.fsx.DescribeFileSystems(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe filesystems: %v", err)
		}

		klog.V(5).InfoS("Checking batch of filesystems", "count", len(output.FileSystems))

		// Search current batch
		for _, fs := range output.FileSystems {
			// Skip if filesystem is being deleted
			if fs.Lifecycle != types.FileSystemLifecycleAvailable &&
				fs.Lifecycle != types.FileSystemLifecycleCreating {
				continue
			}

			// Check tags for volume name match
			for _, tag := range fs.Tags {
				if *tag.Key == VolumeNameTagKey && *tag.Value == volumeName {
					klog.V(2).InfoS("Found existing filesystem",
						"volumeName", volumeName,
						"fileSystemId", *fs.FileSystemId,
						"lifecycle", string(fs.Lifecycle))

					mountName := "fsx"
					if fs.LustreConfiguration.MountName != nil {
						mountName = *fs.LustreConfiguration.MountName
					}

					perUnitStorageThroughput := int32(0)
					if fs.LustreConfiguration.PerUnitStorageThroughput != nil {
						perUnitStorageThroughput = *fs.LustreConfiguration.PerUnitStorageThroughput
					}

					return &FileSystem{
						FileSystemId:             *fs.FileSystemId,
						CapacityGiB:              *fs.StorageCapacity,
						DnsName:                  *fs.DNSName,
						MountName:                mountName,
						StorageType:              string(fs.StorageType),
						DeploymentType:           string(fs.LustreConfiguration.DeploymentType),
						PerUnitStorageThroughput: perUnitStorageThroughput,
					}, nil
				}
			}
		}

		// Check if more results exist
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	klog.V(2).InfoS("No existing filesystem found", "volumeName", volumeName)
	return nil, ErrNotFound
}
