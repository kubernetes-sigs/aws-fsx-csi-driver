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

package mocks

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
)

// IntelligentTieringTestHelper provides helper methods for testing INTELLIGENT_TIERING functionality
type IntelligentTieringTestHelper struct {
	LastCreateInput *fsx.CreateFileSystemInput
}

// NewIntelligentTieringTestHelper creates a new helper for INTELLIGENT_TIERING testing
func NewIntelligentTieringTestHelper() *IntelligentTieringTestHelper {
	return &IntelligentTieringTestHelper{}
}

// CaptureCreateInput stores the CreateFileSystemInput for later validation
func (h *IntelligentTieringTestHelper) CaptureCreateInput(input *fsx.CreateFileSystemInput) {
	h.LastCreateInput = input
}

// ValidateIntelligentTieringRequest validates that the CreateFileSystemInput is properly structured for INTELLIGENT_TIERING
func (h *IntelligentTieringTestHelper) ValidateIntelligentTieringRequest() error {
	if h.LastCreateInput == nil {
		return fmt.Errorf("no CreateFileSystemInput captured")
	}

	input := h.LastCreateInput

	// Validate StorageType is INTELLIGENT_TIERING
	if input.StorageType != types.StorageTypeIntelligentTiering {
		return fmt.Errorf("expected StorageType to be INTELLIGENT_TIERING, got: %v", input.StorageType)
	}

	// Validate StorageCapacity is NOT set for INTELLIGENT_TIERING
	if input.StorageCapacity != nil {
		return fmt.Errorf("StorageCapacity should not be set for INTELLIGENT_TIERING, got: %d", *input.StorageCapacity)
	}

	// Validate LustreConfiguration exists
	if input.LustreConfiguration == nil {
		return fmt.Errorf("LustreConfiguration is required for INTELLIGENT_TIERING")
	}

	lustreConfig := input.LustreConfiguration

	// Validate DeploymentType is PERSISTENT_2
	if lustreConfig.DeploymentType != types.LustreDeploymentTypePersistent2 {
		return fmt.Errorf("expected DeploymentType to be PERSISTENT_2, got: %v", lustreConfig.DeploymentType)
	}

	// Validate ThroughputCapacity is set
	if lustreConfig.ThroughputCapacity == nil {
		return fmt.Errorf("ThroughputCapacity is required for INTELLIGENT_TIERING")
	}

	// Validate ThroughputCapacity is a multiple of 4000
	if *lustreConfig.ThroughputCapacity%4000 != 0 {
		return fmt.Errorf("ThroughputCapacity must be a multiple of 4000, got: %d", *lustreConfig.ThroughputCapacity)
	}

	// Validate MetadataConfiguration exists and is USER_PROVISIONED
	if lustreConfig.MetadataConfiguration == nil {
		return fmt.Errorf("MetadataConfiguration is required for INTELLIGENT_TIERING")
	}

	if lustreConfig.MetadataConfiguration.Mode != types.MetadataConfigurationModeUserProvisioned {
		return fmt.Errorf("expected MetadataConfiguration.Mode to be USER_PROVISIONED, got: %v", lustreConfig.MetadataConfiguration.Mode)
	}

	// Validate MetadataConfiguration IOPS is 6000 or 12000
	if lustreConfig.MetadataConfiguration.Iops == nil {
		return fmt.Errorf("MetadataConfiguration.Iops is required for INTELLIGENT_TIERING")
	}

	iops := *lustreConfig.MetadataConfiguration.Iops
	if iops != 6000 && iops != 12000 {
		return fmt.Errorf("MetadataConfiguration.Iops must be 6000 or 12000, got: %d", iops)
	}

	// Validate DataReadCacheConfiguration exists
	if lustreConfig.DataReadCacheConfiguration == nil {
		return fmt.Errorf("DataReadCacheConfiguration is required for INTELLIGENT_TIERING")
	}

	// Validate DataReadCacheConfiguration SizingMode is valid
	sizingMode := lustreConfig.DataReadCacheConfiguration.SizingMode
	validModes := []types.LustreReadCacheSizingMode{
		types.LustreReadCacheSizingModeNoCache,
		types.LustreReadCacheSizingModeProportionalToThroughputCapacity,
		types.LustreReadCacheSizingModeUserProvisioned,
	}

	isValidMode := false
	for _, mode := range validModes {
		if sizingMode == mode {
			isValidMode = true
			break
		}
	}

	if !isValidMode {
		return fmt.Errorf("DataReadCacheConfiguration.SizingMode must be one of: NO_CACHE, PROPORTIONAL_TO_THROUGHPUT_CAPACITY, USER_PROVISIONED, got: %v", sizingMode)
	}

	// If USER_PROVISIONED, validate SizeGiB is set and >= 32
	if sizingMode == types.LustreReadCacheSizingModeUserProvisioned {
		if lustreConfig.DataReadCacheConfiguration.SizeGiB == nil {
			return fmt.Errorf("DataReadCacheConfiguration.SizeGiB is required when SizingMode is USER_PROVISIONED")
		}

		if *lustreConfig.DataReadCacheConfiguration.SizeGiB < 32 {
			return fmt.Errorf("DataReadCacheConfiguration.SizeGiB must be at least 32 GiB, got: %d", *lustreConfig.DataReadCacheConfiguration.SizeGiB)
		}
	}

	return nil
}

// GetThroughputCapacity returns the ThroughputCapacity from the last captured request
func (h *IntelligentTieringTestHelper) GetThroughputCapacity() (int32, error) {
	if h.LastCreateInput == nil || h.LastCreateInput.LustreConfiguration == nil || h.LastCreateInput.LustreConfiguration.ThroughputCapacity == nil {
		return 0, fmt.Errorf("ThroughputCapacity not found in captured request")
	}
	return *h.LastCreateInput.LustreConfiguration.ThroughputCapacity, nil
}

// GetMetadataIops returns the MetadataConfiguration IOPS from the last captured request
func (h *IntelligentTieringTestHelper) GetMetadataIops() (int32, error) {
	if h.LastCreateInput == nil || h.LastCreateInput.LustreConfiguration == nil || h.LastCreateInput.LustreConfiguration.MetadataConfiguration == nil || h.LastCreateInput.LustreConfiguration.MetadataConfiguration.Iops == nil {
		return 0, fmt.Errorf("MetadataConfiguration.Iops not found in captured request")
	}
	return *h.LastCreateInput.LustreConfiguration.MetadataConfiguration.Iops, nil
}

// GetDataReadCacheSizingMode returns the DataReadCacheConfiguration SizingMode from the last captured request
func (h *IntelligentTieringTestHelper) GetDataReadCacheSizingMode() (types.LustreReadCacheSizingMode, error) {
	if h.LastCreateInput == nil || h.LastCreateInput.LustreConfiguration == nil || h.LastCreateInput.LustreConfiguration.DataReadCacheConfiguration == nil {
		return "", fmt.Errorf("DataReadCacheConfiguration.SizingMode not found in captured request")
	}
	return h.LastCreateInput.LustreConfiguration.DataReadCacheConfiguration.SizingMode, nil
}

// GetDataReadCacheSizeGiB returns the DataReadCacheConfiguration SizeGiB from the last captured request
func (h *IntelligentTieringTestHelper) GetDataReadCacheSizeGiB() (int32, error) {
	if h.LastCreateInput == nil || h.LastCreateInput.LustreConfiguration == nil || h.LastCreateInput.LustreConfiguration.DataReadCacheConfiguration == nil || h.LastCreateInput.LustreConfiguration.DataReadCacheConfiguration.SizeGiB == nil {
		return 0, fmt.Errorf("DataReadCacheConfiguration.SizeGiB not found in captured request")
	}
	return *h.LastCreateInput.LustreConfiguration.DataReadCacheConfiguration.SizeGiB, nil
}
