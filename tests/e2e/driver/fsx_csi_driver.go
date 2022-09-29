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
	"fmt"

	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	fsxcsidriver "sigs.k8s.io/aws-fsx-csi-driver/pkg/driver"
)

// Implement PVTestDriver interface
type fsxCSIDriver struct {
	driverName string
}

// InitFSxCSIDriver returns fsxCSIDriver that implements DynamicPVTestDriver interface
func InitFSxCSIDriver() PVTestDriver {
	return &fsxCSIDriver{
		driverName: fsxcsidriver.DriverName,
	}
}

func (d *fsxCSIDriver) GetDynamicProvisionStorageClass(parameters map[string]string, mountOptions []string, reclaimPolicy *v1.PersistentVolumeReclaimPolicy, bindingMode *storagev1.VolumeBindingMode, allowedTopologyValues []string, namespace string) *storagev1.StorageClass {
	provisioner := d.driverName
	generateName := fmt.Sprintf("%s-%s-dynamic-sc-", namespace, provisioner)
	allowedTopologies := []v1.TopologySelectorTerm{}
	return getStorageClass(generateName, provisioner, parameters, mountOptions, reclaimPolicy, bindingMode, allowedTopologies)
}
