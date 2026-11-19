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
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
)

func TestVolumeUsage(t *testing.T) {
	testCases := []struct {
		name   string
		statfs unix.Statfs_t
		want   []*csi.VolumeUsage
	}{
		{
			name:   "bytes and inodes",
			statfs: unix.Statfs_t{Bsize: 4096, Blocks: 1000, Bfree: 400, Bavail: 300, Files: 100, Ffree: 60},
			want: []*csi.VolumeUsage{
				{Unit: csi.VolumeUsage_BYTES, Total: 4096000, Available: 1228800, Used: 2457600},
				{Unit: csi.VolumeUsage_INODES, Total: 100, Available: 60, Used: 40},
			},
		},
		{
			name:   "no inodes when the filesystem reports none",
			statfs: unix.Statfs_t{Bsize: 4096, Blocks: 1000, Bfree: 400, Bavail: 300},
			want: []*csi.VolumeUsage{
				{Unit: csi.VolumeUsage_BYTES, Total: 4096000, Available: 1228800, Used: 2457600},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := volumeUsage(&tc.statfs)
			if len(got) != len(tc.want) {
				t.Fatalf("Expected %d usage entries, got %v", len(tc.want), got)
			}
			for i, want := range tc.want {
				if got[i].GetUnit() != want.GetUnit() || got[i].GetTotal() != want.GetTotal() ||
					got[i].GetAvailable() != want.GetAvailable() || got[i].GetUsed() != want.GetUsed() {
					t.Fatalf("Expected usage %v, got %v", want, got[i])
				}
			}
		})
	}
}
