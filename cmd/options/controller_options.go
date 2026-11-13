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
	flag "github.com/spf13/pflag"
)

// ControllerOptions contains options and configuration settings for the controller service.
type ControllerOptions struct {
	// ExtraTags is a map of tags that will be attached to each dynamically provisioned resource.
	ExtraTags string
	// ClusterName is the name of the cluster. It is the value of the cluster tag.
	ClusterName string
	// ClusterTagKey is the key of the cluster tag.
	ClusterTagKey string
	// ClusterTagIsolation is one of disable, backfill or enforce.
	ClusterTagIsolation string
}

func (s *ControllerOptions) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.ExtraTags, "extra-tags", "", "Tags (key1=value1,key2=value2) added to every AWS resource the controller creates. "+
		"Not used to find, filter or isolate resources, and not the cluster tag. "+
		"A StorageClass extraTags value overrides these for the same key. Neither can override CSIVolumeName, fsx.csi.aws.com/cluster or the --cluster-tag-key key.")
	fs.StringVar(&s.ClusterName, "cluster-name", "", "Name of the cluster. With --cluster-tag-key set, every file system the controller creates is tagged <cluster-tag-key>=<cluster-name>.")
	fs.StringVar(&s.ClusterTagKey, "cluster-tag-key", "", "Key of the cluster tag, whose value is --cluster-name. Empty (the default) writes no cluster tag.")
	fs.StringVar(&s.ClusterTagIsolation, "cluster-tag-isolation", "disable", "Set this to match the IAM policy attached to the controller. "+
		"disable: no isolation; tags existing file systems this driver provisioned fsx.csi.aws.com/cluster=true, at startup and every hour, and new ones with the cluster tag. "+
		"backfill: requires --cluster-name and --cluster-tag-key; tags existing file systems this driver provisioned, at startup and every hour, and does not filter. "+
		"enforce: requires --cluster-name and --cluster-tag-key; ignores file systems that lack this cluster's tag value, and does not backfill.")
}
