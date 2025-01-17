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
	"math"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
)

func TestGiBToBytes(t *testing.T) {
	var sizeInGiB int32 = 3

	actual := GiBToBytes(sizeInGiB)
	if actual != 3*GiB {
		t.Fatalf("Wrong result for GiBToBytes. Got: %d", actual)
	}
}

func TestRoundUpVolumeSizeEmptyOrScratch1DeploymentType(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    1200,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 1000 Gib",
			sizeInBytes: 1000 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
			expected:    2400,
		},
		{
			name:        "Roundup 3000 Gib",
			sizeInBytes: 3000 * GiB,
			expected:    3600,
		},
		{
			name:        "Roundup 3600 Gib",
			sizeInBytes: 3600 * GiB,
			expected:    3600,
		},
		{
			name:        "Roundup 3600 Gib + 1 Byte",
			sizeInBytes: 3600*GiB + 1,
			expected:    7200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes, "", "", 0)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes, string(types.LustreDeploymentTypeScratch1), "", 0)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}
}

func TestRoundUpVolumeSizeOtherDeploymentType(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    1200,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 1000 Gib",
			sizeInBytes: 1000 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
			expected:    2400,
		},
		{
			name:        "Roundup 2400 Gib",
			sizeInBytes: 2400 * GiB,
			expected:    2400,
		},
		{
			name:        "Roundup 2400 Gib + 1 Byte",
			sizeInBytes: 2400*GiB + 1,
			expected:    4800,
		},
		{
			name:        "Roundup 3600 Gib",
			sizeInBytes: 3600 * GiB,
			expected:    4800,
		},
		{
			name:        "Roundup 4800 Gib",
			sizeInBytes: 4800 * GiB,
			expected:    4800,
		},
		{
			name:        "Roundup 4800 Gib + 1 Byte",
			sizeInBytes: 4800*GiB + 1,
			expected:    7200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes, string(types.LustreDeploymentTypeScratch2), "", 0)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}
}

func TestRoundUpVolumeSizeHddStorageType12Throughput(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    6000,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    6000,
		},
		{
			name:        "Roundup 1000 Gib",
			sizeInBytes: 1000 * GiB,
			expected:    6000,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
			expected:    6000,
		},
		{
			name:        "Roundup 6000 Gib",
			sizeInBytes: 6000 * GiB,
			expected:    6000,
		},
		{
			name:        "Roundup 6000 Gib + 1 Byte",
			sizeInBytes: 6000*GiB + 1,
			expected:    12000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes, string(types.LustreDeploymentTypePersistent1), string(types.StorageTypeHdd), 12)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}
}

func TestRoundUpVolumeSizeHddStorageType40Throughput(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    1800,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    1800,
		},
		{
			name:        "Roundup 1000 Gib",
			sizeInBytes: 1000 * GiB,
			expected:    1800,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
			expected:    3600,
		},
		{
			name:        "Roundup 6000 Gib",
			sizeInBytes: 6000 * GiB,
			expected:    7200,
		},
		{
			name:        "Roundup 6000 Gib + 1 Byte",
			sizeInBytes: 6000*GiB + 1,
			expected:    7200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes, string(types.LustreDeploymentTypePersistent1), string(types.StorageTypeHdd), 40)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}
}

func TestGetURLHost(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GetURLHost host without path",
			url:      "s3://fs-s3-data-repo",
			expected: "fs-s3-data-repo",
		},
		{
			name:     "GetURLHost standard url",
			url:      "s3://fs-s3-data-repo/import-path",
			expected: "fs-s3-data-repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _ := GetURLHost(tc.url)

			if actual != tc.expected {
				t.Fatalf("GetURLHost got wrong result. actual: %s, expected: %s", actual, tc.expected)
			}
		})
	}
}

func TestConvertToInt32(t *testing.T) {
	testCases := []struct {
		name        string
		input       int64
		expected    int32
		expectedErr error
	}{
		{
			name:        "converts okay",
			input:       100,
			expected:    100,
			expectedErr: nil,
		},
		{
			name:        "overflow case",
			input:       math.MaxInt32 + 1,
			expected:    0,
			expectedErr: fmt.Errorf("volume size %d would overflow int32", math.MaxInt32+1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ConvertToInt32(tc.input)

			if actual != tc.expected {
				t.Fatalf("GetURLHost got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}

			if tc.expectedErr != nil {
				if tc.expectedErr.Error() != err.Error() {
					t.Fatalf("GetURLHost got wrong result. actual: %d, expected: %d", err, tc.expectedErr)
				}
			}
		})
	}
}

type TestRequest struct {
	Name    string
	Secrets map[string]string
}

func TestSanitizeRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      interface{}
		expected interface{}
	}{
		{
			name: "Request with Secrets",
			req: &TestRequest{
				Name: "Test",
				Secrets: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			expected: &TestRequest{
				Name:    "Test",
				Secrets: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeRequest(tt.req)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SanitizeRequest() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
