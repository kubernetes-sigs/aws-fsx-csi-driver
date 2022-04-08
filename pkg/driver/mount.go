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
	"k8s.io/utils/mount"
	"os"
)

// Mounter is an interface for mount operations
type Mounter interface {
	mount.Interface
	IsCorruptedMnt(err error) bool
	PathExists(path string) (bool, error)
	MakeDir(pathname string) error
}

type NodeMounter struct {
	mount.Interface
}

func newNodeMounter() Mounter {
	return &NodeMounter{
		Interface: mount.New(""),
	}
}

func (m *NodeMounter) MakeDir(pathname string) error {
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

//IsCorruptedMnt return true if err is about corrupted mount point
func (m *NodeMounter) IsCorruptedMnt(err error) bool {
	return mount.IsCorruptedMnt(err)
}

func (m *NodeMounter) PathExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
