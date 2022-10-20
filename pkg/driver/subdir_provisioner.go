package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

type SubDirProvisioner struct {
	driver *Driver
}

var _ Provisioner = SubDirProvisioner{}

func (p SubDirProvisioner) Provision(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.Volume, error) {
	params := req.GetParameters()
	dnsname := params[volumeParamsDnsName]
	mountname := params[volumeParamsMountName]
	baseDir := strings.Trim(params[volumeParamsBaseDir], "/")

	if len(dnsname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "dnsname is not provided")
	}

	if len(mountname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mountname is not provided")
	}

	tempUUID := uuid.New().String()
	target := filepath.Join(tempMountPathPrefix, tempUUID)

	var volCap *csi.VolumeCapability

	if len(req.GetVolumeCapabilities()) > 0 {
		volCap = req.GetVolumeCapabilities()[0]
	} else {
		volCap = &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		}
	}

	// Mount dnsname@tcp:/mountname/baseDir to create subDir
	if _, err := p.driver.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
		VolumeId:         tempUUID,
		TargetPath:       target,
		VolumeCapability: volCap,
		VolumeContext: map[string]string{
			volumeContextDnsName:   dnsname,
			volumeContextMountName: mountname,
			volumeContextBaseDir:   baseDir,
		},
	}); err != nil {
		return nil, err
	}

	// Ensure to unmount dnsname@tcp:/mountname/baseDir at least once
	defer func() {
		if _, err := p.driver.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
			VolumeId:   tempUUID,
			TargetPath: target,
		}); err == nil {
			if err := os.RemoveAll(target); err != nil {
				klog.Warningf("Could not delete %q: %v", target, err)
			}
		} else {
			klog.Warningf("Could not unmount %q: %v", target, err)
		}
	}()

	// Create subDir (dnsname@tcp:/mountname/baseDir/subDir) under the tempMountPathPrefix/tempUUID
	volName := req.GetName()
	if err := p.driver.mounter.MakeDir(filepath.Join(target, volName)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.Volume{
		VolumeId:      fmt.Sprintf("%s:%s:%s:%s:%s", dnsname, mountname, baseDir, volName, volName),
		CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
		VolumeContext: map[string]string{
			volumeContextDnsName:   dnsname,
			volumeContextMountName: mountname,
			volumeContextBaseDir:   baseDir,
			volumeContextSubDir:    volName,
		},
	}, nil
}

func (p SubDirProvisioner) Delete(ctx context.Context, req *csi.DeleteVolumeRequest) error {
	volumeID := req.GetVolumeId()
	fsxVolume, err := getFsxVolumeFromVolumeID(volumeID)
	if err != nil {
		return err
	}

	if len(fsxVolume.dnsname) == 0 {
		return status.Error(codes.InvalidArgument, "dnsname is not provided")
	}

	if len(fsxVolume.mountname) == 0 {
		return status.Error(codes.InvalidArgument, "mountname is not provided")
	}

	tempUUID := uuid.New().String()
	target := filepath.Join(tempMountPathPrefix, tempUUID)

	volCap := &csi.VolumeCapability{
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}

	// Mount dnsname@tcp:/mountname/baseDir to delete subDir
	if _, err := p.driver.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
		VolumeId:         tempUUID,
		TargetPath:       target,
		VolumeCapability: volCap,
		VolumeContext: map[string]string{
			volumeContextDnsName:   fsxVolume.dnsname,
			volumeContextMountName: fsxVolume.mountname,
			volumeContextBaseDir:   fsxVolume.baseDir,
		},
	}); err != nil {
		return err
	}

	// Ensure to unmount dnsname@tcp:/mountname/baseDir at least once
	defer func() {
		if _, err := p.driver.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
			VolumeId:   tempUUID,
			TargetPath: target,
		}); err == nil {
			if err := os.RemoveAll(target); err != nil {
				klog.Warningf("Could not delete %q: %v", target, err)
			}
		} else {
			klog.Warningf("Could not unmount %q: %v", target, err)
		}
	}()

	// Delete subDir (dnsname@tcp:/mountname/baseDir/subDir) under the tempMountPathPrefix/tempUUID
	if err := os.RemoveAll(filepath.Join(target, fsxVolume.subDir)); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}
