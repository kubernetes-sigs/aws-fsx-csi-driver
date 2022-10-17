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
)

type FilesetProvisioner struct {
	mounter Mounter
}

var _ Provisioner = FilesetProvisioner{}

func (p FilesetProvisioner) Provision(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.Volume, error) {
	params := req.GetParameters()
	dnsname := params[volumeParamsDnsName]
	mountname := params[volumeParamsMountName]
	basefileset := strings.Trim(params[volumeParamsBaseFileset], "/")
	volName := req.GetName()

	if len(dnsname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "dnsname is not provided")
	}

	if len(mountname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "mountname is not provided")
	}

	source := fmt.Sprintf("%s@tcp:/%s", dnsname, mountname)
	target := filepath.Join(tempMountPathPrefix, uuid.New().String())
	mountOptions := []string{"flock"}

	if basefileset != "" {
		source = fmt.Sprintf("%s/%s", source, basefileset)
	}

	if err := p.mounter.MakeDir(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", target, err)
	}

	if err := p.mounter.Mount(source, target, "lustre", mountOptions); err != nil {
		os.Remove(target)
		return nil, status.Errorf(codes.Internal, "Could not mount %q at %q: %v", source, target, err)
	}

	if err := os.MkdirAll(filepath.Join(target, volName), os.FileMode(0o755)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := p.mounter.Unmount(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}

	if err := os.RemoveAll(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete %q: %v", target, err)
	}

	return &csi.Volume{
		VolumeId:      fmt.Sprintf("%s:%s:%s:%s:%s", dnsname, mountname, basefileset, volName, volName),
		CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
		VolumeContext: map[string]string{
			volumeContextDnsName:      dnsname,
			volumeContextMountName:    mountname,
			volumeContextBaseFileset:  basefileset,
			volumeContextMountFileset: volName,
		},
	}, nil
}

func (p FilesetProvisioner) Delete(ctx context.Context, req *csi.DeleteVolumeRequest) error {
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

	source := fmt.Sprintf("%s@tcp:/%s", fsxVolume.dnsname, fsxVolume.mountname)
	target := filepath.Join(tempMountPathPrefix, uuid.New().String())
	mountOptions := []string{"flock"}

	if fsxVolume.basefileset != "" {
		source = fmt.Sprintf("%s/%s", source, fsxVolume.basefileset)
	}

	if err := p.mounter.MakeDir(target); err != nil {
		return status.Errorf(codes.Internal, "Could not create dir %q: %v", target, err)
	}

	if err := p.mounter.Mount(source, target, "lustre", mountOptions); err != nil {
		os.Remove(target)
		return status.Errorf(codes.Internal, "Could not mount %q at %q: %v", source, target, err)
	}

	if err := os.RemoveAll(filepath.Join(target, fsxVolume.mountfileset)); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if err := p.mounter.Unmount(target); err != nil {
		return status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}

	if err := os.RemoveAll(target); err != nil {
		return status.Errorf(codes.Internal, "Could not delete %q: %v", target, err)
	}

	return nil
}
