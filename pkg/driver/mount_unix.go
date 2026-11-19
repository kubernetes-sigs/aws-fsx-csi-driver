//go:build linux || darwin

/*
Copyright 2026 The Kubernetes Authors.

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
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
)

func (m *NodeMounter) GetVolumeStats(volumePath string) ([]*csi.VolumeUsage, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(volumePath, &statfs); err != nil {
		return nil, err
	}
	return volumeUsage(&statfs), nil
}

// volumeUsage computes bytes and inodes with the kubelet's statfs arithmetic.
// Like the EBS driver, it omits inodes when the filesystem reports none.
// VolumeUsage takes int64. Bsize is int64 on linux/amd64 and
// linux/arm64 but uint32 on darwin, and the counts are uint64, so each field is
// converted.
func volumeUsage(statfs *unix.Statfs_t) []*csi.VolumeUsage {
	bsize := int64(statfs.Bsize)
	usage := []*csi.VolumeUsage{
		{
			Unit:      csi.VolumeUsage_BYTES,
			Total:     int64(statfs.Blocks) * bsize,
			Available: int64(statfs.Bavail) * bsize,
			Used:      (int64(statfs.Blocks) - int64(statfs.Bfree)) * bsize,
		},
	}
	if statfs.Files != 0 {
		usage = append(usage, &csi.VolumeUsage{
			Unit:      csi.VolumeUsage_INODES,
			Total:     int64(statfs.Files),
			Available: int64(statfs.Ffree),
			Used:      int64(statfs.Files) - int64(statfs.Ffree),
		})
	}
	return usage
}
