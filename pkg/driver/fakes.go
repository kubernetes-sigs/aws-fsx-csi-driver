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
	"k8s.io/mount-utils"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/internal"
)

func NewFakeMounter() Mounter {
	return &NodeMounter{
		Interface: &mount.FakeMounter{
			MountPoints: []mount.MountPoint{},
		},
	}
}

// NewFakeDriver creates a new mock driver used for testing
func NewFakeDriver(endpoint string) *Driver {
	cloud := cloud.NewFakeCloudProvider()
	return &Driver{
		endpoint: endpoint,
		nodeID:   cloud.GetMetadata().GetInstanceID(),
		cloud:    cloud,
		inFlight: internal.NewInFlight(),
		mounter:  NewFakeMounter(),
	}
}
