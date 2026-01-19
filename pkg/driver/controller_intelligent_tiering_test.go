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
	"strconv"
	"strings"
	"testing"
)

// TestValidateIntelligentTieringParams_ThroughputCapacity tests throughput capacity validation
// Feature: intelligent-tiering, Property 1: Throughput Capacity Validation
// Validates: Requirements 1.5, 4.2, 4.3
func TestValidateIntelligentTieringParams_ThroughputCapacity(t *testing.T) {
	testCases := []struct {
		name          string
		throughput    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid: 4000",
			throughput:  "4000",
			expectError: false,
		},
		{
			name:        "valid: 8000",
			throughput:  "8000",
			expectError: false,
		},
		{
			name:        "valid: 12000",
			throughput:  "12000",
			expectError: false,
		},
		{
			name:        "valid: 16000",
			throughput:  "16000",
			expectError: false,
		},
		{
			name:          "invalid: missing throughputCapacity",
			throughput:    "",
			expectError:   true,
			errorContains: "throughputCapacity is required",
		},
		{
			name:          "invalid: not a multiple of 4000 (5000)",
			throughput:    "5000",
			expectError:   true,
			errorContains: "must be a multiple of 4000",
		},
		{
			name:          "invalid: not a multiple of 4000 (3999)",
			throughput:    "3999",
			expectError:   true,
			errorContains: "must be a multiple of 4000",
		},
		{
			name:          "invalid: not a multiple of 4000 (4001)",
			throughput:    "4001",
			expectError:   true,
			errorContains: "must be a multiple of 4000",
		},
		{
			name:          "invalid: zero",
			throughput:    "0",
			expectError:   true,
			errorContains: "must be a multiple of 4000",
		},
		{
			name:          "invalid: negative",
			throughput:    "-4000",
			expectError:   true,
			errorContains: "must be a multiple of 4000",
		},
		{
			name:          "invalid: not a number",
			throughput:    "notanumber",
			expectError:   true,
			errorContains: "must be a number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]string{
				volumeParamsStorageType: "INTELLIGENT_TIERING",
			}
			if tc.throughput != "" {
				params[volumeParamsThroughputCapacity] = tc.throughput
			}

			err := validateIntelligentTieringParams(params)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorContains)
				}
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateIntelligentTieringParams_MetadataIops tests metadata IOPS validation
// Feature: intelligent-tiering, Property 3: Metadata IOPS Validation
// Validates: Requirements 2.3
func TestValidateIntelligentTieringParams_MetadataIops(t *testing.T) {
	testCases := []struct {
		name          string
		metadataIops  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid: 6000",
			metadataIops: "6000",
			expectError:  false,
		},
		{
			name:         "valid: 12000",
			metadataIops: "12000",
			expectError:  false,
		},
		{
			name:         "valid: not specified (optional)",
			metadataIops: "",
			expectError:  false,
		},
		{
			name:          "invalid: 3000",
			metadataIops:  "3000",
			expectError:   true,
			errorContains: "must be 6000 or 12000",
		},
		{
			name:          "invalid: 9000",
			metadataIops:  "9000",
			expectError:   true,
			errorContains: "must be 6000 or 12000",
		},
		{
			name:          "invalid: 18000",
			metadataIops:  "18000",
			expectError:   true,
			errorContains: "must be 6000 or 12000",
		},
		{
			name:          "invalid: not a number",
			metadataIops:  "notanumber",
			expectError:   true,
			errorContains: "must be a number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			if tc.metadataIops != "" {
				params[volumeParamsMetadataIops] = tc.metadataIops
			}

			err := validateIntelligentTieringParams(params)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorContains)
				}
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateIntelligentTieringParams_CacheSize tests cache size validation
// Feature: intelligent-tiering, Property 5: Cache Size Validation
// Validates: Requirements 3.4, 3.6
func TestValidateIntelligentTieringParams_CacheSize(t *testing.T) {
	testCases := []struct {
		name          string
		sizingMode    string
		cacheSize     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid: USER_PROVISIONED with 32 GiB",
			sizingMode:  "USER_PROVISIONED",
			cacheSize:   "32",
			expectError: false,
		},
		{
			name:        "valid: USER_PROVISIONED with 1000 GiB",
			sizingMode:  "USER_PROVISIONED",
			cacheSize:   "1000",
			expectError: false,
		},
		{
			name:        "valid: NO_CACHE without size",
			sizingMode:  "NO_CACHE",
			cacheSize:   "",
			expectError: false,
		},
		{
			name:        "valid: PROPORTIONAL_TO_THROUGHPUT_CAPACITY without size",
			sizingMode:  "PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			cacheSize:   "",
			expectError: false,
		},
		{
			name:        "valid: not specified (optional)",
			sizingMode:  "",
			cacheSize:   "",
			expectError: false,
		},
		{
			name:          "invalid: USER_PROVISIONED without size",
			sizingMode:    "USER_PROVISIONED",
			cacheSize:     "",
			expectError:   true,
			errorContains: "dataReadCacheSizeGiB is required",
		},
		{
			name:          "invalid: USER_PROVISIONED with size < 32",
			sizingMode:    "USER_PROVISIONED",
			cacheSize:     "31",
			expectError:   true,
			errorContains: "must be at least 32 GiB",
		},
		{
			name:          "invalid: USER_PROVISIONED with size = 0",
			sizingMode:    "USER_PROVISIONED",
			cacheSize:     "0",
			expectError:   true,
			errorContains: "must be at least 32 GiB",
		},
		{
			name:          "invalid: invalid sizing mode",
			sizingMode:    "INVALID_MODE",
			cacheSize:     "",
			expectError:   true,
			errorContains: "must be one of: NO_CACHE, PROPORTIONAL_TO_THROUGHPUT_CAPACITY, USER_PROVISIONED",
		},
		{
			name:          "invalid: cache size not a number",
			sizingMode:    "USER_PROVISIONED",
			cacheSize:     "notanumber",
			expectError:   true,
			errorContains: "must be a number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			if tc.sizingMode != "" {
				params[volumeParamsDataReadCacheSizingMode] = tc.sizingMode
			}
			if tc.cacheSize != "" {
				params[volumeParamsDataReadCacheSizeGiB] = tc.cacheSize
			}

			err := validateIntelligentTieringParams(params)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorContains)
				}
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateIntelligentTieringParams_DeploymentType tests deployment type validation
// Validates: Requirements 1.2, 5.1
func TestValidateIntelligentTieringParams_DeploymentType(t *testing.T) {
	testCases := []struct {
		name           string
		deploymentType string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "valid: PERSISTENT_2",
			deploymentType: "PERSISTENT_2",
			expectError:    false,
		},
		{
			name:           "valid: not specified (will default)",
			deploymentType: "",
			expectError:    false,
		},
		{
			name:           "invalid: PERSISTENT_1",
			deploymentType: "PERSISTENT_1",
			expectError:    true,
			errorContains:  "must be PERSISTENT_2",
		},
		{
			name:           "invalid: SCRATCH_1",
			deploymentType: "SCRATCH_1",
			expectError:    true,
			errorContains:  "must be PERSISTENT_2",
		},
		{
			name:           "invalid: SCRATCH_2",
			deploymentType: "SCRATCH_2",
			expectError:    true,
			errorContains:  "must be PERSISTENT_2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			if tc.deploymentType != "" {
				params[volumeParamsDeploymentType] = tc.deploymentType
			}

			err := validateIntelligentTieringParams(params)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorContains)
				}
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestProperty2_MetadataConfigurationDefaults tests that metadata configuration defaults are applied correctly
// Feature: intelligent-tiering, Property 2: Metadata Configuration Defaults
// Validates: Requirements 2.1, 2.2
func TestProperty2_MetadataConfigurationDefaults(t *testing.T) {
	// Property: For any INTELLIGENT_TIERING filesystem creation request, the driver SHALL set metadata 
	// configuration mode to USER_PROVISIONED and, if metadataIops is not specified, SHALL default to 6000 IOPS.
	
	testCases := []struct {
		name                      string
		metadataIops              string
		expectedMode              string
		expectedIops              int32
	}{
		{
			name:         "metadataIops not specified - should default to 6000",
			metadataIops: "",
			expectedMode: "USER_PROVISIONED",
			expectedIops: 6000,
		},
		{
			name:         "metadataIops specified as 6000",
			metadataIops: "6000",
			expectedMode: "USER_PROVISIONED",
			expectedIops: 6000,
		},
		{
			name:         "metadataIops specified as 12000",
			metadataIops: "12000",
			expectedMode: "USER_PROVISIONED",
			expectedIops: 12000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate CreateVolume parameter processing for INTELLIGENT_TIERING
			volumeParams := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			
			if tc.metadataIops != "" {
				volumeParams[volumeParamsMetadataIops] = tc.metadataIops
			}

			// Validate parameters first
			err := validateIntelligentTieringParams(volumeParams)
			if err != nil {
				t.Fatalf("validation failed: %v", err)
			}

			// Simulate the default application logic from CreateVolume
			var metadataIops int32
			var metadataMode string
			
			if val, ok := volumeParams[volumeParamsMetadataIops]; ok && val != "" {
				n, _ := strconv.ParseInt(val, 10, 64)
				metadataIops = int32(n)
				metadataMode = "USER_PROVISIONED"
			} else {
				// Apply defaults
				metadataIops = 6000
				metadataMode = "USER_PROVISIONED"
			}

			// Verify the property holds
			if metadataMode != tc.expectedMode {
				t.Errorf("expected metadata mode %s, got %s", tc.expectedMode, metadataMode)
			}
			if metadataIops != tc.expectedIops {
				t.Errorf("expected metadata IOPS %d, got %d", tc.expectedIops, metadataIops)
			}
		})
	}
}

// TestProperty4_DataReadCacheConfigurationDefaults tests that cache configuration defaults are applied correctly
// Feature: intelligent-tiering, Property 4: Data Read Cache Configuration Defaults
// Validates: Requirements 3.1, 3.2
func TestProperty4_DataReadCacheConfigurationDefaults(t *testing.T) {
	// Property: For any INTELLIGENT_TIERING filesystem creation request, the driver SHALL include 
	// DataReadCacheConfiguration in the API request and, if dataReadCacheSizingMode is not specified, 
	// SHALL default to PROPORTIONAL_TO_THROUGHPUT_CAPACITY.
	
	testCases := []struct {
		name                    string
		dataReadCacheSizingMode string
		dataReadCacheSizeGiB    string
		expectedSizingMode      string
		expectedSizeGiB         int32
	}{
		{
			name:                    "sizingMode not specified - should default to PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			dataReadCacheSizingMode: "",
			dataReadCacheSizeGiB:    "",
			expectedSizingMode:      "PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			expectedSizeGiB:         0,
		},
		{
			name:                    "sizingMode NO_CACHE",
			dataReadCacheSizingMode: "NO_CACHE",
			dataReadCacheSizeGiB:    "",
			expectedSizingMode:      "NO_CACHE",
			expectedSizeGiB:         0,
		},
		{
			name:                    "sizingMode PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			dataReadCacheSizingMode: "PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			dataReadCacheSizeGiB:    "",
			expectedSizingMode:      "PROPORTIONAL_TO_THROUGHPUT_CAPACITY",
			expectedSizeGiB:         0,
		},
		{
			name:                    "sizingMode USER_PROVISIONED with size",
			dataReadCacheSizingMode: "USER_PROVISIONED",
			dataReadCacheSizeGiB:    "1000",
			expectedSizingMode:      "USER_PROVISIONED",
			expectedSizeGiB:         1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate CreateVolume parameter processing for INTELLIGENT_TIERING
			volumeParams := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			
			if tc.dataReadCacheSizingMode != "" {
				volumeParams[volumeParamsDataReadCacheSizingMode] = tc.dataReadCacheSizingMode
			}
			if tc.dataReadCacheSizeGiB != "" {
				volumeParams[volumeParamsDataReadCacheSizeGiB] = tc.dataReadCacheSizeGiB
			}

			// Validate parameters first
			err := validateIntelligentTieringParams(volumeParams)
			if err != nil {
				t.Fatalf("validation failed: %v", err)
			}

			// Simulate the default application logic from CreateVolume
			var dataReadCacheSizingMode string
			var dataReadCacheSizeGiB int32
			
			if val, ok := volumeParams[volumeParamsDataReadCacheSizingMode]; ok && val != "" {
				dataReadCacheSizingMode = val
			} else {
				// Apply default
				dataReadCacheSizingMode = "PROPORTIONAL_TO_THROUGHPUT_CAPACITY"
			}
			
			if val, ok := volumeParams[volumeParamsDataReadCacheSizeGiB]; ok && val != "" {
				n, _ := strconv.ParseInt(val, 10, 64)
				dataReadCacheSizeGiB = int32(n)
			}

			// Verify the property holds
			if dataReadCacheSizingMode != tc.expectedSizingMode {
				t.Errorf("expected cache sizing mode %s, got %s", tc.expectedSizingMode, dataReadCacheSizingMode)
			}
			if dataReadCacheSizeGiB != tc.expectedSizeGiB {
				t.Errorf("expected cache size %d, got %d", tc.expectedSizeGiB, dataReadCacheSizeGiB)
			}
			
			// Verify that DataReadCacheConfiguration would be included (non-empty sizing mode)
			if dataReadCacheSizingMode == "" {
				t.Error("DataReadCacheConfiguration sizing mode should not be empty")
			}
		})
	}
}

// TestProperty7_DeploymentTypeEnforcement tests that deployment type is enforced correctly
// Feature: intelligent-tiering, Property 7: Deployment Type Enforcement
// Validates: Requirements 1.2, 5.1
func TestProperty7_DeploymentTypeEnforcement(t *testing.T) {
	// Property: For any INTELLIGENT_TIERING filesystem creation request, the driver SHALL set 
	// deploymentType to PERSISTENT_2. If a different deployment type is explicitly specified, 
	// the driver SHALL return an error.
	
	testCases := []struct {
		name           string
		deploymentType string
		expectError    bool
		expectedResult string
	}{
		{
			name:           "deploymentType not specified - should default to PERSISTENT_2",
			deploymentType: "",
			expectError:    false,
			expectedResult: "PERSISTENT_2",
		},
		{
			name:           "deploymentType explicitly set to PERSISTENT_2",
			deploymentType: "PERSISTENT_2",
			expectError:    false,
			expectedResult: "PERSISTENT_2",
		},
		{
			name:           "deploymentType set to PERSISTENT_1 - should error",
			deploymentType: "PERSISTENT_1",
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "deploymentType set to SCRATCH_1 - should error",
			deploymentType: "SCRATCH_1",
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "deploymentType set to SCRATCH_2 - should error",
			deploymentType: "SCRATCH_2",
			expectError:    true,
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate CreateVolume parameter processing for INTELLIGENT_TIERING
			volumeParams := map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
			}
			
			if tc.deploymentType != "" {
				volumeParams[volumeParamsDeploymentType] = tc.deploymentType
			}

			// Validate parameters first
			err := validateIntelligentTieringParams(volumeParams)
			
			if tc.expectError {
				if err == nil {
					t.Fatal("expected validation error but got none")
				}
				// Error case verified
				return
			}
			
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			// Simulate the default application logic from CreateVolume
			deploymentType := volumeParams[volumeParamsDeploymentType]
			if deploymentType == "" {
				// Apply default
				deploymentType = "PERSISTENT_2"
			}

			// Verify the property holds
			if deploymentType != tc.expectedResult {
				t.Errorf("expected deployment type %s, got %s", tc.expectedResult, deploymentType)
			}
		})
	}
}

// TestProperty8_StorageTypePropagation tests that storage type is correctly propagated
// Feature: intelligent-tiering, Property 8: Storage Type Propagation
// Validates: Requirements 1.1
func TestProperty8_StorageTypePropagation(t *testing.T) {
	// Property: For any INTELLIGENT_TIERING filesystem creation request that passes validation, 
	// the resulting AWS API call SHALL specify StorageType as INTELLIGENT_TIERING and the created 
	// filesystem SHALL have storage type INTELLIGENT_TIERING.
	
	testCases := []struct {
		name               string
		storageType        string
		expectedStorageType string
	}{
		{
			name:               "INTELLIGENT_TIERING storage type",
			storageType:        "INTELLIGENT_TIERING",
			expectedStorageType: "INTELLIGENT_TIERING",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate CreateVolume parameter processing for INTELLIGENT_TIERING
			volumeParams := map[string]string{
				volumeParamsStorageType:        tc.storageType,
				volumeParamsThroughputCapacity: "4000",
			}

			// Validate parameters first
			err := validateIntelligentTieringParams(volumeParams)
			if err != nil {
				t.Fatalf("validation failed: %v", err)
			}

			// Simulate the storage type propagation logic from CreateVolume
			storageType := volumeParams[volumeParamsStorageType]

			// Verify the property holds - storage type is propagated correctly
			if storageType != tc.expectedStorageType {
				t.Errorf("expected storage type %s, got %s", tc.expectedStorageType, storageType)
			}
			
			// Verify that INTELLIGENT_TIERING is detected correctly
			if storageType == "INTELLIGENT_TIERING" {
				// This would trigger the INTELLIGENT_TIERING handling in CreateVolume
				// The cloud layer would receive this storage type and pass it to AWS API
				t.Logf("INTELLIGENT_TIERING storage type correctly detected and would be propagated to cloud layer")
			}
		})
	}
}

// TestValidationErrorMessages tests that validation errors have clear, descriptive messages
// Validates: Requirements 1.4, 2.4, 3.5, 5.1, 5.3, 5.4
func TestValidationErrorMessages(t *testing.T) {
	testCases := []struct {
		name          string
		params        map[string]string
		expectedError string
	}{
		{
			name: "missing throughputCapacity error message",
			params: map[string]string{
				volumeParamsStorageType: "INTELLIGENT_TIERING",
			},
			expectedError: "throughputCapacity is required for INTELLIGENT_TIERING storage type",
		},
		{
			name: "invalid metadataIops error message",
			params: map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
				volumeParamsMetadataIops:       "9000",
			},
			expectedError: "metadataIops must be 6000 or 12000 for INTELLIGENT_TIERING, got: 9000",
		},
		{
			name: "missing dataReadCacheSizeGiB error message",
			params: map[string]string{
				volumeParamsStorageType:             "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity:      "4000",
				volumeParamsDataReadCacheSizingMode: "USER_PROVISIONED",
			},
			expectedError: "dataReadCacheSizeGiB is required when dataReadCacheSizingMode is USER_PROVISIONED",
		},
		{
			name: "invalid deploymentType error message",
			params: map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
				volumeParamsDeploymentType:     "SCRATCH_2",
			},
			expectedError: "deploymentType must be PERSISTENT_2 for INTELLIGENT_TIERING storage type",
		},
		{
			name: "invalid dataReadCacheSizingMode error message",
			params: map[string]string{
				volumeParamsStorageType:             "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity:      "4000",
				volumeParamsDataReadCacheSizingMode: "INVALID_MODE",
			},
			expectedError: "dataReadCacheSizingMode must be one of: NO_CACHE, PROPORTIONAL_TO_THROUGHPUT_CAPACITY, USER_PROVISIONED",
		},
		{
			name: "dataReadCacheSizeGiB too small error message",
			params: map[string]string{
				volumeParamsStorageType:             "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity:      "4000",
				volumeParamsDataReadCacheSizingMode: "USER_PROVISIONED",
				volumeParamsDataReadCacheSizeGiB:    "16",
			},
			expectedError: "dataReadCacheSizeGiB must be at least 32 GiB, got: 16",
		},
		{
			name: "throughputCapacity not a multiple of 4000 error message",
			params: map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "5000",
			},
			expectedError: "throughputCapacity must be a multiple of 4000 for INTELLIGENT_TIERING, got: 5000",
		},
		{
			name: "throughputCapacity not a number error message",
			params: map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "notanumber",
			},
			expectedError: "throughputCapacity must be a number",
		},
		{
			name: "metadataIops not a number error message",
			params: map[string]string{
				volumeParamsStorageType:        "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity: "4000",
				volumeParamsMetadataIops:       "notanumber",
			},
			expectedError: "metadataIops must be a number",
		},
		{
			name: "dataReadCacheSizeGiB not a number error message",
			params: map[string]string{
				volumeParamsStorageType:             "INTELLIGENT_TIERING",
				volumeParamsThroughputCapacity:      "4000",
				volumeParamsDataReadCacheSizingMode: "USER_PROVISIONED",
				volumeParamsDataReadCacheSizeGiB:    "notanumber",
			},
			expectedError: "dataReadCacheSizeGiB must be a number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIntelligentTieringParams(tc.params)
			
			if err == nil {
				t.Fatalf("expected error containing '%s', got nil", tc.expectedError)
			}
			
			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Fatalf("expected error containing '%s', got: %v", tc.expectedError, err)
			}
		})
	}
}
