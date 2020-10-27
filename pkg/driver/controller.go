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
	"fmt"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	// controllerCaps represents the capability of controller service
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
)

const (
	volumeContextSubPath                      = "subpath"
	volumeContextDnsName                      = "dnsname"
	volumeContextMountName                    = "mountname"
	sharedVolumeIdPrefix                      = "shared"
	volumeParamsFileSystemId                  = "fileSystemId"
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
)

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

	// create a new volume with idempotency
	// idempotency is handled by `CreateFileSystem`

	volumeParams := req.GetParameters()
	fileSystemId := volumeParams[volumeParamsFileSystemId]
	if fileSystemId != "" {
		return d.findExistingFilesystem(ctx, fileSystemId, volName)
	} else {
		return d.createFilesystemFromRequest(ctx, req)
	}
}

func (d *Driver) findExistingFilesystem(ctx context.Context, fileSystemId, volName string) (*csi.CreateVolumeResponse, error) {
	var (
		fs  *cloud.FileSystem
		err error
	)
	if fs, err = d.cloud.DescribeFileSystem(ctx, fileSystemId); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not find existing filesystem %q: %v", fileSystemId, err)
	}
	err = d.cloud.WaitForFileSystemAvailable(ctx, fs.FileSystemId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Filesystem is not ready: %v", err)
	}
	return newCreateVolumeResponseWithSubPath(volName, fs), nil
}

func (d *Driver) createFilesystemFromRequest(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volumeParams := req.GetParameters()
	subnetId := volumeParams[volumeParamsSubnetId]
	securityGroupIds := volumeParams[volumeParamsSecurityGroupIds]
	fsOptions := &cloud.FileSystemOptions{
		SubnetId:         subnetId,
		SecurityGroupIds: strings.Split(securityGroupIds, ","),
	}

	if val, ok := volumeParams[volumeParamsAutoImportPolicy]; ok {
		fsOptions.AutoImportPolicy = val
	}

	if val, ok := volumeParams[volumeParamsS3ImportPath]; ok {
		fsOptions.S3ImportPath = val
	}

	if val, ok := volumeParams[volumeParamsS3ExportPath]; ok {
		fsOptions.S3ExportPath = val
	}

	if val, ok := volumeParams[volumeParamsDeploymentType]; ok {
		fsOptions.DeploymentType = val
	}

	if val, ok := volumeParams[volumeParamsKmsKeyId]; ok {
		fsOptions.KmsKeyId = val
	}

	if val, ok := volumeParams[volumeParamsDailyAutomaticBackupStartTime]; ok {
		fsOptions.DailyAutomaticBackupStartTime = val
	}

	if val, ok := volumeParams[volumeParamsAutomaticBackupRetentionDays]; ok {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "automaticBackupRetentionDays must be a number")
		}
		fsOptions.AutomaticBackupRetentionDays = n
	}

	if val, ok := volumeParams[volumeParamsCopyTagsToBackups]; ok {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "copyTagsToBackups must be a bool")
		}
		fsOptions.CopyTagsToBackups = b
	}

	if val, ok := volumeParams[volumeParamsStorageType]; ok {
		fsOptions.StorageType = val
	}

	if val, ok := volumeParams[volumeParamsDriveCacheType]; ok {
		fsOptions.DriveCacheType = val
	}

	if val, ok := volumeParams[volumeParamsPerUnitStorageThroughput]; ok {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "perUnitStorageThroughput must be a number")
		}
		fsOptions.PerUnitStorageThroughput = n
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		fsOptions.CapacityGiB = cloud.DefaultVolumeSize
	} else {
		fsOptions.CapacityGiB = util.RoundUpVolumeSize(capRange.GetRequiredBytes(), fsOptions.DeploymentType, fsOptions.StorageType, fsOptions.PerUnitStorageThroughput)
	}
	volName := req.GetName()
	fs, err := d.cloud.CreateFileSystem(ctx, volName, fsOptions)
	if err != nil {
		switch err {
		case cloud.ErrFsExistsDiffSize:
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "Could not create volume %q: %v", volName, err)
		}
	}
	err = d.cloud.WaitForFileSystemAvailable(ctx, fs.FileSystemId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Filesystem is not ready: %v", err)
	}

	return newCreateVolumeResponse(fs), nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(4).Infof("DeleteVolume: called with args: %#v", req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}
	// We don't have any metadata during the delete step, just the VolumeId.
	// As such, we prefix volumes from a shared fSX volume with a prefix.
	// Don't delete those, they are not managed by this driver.
	if strings.HasPrefix(volumeID, sharedVolumeIdPrefix) {
		// TODO: If filesystem exists, we might want to remove the folder
		klog.V(4).Infof("DeleteVolume: shared volume found, returning with success")
		return &csi.DeleteVolumeResponse{}, nil
	}

	if err := d.cloud.DeleteFileSystem(ctx, volumeID); err != nil {
		if err == cloud.ErrNotFound {
			klog.V(4).Infof("DeleteVolume: volume not found, returning with success")
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "Could not delete volume ID %q: %v", volumeID, err)
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
	return nil, status.Error(codes.Unimplemented, "")
}

func newCreateVolumeResponseWithSubPath(subPath string, fs *cloud.FileSystem) *csi.CreateVolumeResponse {
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      fmt.Sprintf("%s/%s/%s", sharedVolumeIdPrefix, fs.FileSystemId, subPath),
			CapacityBytes: util.GiBToBytes(fs.CapacityGiB),
			VolumeContext: map[string]string{
				volumeContextDnsName:   fs.DnsName,
				volumeContextMountName: fs.MountName,
				volumeContextSubPath:   subPath,
			},
		},
	}
}

func newCreateVolumeResponse(fs *cloud.FileSystem) *csi.CreateVolumeResponse {
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      fs.FileSystemId,
			CapacityBytes: util.GiBToBytes(fs.CapacityGiB),
			VolumeContext: map[string]string{
				volumeContextDnsName:   fs.DnsName,
				volumeContextMountName: fs.MountName,
			},
		},
	}
}
