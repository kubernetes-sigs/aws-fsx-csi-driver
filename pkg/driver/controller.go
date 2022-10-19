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

package driver

import (
	"context"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/util"
)

// controllerCaps represents the capability of controller service
var controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
	csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
}

const (
	volumeContextDnsName                      = "dnsname"
	volumeContextMountName                    = "mountname"
	volumeContextBaseDir                      = "basedir"
	volumeContextSubDir                       = "subdir"
	volumeParamsSubnetId                      = "subnetId"
	volumeParamsSecurityGroupIds              = "securityGroupIds"
	volumeParamsAutoImportPolicy              = "autoImportPolicy"
	volumeParamsS3ImportPath                  = "s3ImportPath"
	volumeParamsS3ExportPath                  = "s3ExportPath"
	volumeParamsDeploymentType                = "deploymentType"
	volumeParamsKmsKeyId                      = "kmsKeyId"
	volumeParamsPerUnitStorageThroughput      = "perUnitStorageThroughput"
	volumeParamsStorageType                   = "storageType"
	volumeParamsDriveCacheType                = "driveCacheType"
	volumeParamsAutomaticBackupRetentionDays  = "automaticBackupRetentionDays"
	volumeParamsDailyAutomaticBackupStartTime = "dailyAutomaticBackupStartTime"
	volumeParamsCopyTagsToBackups             = "copyTagsToBackups"
	volumeParamsDataCompressionType           = "dataCompressionType"
	volumeParamsWeeklyMaintenanceStartTime    = "weeklyMaintenanceStartTime"
	volumeParamsFileSystemTypeVersion         = "fileSystemTypeVersion"
	volumeParamsExtraTags                     = "extraTags"
	volumeParamsDnsName                       = "dnsname"
	volumeParamsMountName                     = "mountname"
	volumeParamsBaseDir                       = "basedir"
)

const (
	provisioningModeFileSystem = "fsx-filesystem"
	provisioningModeSubDir     = "fsx-subdir"
)

const tempMountPathPrefix = "/tmp/csi"

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(4).Infof("CreateVolume: called with args %#v", req)
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if !d.isValidVolumeCapabilities(volCaps) {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported")
	}

	var provisioningMode string
	params := req.GetParameters()
	if _, ok := params[volumeParamsDnsName]; !ok {
		provisioningMode = provisioningModeFileSystem
	} else {
		provisioningMode = provisioningModeSubDir
	}

	provisioner, err := d.newProvisioner(provisioningMode)
	if err != nil {
		return nil, err
	}

	klog.V(4).Infof("CreateVolume: provisioning volume in %s mode", provisioningMode)
	vol, err := provisioner.Provision(ctx, req)
	if err != nil {
		return nil, err
	}

	return &csi.CreateVolumeResponse{Volume: vol}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(4).Infof("DeleteVolume: called with args: %#v", req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	fsxVolume, err := getFsxVolumeFromVolumeID(volumeID)
	if err != nil {
		return nil, err
	}

	provisioningMode := fsxVolume.GetProvisioningMode()

	provisioner, err := d.newProvisioner(provisioningMode)
	if err != nil {
		return nil, err
	}

	klog.V(4).Infof("DeleteVolume: deleting volume in %s mode", provisioningMode)
	if err := provisioner.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).Infof("ControllerGetCapabilities: called with args %#v", req)
	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	klog.V(4).Infof("GetCapacity: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.V(4).Infof("ListVolumes: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	klog.V(4).Infof("ValidateVolumeCapabilities: called with args %#v", req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if _, err := d.cloud.DescribeFileSystem(ctx, volumeID); err != nil {
		if err == cloud.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Volume not found")
		}
		return nil, status.Errorf(codes.Internal, "Could not get volume with ID %q: %v", volumeID, err)
	}

	confirmed := d.isValidVolumeCapabilities(volCaps)
	if confirmed {
		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				// TODO if volume context is provided, should validate it too
				// VolumeContext:      req.GetVolumeContext(),
				VolumeCapabilities: volCaps,
				// TODO if parameters are provided, should validate them too
				// Parameters:      req.GetParameters(),
			},
		}, nil
	} else {
		return &csi.ValidateVolumeCapabilitiesResponse{}, nil
	}
}

func (d *Driver) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range volumeCaps {
			if c.GetMode() == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.V(4).Infof("ControllerExpandVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	fsxVolume, err := getFsxVolumeFromVolumeID(volumeID)
	if err != nil {
		return nil, err
	}

	if provisioningMode := fsxVolume.GetProvisioningMode(); provisioningMode != provisioningModeFileSystem {
		return nil, status.Errorf(codes.Unimplemented, "ControllerExpandVolume is not supported in %s mode", provisioningMode)
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity range not provided")
	}

	fs, err := d.cloud.DescribeFileSystem(ctx, volumeID)
	if err != nil {
		if err == cloud.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "Filesystem not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "DescribeFileSystem failed: %v", err)
	}

	newSizeGiB := util.RoundUpVolumeSize(capRange.GetRequiredBytes(), fs.DeploymentType, fs.StorageType, fs.PerUnitStorageThroughput)
	if util.GiBToBytes(newSizeGiB) != capRange.GetRequiredBytes() {
		klog.V(4).Infof("ControllerExpandVolume: requested storage capacity of %d bytes has been rounded to a valid storage capacity of %d GiB", capRange.GetRequiredBytes(), newSizeGiB)
	}
	if capRange.GetLimitBytes() > 0 && util.GiBToBytes(newSizeGiB) > capRange.GetLimitBytes() {
		return nil, status.Errorf(codes.OutOfRange, "Requested storage capacity of %d bytes exceeds capacity limit of %d bytes.", util.GiBToBytes(newSizeGiB), capRange.GetLimitBytes())
	}
	if newSizeGiB <= fs.CapacityGiB {
		// Current capacity is sufficient to satisfy the request
		klog.V(4).Infof("ControllerExpandVolume: current filesystem capacity of %d GiB matches or exceeds requested storage capacity of %d GiB, returning with success", fs.CapacityGiB, newSizeGiB)
		return &csi.ControllerExpandVolumeResponse{
			CapacityBytes:         util.GiBToBytes(fs.CapacityGiB),
			NodeExpansionRequired: false,
		}, nil
	}

	finalGiB, err := d.cloud.ResizeFileSystem(ctx, volumeID, newSizeGiB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resize failed: %v", err)
	}

	err = d.cloud.WaitForFileSystemResize(ctx, fs.FileSystemId, finalGiB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "filesystem is not resized: %v", err)
	}

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         util.GiBToBytes(finalGiB),
		NodeExpansionRequired: false,
	}, nil
}

func (d *Driver) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) newProvisioner(mode string) (Provisioner, error) {
	switch mode {
	case provisioningModeFileSystem:
		return FileSystemProvisioner{cloud: d.cloud}, nil
	case provisioningModeSubDir:
		return SubDirProvisioner{mounter: d.mounter}, nil
	default:
		return nil, status.Errorf(codes.Internal, "Invalid provisioning mode %s", mode)
	}
}

// Fsx Volume will be mounted
// filesystem provisioning (fsid is not empty):
//   - dnsname@tcp:/mountname
//
// subdir provisioning:
//   - dnsname@tcp:/mountname/basedir/subdir
type FsxVolume struct {
	fsid      string
	dnsname   string
	mountname string
	// Base directory (fileset) path to create volumes under in subdir provisioning mode
	baseDir string
	// Sub directory (fileset) name (not path) to create volumes under in subdir provisioning mode
	subDir string
	// uuid keeps VolumeID uniqueness
	// because Fsx supports MULTI_NODE_MULTI_WRITER access mode, so can mount same directory from multiple Volumes
	uuid string
}

func (v FsxVolume) GetProvisioningMode() string {
	if v.fsid != "" {
		return provisioningModeFileSystem
	}

	return provisioningModeSubDir
}

// VolumeID form is expected one of the following
// filesystem provisioning:
//   - fs-xxx
//
// subdir provisioning:
//   - 10.x.x.x:mountname:::uuid
//   - 10.x.x.x:mountname::basedir:uuid
//   - 10.x.x.x:mountname:basedir:subdir:uuid
func getFsxVolumeFromVolumeID(id string) (FsxVolume, error) {
	tokens := strings.Split(id, volumeIDSeparator)
	if len(tokens) == 1 {
		// filesystem provisioning
		return FsxVolume{fsid: tokens[0]}, nil
	} else if len(tokens) != 5 {
		return FsxVolume{}, status.Errorf(codes.InvalidArgument, "Volume ID '%s' is invalid: Expected one or four five separated by '%s'", id, volumeIDSeparator)
	}

	// subdir provisioning
	return FsxVolume{
		fsid:      "",
		dnsname:   tokens[0],
		mountname: tokens[1],
		baseDir:   tokens[2],
		subDir:    tokens[3],
		uuid:      tokens[4],
	}, nil
}
