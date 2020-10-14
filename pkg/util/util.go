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

package util

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/fsx"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	GiB = 1024 * 1024 * 1024
)

// RoundUpVolumeSize rounds the volume size in bytes up to
// 1200 GiB, 2400 GiB, or multiples of 3600 GiB for DeploymentType SCRATCH_1,
// to 1200 GiB or multiples of 2400 GiB for DeploymentType SCRATCH_2 or for
// DeploymentType PERSISTENT_1 and StorageType SSD, multiples of 6000 GiB for
// DeploymentType PERSISTENT_1, StorageType HDD, and PerUnitStorageThroughput 12,
// and multiples of 1800 GiB for DeploymentType PERSISTENT_1, StorageType HDD, and
// PerUnitStorageThroughput 40.
func RoundUpVolumeSize(volumeSizeBytes int64, deploymentType string, storageType string, perUnitStorageThroughput int64) int64 {
	if storageType == fsx.StorageTypeHdd {
		if perUnitStorageThroughput == 12 {
			return roundUpSize(volumeSizeBytes, 6000*GiB) * 6000
		} else {
			return roundUpSize(volumeSizeBytes, 1800*GiB) * 1800
		}
	} else {
		if deploymentType == fsx.LustreDeploymentTypeScratch1 ||
			deploymentType == "" {
			if volumeSizeBytes < 3600*GiB {
				return roundUpSize(volumeSizeBytes, 1200*GiB) * 1200
			} else {
				return roundUpSize(volumeSizeBytes, 3600*GiB) * 3600
			}
		} else {
			if volumeSizeBytes < 2400*GiB {
				return roundUpSize(volumeSizeBytes, 1200*GiB) * 1200
			} else {
				return roundUpSize(volumeSizeBytes, 2400*GiB) * 2400
			}
		}
	}
}

// GiBToBytes converts GiB to Bytes
func GiBToBytes(volumeSizeGiB int64) int64 {
	return volumeSizeGiB * GiB
}

func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("could not parse endpoint: %v", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("could not remove unix domain socket %q: %v", addr, err)
		}
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", scheme)
	}

	return scheme, addr, nil
}

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

// GetURLHost returns hostname  of given url
func GetURLHost(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)

	if err != nil {
		return "", fmt.Errorf("Could not parse url: %v", err)
	}

	return u.Host, nil
}
