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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

// lustreLliteDir is where the Lustre client exposes per-mount llite tunables.
// It needs the host's debugfs mounted into the node container.
const lustreLliteDir = "/sys/kernel/debug/lustre/llite"

// lustreCacheLimiter caps the Lustre client read cache (llite max_cached_mb).
// Every new mount starts at the Lustre default of half of RAM; on large-RAM
// nodes, evicting a cache that size can stall Lustre long enough to time out
// the workload.
type lustreCacheLimiter struct {
	lliteDir string
	limitMB  int
	// lliteInstance returns the llite instance name of the Lustre mount at a path.
	lliteInstance func(path string) (string, error)
}

// newLustreCacheLimiter returns nil when requestedMB is not positive. The limit
// never exceeds half of RAM, the Lustre default.
func newLustreCacheLimiter(lliteDir string, requestedMB, totalRAMMB int) *lustreCacheLimiter {
	if requestedMB <= 0 {
		return nil
	}
	return &lustreCacheLimiter{
		lliteDir:      lliteDir,
		limitMB:       min(requestedMB, totalRAMMB/2),
		lliteInstance: lfsGetname,
	}
}

// applyLimit sets the limit on the Lustre mount at target only. Each mount has
// its own llite instance, so other mounts on the node keep their own settings.
func (l *lustreCacheLimiter) applyLimit(target string) error {
	instance, err := l.lliteInstance(target)
	if err != nil {
		return err
	}
	if instance == "" || instance != filepath.Base(instance) {
		return fmt.Errorf("unexpected llite instance %q for %s", instance, target)
	}
	return l.applyLimitToFile(filepath.Join(l.lliteDir, instance, "max_cached_mb"))
}

func (l *lustreCacheLimiter) applyLimitToFile(file string) error {
	current, err := readMaxCachedMB(file)
	if err != nil {
		return err
	}
	if current == l.limitMB {
		return nil
	}

	klog.InfoS("Setting Lustre max_cached_mb", "file", file, "fromMB", current, "toMB", l.limitMB)
	return os.WriteFile(file, []byte(strconv.Itoa(l.limitMB)+"\n"), 0)
}

// lfsGetname runs `lfs getname <path>`, which prints "<instance> <mount point>".
func lfsGetname(path string) (string, error) {
	out, err := exec.Command("lfs", "getname", path).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("lfs getname %s: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return parseLfsGetname(string(out))
}

func parseLfsGetname(out string) (string, error) {
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return "", fmt.Errorf("unexpected lfs getname output %q", out)
	}
	return fields[0], nil
}

func readMaxCachedMB(file string) (int, error) {
	value, err := readField(file, "max_cached_mb:")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(value)
}

// totalRAMMB reads MemTotal from a /proc/meminfo-formatted file.
func totalRAMMB(meminfoPath string) (int, error) {
	value, err := readField(meminfoPath, "MemTotal:")
	if err != nil {
		return 0, err
	}
	kb, err := strconv.Atoi(strings.TrimSuffix(value, " kB"))
	if err != nil {
		return 0, fmt.Errorf("parsing MemTotal in %s: %w", meminfoPath, err)
	}
	return kb / 1024, nil
}

// readField returns the trimmed value of the first line starting with key.
func readField(file, key string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(scanner.Text(), key); ok {
			return strings.TrimSpace(value), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s: no %q line", file, key)
}
