package driver

import (
	"context"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/util"
)

type FileSystemProvisioner struct {
	driver *Driver
}

var _ Provisioner = FileSystemProvisioner{}

func (p FileSystemProvisioner) Provision(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.Volume, error) {
	// create a new volume with idempotency
	// idempotency is handled by `CreateFileSystem`
	params := req.GetParameters()
	subnetId := params[volumeParamsSubnetId]
	securityGroupIds := params[volumeParamsSecurityGroupIds]
	fsOptions := &cloud.FileSystemOptions{
		SubnetId:         subnetId,
		SecurityGroupIds: strings.Split(securityGroupIds, ","),
	}

	if val, ok := params[volumeParamsAutoImportPolicy]; ok {
		fsOptions.AutoImportPolicy = val
	}

	if val, ok := params[volumeParamsS3ImportPath]; ok {
		fsOptions.S3ImportPath = val
	}

	if val, ok := params[volumeParamsS3ExportPath]; ok {
		fsOptions.S3ExportPath = val
	}

	if val, ok := params[volumeParamsDeploymentType]; ok {
		fsOptions.DeploymentType = val
	}

	if val, ok := params[volumeParamsKmsKeyId]; ok {
		fsOptions.KmsKeyId = val
	}

	if val, ok := params[volumeParamsDailyAutomaticBackupStartTime]; ok {
		fsOptions.DailyAutomaticBackupStartTime = val
	}

	if val, ok := params[volumeParamsAutomaticBackupRetentionDays]; ok {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "automaticBackupRetentionDays must be a number")
		}
		fsOptions.AutomaticBackupRetentionDays = n
	}

	if val, ok := params[volumeParamsCopyTagsToBackups]; ok {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "copyTagsToBackups must be a bool")
		}
		fsOptions.CopyTagsToBackups = b
	}

	if val, ok := params[volumeParamsStorageType]; ok {
		fsOptions.StorageType = val
	}

	if val, ok := params[volumeParamsDriveCacheType]; ok {
		fsOptions.DriveCacheType = val
	}

	if val, ok := params[volumeParamsDataCompressionType]; ok {
		fsOptions.DataCompressionType = val
	}

	if val, ok := params[volumeParamsPerUnitStorageThroughput]; ok {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "perUnitStorageThroughput must be a number")
		}
		fsOptions.PerUnitStorageThroughput = n
	}

	if val, ok := params[volumeParamsWeeklyMaintenanceStartTime]; ok {
		fsOptions.WeeklyMaintenanceStartTime = val
	}

	if val, ok := params[volumeParamsFileSystemTypeVersion]; ok {
		fsOptions.FileSystemTypeVersion = val
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		fsOptions.CapacityGiB = cloud.DefaultVolumeSize
	} else {
		fsOptions.CapacityGiB = util.RoundUpVolumeSize(capRange.GetRequiredBytes(), fsOptions.DeploymentType, fsOptions.StorageType, fsOptions.PerUnitStorageThroughput)
	}

	if val, ok := params[volumeParamsExtraTags]; ok {
		extraTags := strings.Split(val, ",")
		fsOptions.ExtraTags = extraTags
	}

	volName := req.GetName()
	fs, err := p.driver.cloud.CreateFileSystem(ctx, volName, fsOptions)
	if err != nil {
		switch err {
		case cloud.ErrFsExistsDiffSize:
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "Could not create volume %q: %v", volName, err)
		}
	}

	err = p.driver.cloud.WaitForFileSystemAvailable(ctx, fs.FileSystemId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Filesystem is not ready: %v", err)
	}

	return &csi.Volume{
		VolumeId:      fs.FileSystemId,
		CapacityBytes: util.GiBToBytes(fs.CapacityGiB),
		VolumeContext: map[string]string{
			volumeContextDnsName:   fs.DnsName,
			volumeContextMountName: fs.MountName,
		},
	}, nil
}

func (p FileSystemProvisioner) Delete(ctx context.Context, req *csi.DeleteVolumeRequest) error {
	volumeID := req.GetVolumeId()
	if err := p.driver.cloud.DeleteFileSystem(ctx, volumeID); err != nil {
		if err == cloud.ErrNotFound {
			klog.V(4).Infof("DeleteVolume: volume not found, returning with success")
			return nil
		}
		return status.Errorf(codes.Internal, "Could not delete volume ID %q: %v", volumeID, err)
	}

	return nil
}
