package driver

import (
	"context"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const volumeIDSeparator = ":"

type Provisioner interface {
	Provision(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.Volume, error)
	Delete(ctx context.Context, req *csi.DeleteVolumeRequest) error
}

type FsxVolume struct {
	fsid        string
	dnsname     string
	mountname   string
	basefileset string
	uuid        string
}

// VolumeID form is expected one of the following
// filesystem provisioning:
// - fs-xxx (filesystem provisioning)
// fileset provisioning:
// - 10.x.x.x:mountname::uuid
// - 10.x.x.x:mountname:basefileset:uuid
func getFsxVolumeFromVolumeID(id string) (FsxVolume, error) {
	tokens := strings.Split(id, volumeIDSeparator)
	if len(tokens) == 1 {
		return FsxVolume{fsid: tokens[0]}, nil
	} else if len(tokens) != 4 {
		return FsxVolume{}, status.Errorf(codes.InvalidArgument, "Volume ID '%s' is invalid: Expected one or four fields separated by '%s'", id, volumeIDSeparator)
	}

	return FsxVolume{
		fsid:        "",
		dnsname:     tokens[0],
		mountname:   tokens[1],
		basefileset: tokens[2],
		uuid:        tokens[3],
	}, nil
}
