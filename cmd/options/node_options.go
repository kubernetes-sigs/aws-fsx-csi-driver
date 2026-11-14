/*
Copyright 2020 The Kubernetes Authors.

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

package options

import (
	"fmt"
	"time"

	flag "github.com/spf13/pflag"
)

// minForcefulUnmountTimeout is the smallest positive --forceful-unmount-timeout
// accepted. A normal unmount of a busy network filesystem can legitimately take
// tens of seconds, and escalating to umount -f before it has had a fair chance
// risks tearing down a mount that was about to come down cleanly.
const minForcefulUnmountTimeout = 30 * time.Second

// NodeOptions contains options and configuration settings for the node service.
type NodeOptions struct {
	// ForcefulUnmountTimeout is the duration after which a hanging unmount will be
	// retried with umount -f, while periodically checking if the mount point has
	// already disappeared. Negative disables the feature (default, preserves
	// existing behavior); 0 skips the normal unmount entirely.
	ForcefulUnmountTimeout time.Duration

	// LustreMaxCachedMB caps the Lustre client read cache of each new mount. Zero leaves the Lustre default.
	LustreMaxCachedMB int
}

func (o *NodeOptions) AddFlags(fs *flag.FlagSet) {
	fs.DurationVar(&o.ForcefulUnmountTimeout, "forceful-unmount-timeout", -1,
		"Controls forced unmount behavior. "+
			"Set to a positive duration (at least 30s, e.g. 2m) to wait that long "+
			"for a normal unmount before escalating to umount -f. "+
			"Set to 0 to call umount -f directly (skip normal unmount). "+
			"Set to a negative duration to disable forced unmount entirely (default).")
	fs.IntVar(&o.LustreMaxCachedMB, "lustre-max-cached-mb", 0, "Cap on the Lustre client read cache (llite max_cached_mb) of each new mount, never above half of RAM. 0 leaves the Lustre default. Requires the host's debugfs at /sys/kernel/debug.")
}

// Validate checks that NodeOptions values are within acceptable bounds.
func (o *NodeOptions) Validate() error {
	if o.ForcefulUnmountTimeout > 0 && o.ForcefulUnmountTimeout < minForcefulUnmountTimeout {
		return fmt.Errorf("--forceful-unmount-timeout must be at least %v if positive (use 0 to force unmount immediately, or a negative value to disable), got %v",
			minForcefulUnmountTimeout, o.ForcefulUnmountTimeout)
	}
	return nil
}
