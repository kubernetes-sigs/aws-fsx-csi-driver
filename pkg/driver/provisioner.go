package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

const volumeIDSeparator = ":"

type Provisioner interface {
	Provision(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.Volume, error)
	Delete(ctx context.Context, req *csi.DeleteVolumeRequest) error
}
