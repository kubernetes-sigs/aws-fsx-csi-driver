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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const fakeMaxCachedMB = `users: 1
max_cached_mb: 515000
used_mb: 0
unused_mb: 515000
reclaim_count: 0
`

func writeFakeLlite(t *testing.T, lliteDir, instance, content string) string {
	t.Helper()
	dir := filepath.Join(lliteDir, instance)
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "max_cached_mb")
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	assert.NoError(t, err)
	return string(b)
}

func TestNewLustreCacheLimiter(t *testing.T) {
	testCases := []struct {
		name        string
		requestedMB int
		totalRAMMB  int
		wantNil     bool
		wantLimitMB int
	}{
		{name: "disabled when zero", requestedMB: 0, totalRAMMB: 1048576, wantNil: true},
		{name: "requested limit on a large node", requestedMB: 102400, totalRAMMB: 1048576, wantLimitMB: 102400},
		{name: "half of RAM on a small node", requestedMB: 102400, totalRAMMB: 65536, wantLimitMB: 32768},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := newLustreCacheLimiter(t.TempDir(), tc.requestedMB, tc.totalRAMMB)
			if tc.wantNil {
				assert.Nil(t, l)
				return
			}
			assert.Equal(t, tc.wantLimitMB, l.limitMB)
		})
	}
}

// newTestLustreCacheLimiter resolves every path through instances, standing in
// for `lfs getname`.
func newTestLustreCacheLimiter(lliteDir string, instances map[string]string) *lustreCacheLimiter {
	l := newLustreCacheLimiter(lliteDir, 102400, 1048576)
	l.lliteInstance = func(path string) (string, error) {
		instance, ok := instances[path]
		if !ok {
			return "", fmt.Errorf("%s is not a Lustre mount", path)
		}
		return instance, nil
	}
	return l
}

func TestLustreCacheLimiterApplyLimitSetsOnlyTheTargetMount(t *testing.T) {
	lliteDir := t.TempDir()
	target := writeFakeLlite(t, lliteDir, "fsx-ffff0001", fakeMaxCachedMB)
	other := writeFakeLlite(t, lliteDir, "fsx-ffff0002", fakeMaxCachedMB)
	l := newTestLustreCacheLimiter(lliteDir, map[string]string{"/mnt/a": "fsx-ffff0001", "/mnt/b": "fsx-ffff0002"})

	err := l.applyLimit("/mnt/a")

	assert.NoError(t, err)
	assert.Equal(t, "102400\n", readFile(t, target))
	assert.Equal(t, fakeMaxCachedMB, readFile(t, other))
}

func TestLustreCacheLimiterApplyLimitSkipsMountAlreadyAtLimit(t *testing.T) {
	lliteDir := t.TempDir()
	content := "users: 1\nmax_cached_mb: 102400\nused_mb: 0\n"
	path := writeFakeLlite(t, lliteDir, "fsx-ffff0001", content)
	l := newTestLustreCacheLimiter(lliteDir, map[string]string{"/mnt/a": "fsx-ffff0001"})

	err := l.applyLimit("/mnt/a")

	assert.NoError(t, err)
	assert.Equal(t, content, readFile(t, path))
}

func TestLustreCacheLimiterApplyLimitErrors(t *testing.T) {
	testCases := []struct {
		name      string
		instances map[string]string
		llite     string
		wantErr   string
	}{
		{name: "target is not a Lustre mount", instances: map[string]string{}, wantErr: "not a Lustre mount"},
		{name: "instance has no llite directory", instances: map[string]string{"/mnt/a": "fsx-ffff0009"}, wantErr: "no such file"},
		{name: "instance is not a plain name", instances: map[string]string{"/mnt/a": "../fsx-ffff0001"}, wantErr: "unexpected llite instance"},
		{name: "max_cached_mb is unparsable", instances: map[string]string{"/mnt/a": "fsx-ffff0001"}, llite: "garbage\n", wantErr: "max_cached_mb"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lliteDir := t.TempDir()
			if tc.llite != "" {
				writeFakeLlite(t, lliteDir, "fsx-ffff0001", tc.llite)
			}

			err := newTestLustreCacheLimiter(lliteDir, tc.instances).applyLimit("/mnt/a")

			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestParseLfsGetname(t *testing.T) {
	testCases := []struct {
		name         string
		out          string
		wantInstance string
		wantErr      bool
	}{
		{name: "one mount", out: "wu3tbaev-ffff919a118d5000 /mnt/kubelet/pods/uid/volumes/kubernetes.io~csi/pv/mount\n", wantInstance: "wu3tbaev-ffff919a118d5000"},
		{name: "empty", out: "", wantErr: true},
		{name: "several mounts", out: "fsx-ffff0001 /mnt/a\nfsx-ffff0002 /mnt/b\n", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance, err := parseLfsGetname(tc.out)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantInstance, instance)
		})
	}
}

func TestTotalRAMMB(t *testing.T) {
	meminfo := filepath.Join(t.TempDir(), "meminfo")
	assert.NoError(t, os.WriteFile(meminfo, []byte("MemTotal:       1056964608 kB\nMemFree:        1000 kB\n"), 0o644))

	got, err := totalRAMMB(meminfo)

	assert.NoError(t, err)
	assert.Equal(t, 1032192, got)
}
