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
	fsOptions := &cloud.FileSystemOptions{}

	for k, v := range params {
		switch k {
		case volumeParamsSubnetId:
			fsOptions.SubnetId = v
		case volumeParamsSecurityGroupIds:
			fsOptions.SecurityGroupIds = strings.Split(v, ",")
		case volumeParamsAutoImportPolicy:
			fsOptions.AutoImportPolicy = v
		case volumeParamsS3ImportPath:
			fsOptions.S3ImportPath = v
		case volumeParamsS3ExportPath:
			fsOptions.S3ExportPath = v
		case volumeParamsDeploymentType:
			fsOptions.DeploymentType = v
		case volumeParamsKmsKeyId:
			fsOptions.KmsKeyId = v
		case volumeParamsDailyAutomaticBackupStartTime:
			fsOptions.DailyAutomaticBackupStartTime = v
		case volumeParamsAutomaticBackupRetentionDays:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "automaticBackupRetentionDays must be a number")
			}
			fsOptions.AutomaticBackupRetentionDays = n
		case volumeParamsCopyTagsToBackups:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "copyTagsToBackups must be a bool")
			}
			fsOptions.CopyTagsToBackups = b
		case volumeParamsStorageType:
			fsOptions.StorageType = v
		case volumeParamsDriveCacheType:
			fsOptions.DriveCacheType = v
		case volumeParamsDataCompressionType:
			fsOptions.DataCompressionType = v
		case volumeParamsPerUnitStorageThroughput:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "perUnitStorageThroughput must be a number")
			}
			fsOptions.PerUnitStorageThroughput = n
		case volumeParamsWeeklyMaintenanceStartTime:
			fsOptions.WeeklyMaintenanceStartTime = v
		case volumeParamsFileSystemTypeVersion:
			fsOptions.FileSystemTypeVersion = v
		case volumeParamsExtraTags:
			extraTags := strings.Split(v, ",")
			fsOptions.ExtraTags = extraTags
		default:
			if !strings.HasPrefix(k, kubernetesExternalProvisionerKeyPrefix) {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid parameter key %q for CreateVolume in %s mode", k, provisioningModeFileSystem)
			}
		}
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		fsOptions.CapacityGiB = cloud.DefaultVolumeSize
	} else {
		fsOptions.CapacityGiB = util.RoundUpVolumeSize(capRange.GetRequiredBytes(), fsOptions.DeploymentType, fsOptions.StorageType, fsOptions.PerUnitStorageThroughput)
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
