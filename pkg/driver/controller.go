/*
Copyright 2018 The Kubernetes Authors.

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
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/cloud"
	"github.com/kubernetes-sigs/aws-fsx-csi-driver/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

var (
	// controllerCaps represents the capability of controller service
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
	validVolumeContext = sets.NewString("dnsname")
	validParameters    = sets.NewString("subnetId", "securityGroupIds", "s3ImportPath", "s3ExportPath")
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
	capRange := req.GetCapacityRange()
	var volumeSizeGiB int64
	if capRange == nil {
		volumeSizeGiB = cloud.DefaultVolumeSize
	} else {
		volumeSizeGiB = util.RoundUp3600GiB(capRange.GetRequiredBytes())
	}

	volumeParams := req.GetParameters()
	invalidParams := d.isValidParameters(volumeParams)
	if len(invalidParams) > 0 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Volume parameters %v not supported", invalidParams))
	}

	subnetId := volumeParams["subnetId"]
	securityGroupIds := volumeParams["securityGroupIds"]
	fsOptions := &cloud.FileSystemOptions{
		CapacityGiB:      volumeSizeGiB,
		SubnetId:         subnetId,
		SecurityGroupIds: strings.Split(securityGroupIds, ","),
	}

	if val, ok := volumeParams["s3ImportPath"]; ok {
		fsOptions.S3ImportPath = val
	}

	if val, ok := volumeParams["s3ExportPath"]; ok {
		fsOptions.S3ExportPath = val
	}

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

	volContext := req.GetVolumeContext()
	invalidContext := d.isValidVolumeContext(volContext)
	contextValid := len(invalidContext) == 0

	capsValid := d.isValidVolumeCapabilities(volCaps)

	volParams := req.GetParameters()
	invalidParams := d.isValidParameters(volParams)
	paramsValid := len(invalidParams) == 0

	if contextValid && capsValid && paramsValid {
		if _, err := d.cloud.DescribeFileSystem(ctx, volumeID); err != nil {
			if err == cloud.ErrNotFound {
				return nil, status.Error(codes.NotFound, "Volume not found")
			}
			return nil, status.Errorf(codes.Internal, "Could not get volume with ID %q: %v", volumeID, err)
		}

		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				VolumeContext:      volContext,
				VolumeCapabilities: volCaps,
				Parameters:         volParams,
			},
		}, nil
	} else {
		message := ""
		if !contextValid {
			message += fmt.Sprintf("Volume context %v not supported;", invalidContext)
		}
		if !capsValid {
			message += "Volume capabilities not supported;"
		}
		if !paramsValid {
			message += fmt.Sprintf("Volume parameters %v not supported;", invalidParams)
		}
		return &csi.ValidateVolumeCapabilitiesResponse{
			Message: message,
		}, nil
	}
}

// isValidVolumeContext returns nil if the volume context is valid or an array of invalid ones
func (d *Driver) isValidVolumeContext(volumeContext map[string]string) []string {
	invalidVolumeContext := []string{}
	for k := range volumeContext {
		if !validVolumeContext.Has(k) {
			invalidVolumeContext = append(invalidVolumeContext, k)
		}
	}
	if len(invalidVolumeContext) > 0 {
		return invalidVolumeContext
	}
	return nil
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

// isValidParameters returns nil if the parameters are valid or an array of invalid ones
func (d *Driver) isValidParameters(parameters map[string]string) []string {
	invalidParameters := []string{}
	for k := range parameters {
		if !validParameters.Has(k) {
			invalidParameters = append(invalidParameters, k)
		}
	}
	if len(invalidParameters) > 0 {
		return invalidParameters
	}
	return nil
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

func newCreateVolumeResponse(fs *cloud.FileSystem) *csi.CreateVolumeResponse {
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      fs.FileSystemId,
			CapacityBytes: util.GiBToBytes(fs.CapacityGiB),
			VolumeContext: map[string]string{
				"dnsname": fs.DnsName,
			},
		},
	}
}
