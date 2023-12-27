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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/fsx"
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
	// PollCheckTimeout specifies the time limit for polling
	// DescribeFileSystems/DescribeDataRepositoryAssociation for a completed
	// create/update operation. FSx for Lustre filesystem
	// creation time is around 5 minutes, Data Repository Association
	// creation time is around 10 minutes, and update time varies depending on
	// target file system values
	PollCheckTimeout = 15 * time.Minute
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

	// ErrMultiAssociations is an error that is returned when multiple
	// associations are found with the same volume name.
	ErrMultiAssociations = errors.New("Multiple data repository associations with same ID")

	// ErrFsExistsDiffSize is an error that is returned if a filesystem
	// exists with a given ID, but a different capacity is requested.
	ErrFsExistsDiffSize = errors.New("There is already a disk with same ID and different size")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("Resource was not found")
)

// FileSystem represents a FSx for Lustre filesystem
type FileSystem struct {
	FileSystemId             string
	CapacityGiB              int64
	DnsName                  string
	MountName                string
	StorageType              string
	DeploymentType           string
	PerUnitStorageThroughput int64
}

// FileSystemOptions represents the options to create FSx for Lustre filesystem
type FileSystemOptions struct {
	CapacityGiB                   int64
	SubnetId                      string
	SecurityGroupIds              []string
	AutoImportPolicy              string
	S3ImportPath                  string
	S3ExportPath                  string
	DeploymentType                string
	KmsKeyId                      string
	PerUnitStorageThroughput      int64
	StorageType                   string
	DriveCacheType                string
	DailyAutomaticBackupStartTime string
	AutomaticBackupRetentionDays  int64
	CopyTagsToBackups             bool
	DataCompressionType           string
	WeeklyMaintenanceStartTime    string
	FileSystemTypeVersion         string
	ExtraTags                     []string
}

type DataRepositoryAssociation struct {
	AssociationId               string
	FileSystemId                string
	BatchImportMetaDataOnCreate bool
	DataRepositoryPath          string
	FileSystemPath              string
	S3                          *S3DataRepositoryConfiguration
}

type DataRepositoryAssociationOptions struct {
	BatchImportMetaDataOnCreate bool                           `yaml:"batchImportMetaDataOnCreate,omitempty" json:"batchImportMetaDataOnCreate,omitempty"`
	DataRepositoryPath          string                         `yaml:"dataRepositoryPath,omitempty" json:"dataRepositoryPath,omitempty"`
	FileSystemPath              string                         `yaml:"fileSystemPath,omitempty" json:"fileSystemPath,omitempty"`
	S3                          *S3DataRepositoryConfiguration `yaml:"s3,omitempty" json:"s3,omitempty"`
}

type S3DataRepositoryConfiguration struct {
	AutoExportPolicy *AutoExportPolicy `yaml:"autoExportPolicy,omitempty" json:"autoExportPolicy,omitempty"`
	AutoImportPolicy *AutoImportPolicy `yaml:"autoImportPolicy,omitempty" json:"autoImportPolicy,omitempty"`
}

type AutoExportPolicy struct {
	Events []*string `yaml:"events" json:"events"`
}

type AutoImportPolicy struct {
	Events []*string `yaml:"events" json:"events"`
}

// FSx abstracts FSx client to facilitate its mocking.
// See https://docs.aws.amazon.com/sdk-for-go/api/service/fsx/ for details
type FSx interface {
	CreateFileSystemWithContext(aws.Context, *fsx.CreateFileSystemInput, ...request.Option) (*fsx.CreateFileSystemOutput, error)
	UpdateFileSystemWithContext(aws.Context, *fsx.UpdateFileSystemInput, ...request.Option) (*fsx.UpdateFileSystemOutput, error)
	DeleteFileSystemWithContext(aws.Context, *fsx.DeleteFileSystemInput, ...request.Option) (*fsx.DeleteFileSystemOutput, error)
	DescribeFileSystemsWithContext(aws.Context, *fsx.DescribeFileSystemsInput, ...request.Option) (*fsx.DescribeFileSystemsOutput, error)
	CreateDataRepositoryAssociationWithContext(ctx aws.Context, input *fsx.CreateDataRepositoryAssociationInput, opts ...request.Option) (*fsx.CreateDataRepositoryAssociationOutput, error)
	DescribeDataRepositoryAssociationsWithContext(ctx aws.Context, input *fsx.DescribeDataRepositoryAssociationsInput, opts ...request.Option) (*fsx.DescribeDataRepositoryAssociationsOutput, error)
	DeleteDataRepositoryAssociationWithContext(ctx aws.Context, input *fsx.DeleteDataRepositoryAssociationInput, opts ...request.Option) (*fsx.DeleteDataRepositoryAssociationOutput, error)
}

type Cloud interface {
	CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error)
	ResizeFileSystem(ctx context.Context, fileSystemId string, newSizeGiB int64) (int64, error)
	DeleteFileSystem(ctx context.Context, fileSystemId string) (err error)
	DescribeFileSystem(ctx context.Context, fileSystemId string) (fs *FileSystem, err error)
	WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error
	WaitForFileSystemResize(ctx context.Context, fileSystemId string, resizeGiB int64) error
	CreateDataRepositoryAssociation(ctx context.Context, filesystemId string, draOptions *DataRepositoryAssociationOptions) (*DataRepositoryAssociation, error)
	DescribeDataRepositoryAssociationsInFileSystem(ctx context.Context, fileSystemId string) ([]*DataRepositoryAssociation, error)
	WaitForDataRepositoryAssociationAvailable(ctx context.Context, associationId string) error
	DeleteDataRepositoryAssociation(ctx context.Context, associationId string) error
}

type cloud struct {
	region string
	fsx    FSx
}

// NewCloud returns a new instance of AWS cloud
// It panics if session is invalid
func NewCloud(region string) (Cloud, error) {
	awsConfig := &aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
		// Set MaxRetries to a high value. It will be "overwritten" if context deadline comes sooner.
		MaxRetries: aws.Int(8),
	}

	os.Setenv("AWS_EXECUTION_ENV", "aws-fsx-csi-driver-"+driverVersion)

	svc := fsx.New(session.Must(session.NewSession(awsConfig)))
	return &cloud{
		region: region,
		fsx:    svc,
	}, nil
}

func (c *cloud) CreateFileSystem(ctx context.Context, volumeName string, fileSystemOptions *FileSystemOptions) (fs *FileSystem, err error) {
	if len(fileSystemOptions.SubnetId) == 0 {
		return nil, fmt.Errorf("SubnetId is required")
	}

	lustreConfiguration := &fsx.CreateFileSystemLustreConfiguration{}

	if fileSystemOptions.AutoImportPolicy != "" {
		lustreConfiguration.SetAutoImportPolicy(fileSystemOptions.AutoImportPolicy)
	}

	if fileSystemOptions.S3ImportPath != "" {
		lustreConfiguration.SetImportPath(fileSystemOptions.S3ImportPath)
	}

	if fileSystemOptions.S3ExportPath != "" {
		lustreConfiguration.SetExportPath(fileSystemOptions.S3ExportPath)
	}

	if fileSystemOptions.DeploymentType != "" {
		lustreConfiguration.SetDeploymentType(fileSystemOptions.DeploymentType)
	}

	if fileSystemOptions.DriveCacheType != "" {
		lustreConfiguration.SetDriveCacheType(fileSystemOptions.DriveCacheType)
	}

	if fileSystemOptions.PerUnitStorageThroughput != 0 {
		lustreConfiguration.SetPerUnitStorageThroughput(fileSystemOptions.PerUnitStorageThroughput)
	}

	if fileSystemOptions.AutomaticBackupRetentionDays != 0 {
		lustreConfiguration.SetAutomaticBackupRetentionDays(fileSystemOptions.AutomaticBackupRetentionDays)
		if fileSystemOptions.DailyAutomaticBackupStartTime != "" {
			lustreConfiguration.SetDailyAutomaticBackupStartTime(fileSystemOptions.DailyAutomaticBackupStartTime)
		}
	}

	if fileSystemOptions.CopyTagsToBackups {
		lustreConfiguration.SetCopyTagsToBackups(true)
	}

	if fileSystemOptions.DataCompressionType != "" {
		lustreConfiguration.SetDataCompressionType(fileSystemOptions.DataCompressionType)
	}

	if fileSystemOptions.WeeklyMaintenanceStartTime != "" {
		lustreConfiguration.SetWeeklyMaintenanceStartTime(fileSystemOptions.WeeklyMaintenanceStartTime)
	}

	var tags = []*fsx.Tag{
		{
			Key:   aws.String(VolumeNameTagKey),
			Value: aws.String(volumeName),
		},
	}

	for _, extraTag := range fileSystemOptions.ExtraTags {
		extraTagSplit := strings.Split(extraTag, "=")
		tagKey := extraTagSplit[0]
		tagValue := extraTagSplit[1]

		tags = append(tags, &fsx.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})
	}

	input := &fsx.CreateFileSystemInput{
		ClientRequestToken:  aws.String(volumeName),
		FileSystemType:      aws.String("LUSTRE"),
		LustreConfiguration: lustreConfiguration,
		StorageCapacity:     aws.Int64(fileSystemOptions.CapacityGiB),
		SubnetIds:           []*string{aws.String(fileSystemOptions.SubnetId)},
		SecurityGroupIds:    aws.StringSlice(fileSystemOptions.SecurityGroupIds),
		Tags:                tags,
	}

	if fileSystemOptions.FileSystemTypeVersion != "" {
		input.FileSystemTypeVersion = aws.String(fileSystemOptions.FileSystemTypeVersion)
	}
	if fileSystemOptions.StorageType != "" {
		input.StorageType = aws.String(fileSystemOptions.StorageType)
	}
	if fileSystemOptions.KmsKeyId != "" {
		input.KmsKeyId = aws.String(fileSystemOptions.KmsKeyId)
	}

	output, err := c.fsx.CreateFileSystemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("CreateFileSystem failed: %v", err)
	}

	mountName := "fsx"
	if output.FileSystem.LustreConfiguration.MountName != nil {
		mountName = *output.FileSystem.LustreConfiguration.MountName
	}

	perUnitStorageThroughput := int64(0)
	if output.FileSystem.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *output.FileSystem.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileSystem{
		FileSystemId:             *output.FileSystem.FileSystemId,
		CapacityGiB:              *output.FileSystem.StorageCapacity,
		DnsName:                  *output.FileSystem.DNSName,
		MountName:                mountName,
		StorageType:              *output.FileSystem.StorageType,
		DeploymentType:           *output.FileSystem.LustreConfiguration.DeploymentType,
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

// ResizeFileSystem makes a request to the FSx API to update the storage capacity of the filesystem.
func (c *cloud) ResizeFileSystem(ctx context.Context, fileSystemId string, newSizeGiB int64) (int64, error) {
	originalFs, err := c.getFileSystem(ctx, fileSystemId)
	if err != nil {
		return 0, fmt.Errorf("DescribeFileSystems failed: %v", err)
	}

	input := &fsx.UpdateFileSystemInput{
		FileSystemId:    aws.String(fileSystemId),
		StorageCapacity: aws.Int64(newSizeGiB),
	}

	_, err = c.fsx.UpdateFileSystemWithContext(ctx, input)
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

	perUnitStorageThroughput := int64(0)
	if fs.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *fs.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileSystem{
		FileSystemId:             *fs.FileSystemId,
		CapacityGiB:              *fs.StorageCapacity,
		DnsName:                  *fs.DNSName,
		MountName:                mountName,
		StorageType:              *fs.StorageType,
		DeploymentType:           *fs.LustreConfiguration.DeploymentType,
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

func (c *cloud) WaitForFileSystemAvailable(ctx context.Context, fileSystemId string) error {
	err := wait.Poll(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		fs, err := c.getFileSystem(ctx, fileSystemId)
		if err != nil {
			return true, err
		}
		klog.V(2).InfoS("WaitForFileSystemAvailable", "filesystem", fileSystemId, "status", *fs.Lifecycle)
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

// WaitForFileSystemResize polls the FSx API for status of the update operation with the given target storage
// capacity. The polling terminates when the update operation reaches a completed, failed, or unknown state.
func (c *cloud) WaitForFileSystemResize(ctx context.Context, fileSystemId string, resizeGiB int64) error {
	err := wait.PollImmediate(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		updateAction, err := c.getUpdateResizeAdministrativeAction(ctx, fileSystemId, resizeGiB)
		if err != nil {
			return true, err
		}

		klog.V(2).InfoS("WaitForFileSystemResize", "filesystem", fileSystemId, "update status", *updateAction.Status)
		switch *updateAction.Status {
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

func (c *cloud) CreateDataRepositoryAssociation(
	ctx context.Context, filesystemId string, draOptions *DataRepositoryAssociationOptions,
) (*DataRepositoryAssociation, error) {

	input := &fsx.CreateDataRepositoryAssociationInput{
		BatchImportMetaDataOnCreate: aws.Bool(draOptions.BatchImportMetaDataOnCreate),
		DataRepositoryPath:          aws.String(draOptions.DataRepositoryPath),
		FileSystemId:                aws.String(filesystemId),
		FileSystemPath:              aws.String(draOptions.FileSystemPath),
	}
	if draOptions.S3 != nil {
		s3config := &fsx.S3DataRepositoryConfiguration{}
		if draOptions.S3.AutoExportPolicy != nil {
			s3config.AutoExportPolicy = &fsx.AutoExportPolicy{
				Events: draOptions.S3.AutoExportPolicy.Events,
			}
		}
		if draOptions.S3.AutoImportPolicy != nil {
			s3config.AutoImportPolicy = &fsx.AutoImportPolicy{
				Events: draOptions.S3.AutoImportPolicy.Events,
			}
		}
		input.S3 = s3config
	}

	output, err := c.fsx.CreateDataRepositoryAssociationWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("CreateDataRepositoryAssociation failed: %v", err)
	}

	dra := &DataRepositoryAssociation{
		AssociationId:               *output.Association.AssociationId,
		FileSystemId:                *output.Association.FileSystemId,
		BatchImportMetaDataOnCreate: *output.Association.BatchImportMetaDataOnCreate,
		DataRepositoryPath:          *output.Association.DataRepositoryPath,
		FileSystemPath:              *output.Association.FileSystemPath,
	}
	if output.Association.S3 != nil {
		s3 := S3DataRepositoryConfiguration{}
		if output.Association.S3.AutoExportPolicy != nil {
			s3.AutoExportPolicy = &AutoExportPolicy{
				Events: output.Association.S3.AutoExportPolicy.Events,
			}
		}
		if output.Association.S3.AutoImportPolicy != nil {
			s3.AutoImportPolicy = &AutoImportPolicy{
				Events: output.Association.S3.AutoImportPolicy.Events,
			}
		}
		dra.S3 = &s3
	}
	return dra, nil
}

func (c *cloud) DescribeDataRepositoryAssociationsInFileSystem(ctx context.Context, fileSystemId string) ([]*DataRepositoryAssociation, error) {
	input := &fsx.DescribeDataRepositoryAssociationsInput{
		Filters: []*fsx.Filter{{
			Name:   aws.String("file-system-id"),
			Values: []*string{aws.String(fileSystemId)},
		}},
	}

	output, err := c.fsx.DescribeDataRepositoryAssociationsWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	dras := []*DataRepositoryAssociation{}
	for _, fsxDra := range output.Associations {
		dra := &DataRepositoryAssociation{
			AssociationId:               *fsxDra.AssociationId,
			FileSystemId:                *fsxDra.FileSystemId,
			BatchImportMetaDataOnCreate: *fsxDra.BatchImportMetaDataOnCreate,
			DataRepositoryPath:          *fsxDra.DataRepositoryPath,
			FileSystemPath:              *fsxDra.FileSystemPath,
		}
		if fsxDra.S3 != nil {
			s3 := S3DataRepositoryConfiguration{}
			if fsxDra.S3.AutoExportPolicy != nil {
				s3.AutoExportPolicy = &AutoExportPolicy{
					Events: fsxDra.S3.AutoExportPolicy.Events,
				}
			}
			if fsxDra.S3.AutoImportPolicy != nil {
				s3.AutoImportPolicy = &AutoImportPolicy{
					Events: fsxDra.S3.AutoImportPolicy.Events,
				}
			}
			dra.S3 = &s3
		}
		dras = append(dras, dra)
	}

	return dras, nil
}

func (c *cloud) WaitForDataRepositoryAssociationAvailable(ctx context.Context, associationId string) error {
	err := wait.Poll(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		assoc, err := c.getDataRepositoryAssociation(ctx, associationId)
		if err != nil {
			return true, err
		}
		klog.V(2).InfoS("WaitForDataRepositoryAssociationAvailable", "associationId", associationId, "status", *assoc.Lifecycle)
		switch *assoc.Lifecycle {
		case "AVAILABLE":
			return true, nil
		case "CREATING":
			return false, nil
		default:
			return true, fmt.Errorf("unexpected state for data repository association %s: %q", associationId, *assoc.Lifecycle)
		}
	})

	return err
}

func (c *cloud) DeleteDataRepositoryAssociation(ctx context.Context, associationId string) error {
	input := &fsx.DeleteDataRepositoryAssociationInput{
		AssociationId:          aws.String(associationId),
		DeleteDataInFileSystem: aws.Bool(true),
	}
	if _, err := c.fsx.DeleteDataRepositoryAssociationWithContext(ctx, input); err != nil {
		if isAssociationNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("DeleteDataRepositoryAssociation failed: %v", err)
	}
	return nil
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

// getUpdateResizeAdministrativeAction retrieves the AdministrativeAction associated with a file system update with the
// given target storage capacity, if one exists.
func (c *cloud) getUpdateResizeAdministrativeAction(ctx context.Context, fileSystemId string, resizeGiB int64) (*fsx.AdministrativeAction, error) {
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
		if *action.AdministrativeActionType == "FILE_SYSTEM_UPDATE" &&
			action.TargetFileSystemValues.StorageCapacity != nil &&
			*action.TargetFileSystemValues.StorageCapacity == resizeGiB {
			return action, nil
		}
	}

	return nil, fmt.Errorf("there is no update with storage capacity of %d GiB on filesystem %s", resizeGiB, fileSystemId)
}

func (c *cloud) getDataRepositoryAssociation(ctx context.Context, associationId string) (*fsx.DataRepositoryAssociation, error) {
	input := &fsx.DescribeDataRepositoryAssociationsInput{
		AssociationIds: []*string{aws.String(associationId)},
	}

	output, err := c.fsx.DescribeDataRepositoryAssociationsWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(output.Associations) > 1 {
		return nil, ErrMultiAssociations
	}

	if len(output.Associations) == 0 {
		return nil, ErrNotFound
	}

	return output.Associations[0], nil
}

func isFileSystemNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeFileSystemNotFound {
			return true
		}
	}
	return false
}

func isAssociationNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeDataRepositoryAssociationNotFound {
			return true
		}
	}
	return false
}

// isBadRequestUpdateInProgress identifies an error returned from the FSx API as a BadRequest with an "update already
// in progress" message.
func isBadRequestUpdateInProgress(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeBadRequest &&
			awsErr.Message() == "Unable to perform the storage capacity update. There is an update already in progress." {
			return true
		}
	}
	return false
}
