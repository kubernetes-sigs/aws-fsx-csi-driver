/*
Copyright 2018 The Kubernetes Authors.

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
	"testing"
)

func TestGiBToBytes(t *testing.T) {
	var sizeInGiB int64 = 3

	actual := GiBToBytes(sizeInGiB)
	if actual != 3*GiB {
		t.Fatalf("Wrong result for GiBToBytes. Got: %d", actual)
	}
}

func TestRoundUp3600GiB(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    3600,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    3600,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
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
			actual := RoundUp3600GiB(tc.sizeInBytes)
			if actual != tc.expected {
				t.Fatalf("RoundUp3600GiB got wrong result. actual: %d, expected: %d", actual, tc.expected)
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
