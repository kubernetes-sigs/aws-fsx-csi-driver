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
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
)

// Values of --cluster-tag-isolation.
const (
	// ClusterTagIsolationDisable does not isolate. It adds the cluster tag to new file systems, and
	// the shared tag (cloud.SharedTagKey) to existing file systems this driver provisioned.
	ClusterTagIsolationDisable = "disable"
	// ClusterTagIsolationBackfill also adds the cluster tag to existing file systems this driver provisioned.
	ClusterTagIsolationBackfill = "backfill"
	// ClusterTagIsolationEnforce ignores file systems that lack this cluster's tag value.
	ClusterTagIsolationEnforce = "enforce"
)

// clusterTagBackfillInterval is the time between two backfill sweeps. New file systems get the
// driver's tags at create time, so later sweeps only pick up file systems an administrator tagged.
const clusterTagBackfillInterval = time.Hour

// maxListedResources caps the resource lists in the sweep summary line.
const maxListedResources = 20

// provisionedByAnnotation is set by the external-provisioner on the PVs it creates.
const provisionedByAnnotation = "pv.kubernetes.io/provisioned-by"

// ClusterTagOptions configures the cluster tag. The cluster tag is written as
// <TagKey>=<ClusterName> on every file system the controller creates, so that
// IAM policies can limit the controller to its own cluster's file systems.
type ClusterTagOptions struct {
	ClusterName string
	TagKey      string
	Isolation   string
}

// NewClusterTagOptions validates the cluster tag settings and the --extra-tags value against them.
func NewClusterTagOptions(clusterName, tagKey, isolation, extraTags string) (*ClusterTagOptions, error) {
	o := &ClusterTagOptions{
		ClusterName: strings.TrimSpace(clusterName),
		TagKey:      strings.TrimSpace(tagKey),
	}
	var err error
	if o.Isolation, err = parseClusterTagIsolation(isolation); err != nil {
		return nil, err
	}
	if o.Isolation != ClusterTagIsolationDisable && (o.ClusterName == "" || o.TagKey == "") {
		return nil, fmt.Errorf("--cluster-tag-isolation=%s requires --cluster-name and --cluster-tag-key", o.Isolation)
	}
	if o.TagKey == cloud.VolumeNameTagKey || o.TagKey == cloud.SharedTagKey || strings.HasPrefix(strings.ToLower(o.TagKey), "aws:") {
		return nil, fmt.Errorf("invalid --cluster-tag-key %q: %s, %s and keys starting with aws: are reserved",
			o.TagKey, cloud.VolumeNameTagKey, cloud.SharedTagKey)
	}
	if strings.ContainsAny(o.ClusterName, "=,") {
		return nil, fmt.Errorf("invalid --cluster-name %q: must not contain '=' or ','", o.ClusterName)
	}
	if extraTags != "" {
		for _, tag := range strings.Split(extraTags, ",") {
			key, _, _ := strings.Cut(tag, "=")
			if err := o.checkTagKey(key); err != nil {
				return nil, fmt.Errorf("invalid --extra-tags: %w", err)
			}
		}
	}
	return o, nil
}

// parseClusterTagIsolation accepts exactly "", disable, backfill and enforce. An empty value means
// disable. Values are not trimmed or lowercased.
func parseClusterTagIsolation(v string) (string, error) {
	switch v {
	case "":
		return ClusterTagIsolationDisable, nil
	case ClusterTagIsolationDisable, ClusterTagIsolationBackfill, ClusterTagIsolationEnforce:
		return v, nil
	}
	return "", fmt.Errorf(`invalid --cluster-tag-isolation %q: must be "disable", "backfill" or "enforce"`, v)
}

// logSettings logs the cluster tag isolation settings once at controller startup.
func (o *ClusterTagOptions) logSettings() {
	kv := []interface{}{"clusterName", o.ClusterName, "tagKey", o.TagKey, "clusterTagIsolation", o.Isolation}
	if o.sweeps() {
		kv = append(kv, "backfillInterval", clusterTagBackfillInterval.String())
	}
	klog.InfoS("cluster tag isolation settings", kv...)
}

// Tagging reports whether new file systems get the cluster tag.
func (o *ClusterTagOptions) Tagging() bool {
	return o != nil && o.ClusterName != "" && o.TagKey != ""
}

func (o *ClusterTagOptions) enforce() bool {
	return o != nil && o.Isolation == ClusterTagIsolationEnforce
}

func (o *ClusterTagOptions) backfill() bool {
	return o != nil && o.Isolation == ClusterTagIsolationBackfill
}

// sweeps reports whether the controller tags existing file systems: in disable
// and backfill, not in enforce.
func (o *ClusterTagOptions) sweeps() bool {
	return o != nil && !o.enforce()
}

// sweepTags returns the tags a sweep adds to a file system this driver
// provisioned: the shared tag, and in backfill also the cluster tag.
func (o *ClusterTagOptions) sweepTags() map[string]string {
	tags := map[string]string{cloud.SharedTagKey: cloud.SharedTagValue}
	if o.backfill() {
		tags[o.TagKey] = o.ClusterName
	}
	return tags
}

// sweepTagKeys returns the sorted keys of sweepTags, for log lines.
func (o *ClusterTagOptions) sweepTagKeys() []string {
	keys := []string{cloud.SharedTagKey}
	if o.backfill() {
		keys = append(keys, o.TagKey)
	}
	sort.Strings(keys)
	return keys
}

// checkTagKey rejects user-supplied tag keys that only the driver may set.
func (o *ClusterTagOptions) checkTagKey(key string) error {
	if key == cloud.VolumeNameTagKey || key == cloud.SharedTagKey {
		return fmt.Errorf("tag key %s is set by the driver", key)
	}
	if o.Tagging() && key == o.TagKey {
		return fmt.Errorf("tag key %s is the cluster tag (--cluster-tag-key) and is set by the driver", key)
	}
	return nil
}

// createTags returns the "key=value" tags for a new file system. StorageClass
// extraTags override --extra-tags for the same key. The cluster tag is last
// and cannot be overridden.
func (o *ClusterTagOptions) createTags(optionsTags []string, storageClassTags []string) ([]string, error) {
	var keys []string
	values := map[string]string{}
	add := func(tag string) {
		key, value, _ := strings.Cut(tag, "=")
		if _, ok := values[key]; !ok {
			keys = append(keys, key)
		}
		values[key] = value
	}
	for _, tag := range optionsTags {
		add(tag)
	}
	for _, tag := range storageClassTags {
		key, _, _ := strings.Cut(tag, "=")
		if err := o.checkTagKey(key); err != nil {
			return nil, fmt.Errorf("invalid extraTags: %w", err)
		}
		add(tag)
	}
	tags := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		tags = append(tags, key+"="+values[key])
	}
	if o.Tagging() {
		tags = append(tags, o.TagKey+"="+o.ClusterName)
	}
	return tags, nil
}

// checkOwnCluster returns an error when isolation is enforced and the file
// system does not carry this cluster's tag value.
func (o *ClusterTagOptions) checkOwnCluster(fs *cloud.FileSystem) error {
	if !o.enforce() {
		return nil
	}
	if v, ok := fs.Tags[o.TagKey]; ok && v == o.ClusterName {
		return nil
	}
	klog.InfoS("ignoring resource not tagged for this cluster", "resource", fs.FileSystemId, "tagKey", o.TagKey, "clusterName", o.ClusterName, "tagValue", fs.Tags[o.TagKey])
	return status.Errorf(codes.FailedPrecondition, "file system %s is not tagged %s=%s; --cluster-tag-isolation=enforce ignores it",
		fs.FileSystemId, o.TagKey, o.ClusterName)
}

// backfillResult counts the outcome of one backfill sweep. The counters mean:
//   - tagged: tag calls that succeeded;
//   - failed: tag calls that failed (not also counted in remaining);
//   - needsAdministrator: distinct file systems reported for an administrator;
//   - remaining: file systems, or whole listings, that could not be checked because a list or
//     describe call failed. A failed PersistentVolume listing counts 1.
type backfillResult struct {
	tagged             int
	failed             int
	needsAdministrator int
	remaining          int

	failedResources             []string
	needsAdministratorResources []string
	firstError                  error
	reported                    map[string]struct{}
}

func (r *backfillResult) complete() bool {
	return r.failed == 0 && r.needsAdministrator == 0 && r.remaining == 0
}

func (r *backfillResult) setError(err error) {
	if r.firstError == nil {
		r.firstError = err
	}
}

func (r *backfillResult) tagFailed(resource string, err error) {
	r.failed++
	r.failedResources = append(r.failedResources, resource)
	r.setError(err)
}

func (r *backfillResult) notChecked(err error) {
	r.remaining++
	r.setError(err)
}

func (r *backfillResult) needsAdmin(resource string, tagKeys []string) {
	if _, ok := r.reported[resource]; ok {
		return
	}
	if r.reported == nil {
		r.reported = map[string]struct{}{}
	}
	r.reported[resource] = struct{}{}
	r.needsAdministrator++
	r.needsAdministratorResources = append(r.needsAdministratorResources, resource)
	klog.V(4).InfoS("resource needs cluster tag from an administrator", "resource", resource, "tagKeys", tagKeys)
}

// summary returns the key/value pairs of the "cluster tag backfill sweep finished" line.
func (r *backfillResult) summary() []interface{} {
	kv := []interface{}{"tagged", r.tagged, "failed", r.failed, "needsAdministrator", r.needsAdministrator, "remaining", r.remaining}
	kv = appendResourceList(kv, "needsAdministrator", r.needsAdministratorResources)
	kv = appendResourceList(kv, "failed", r.failedResources)
	if (r.failed > 0 || r.remaining > 0) && r.firstError != nil {
		kv = append(kv, "firstError", r.firstError.Error())
	}
	return kv
}

// appendResourceList adds <name>Resources with at most maxListedResources IDs, and <name>Total
// when the list was cut.
func appendResourceList(kv []interface{}, name string, resources []string) []interface{} {
	if len(resources) == 0 {
		return kv
	}
	if len(resources) > maxListedResources {
		return append(kv, name+"Resources", resources[:maxListedResources], name+"Total", len(resources))
	}
	return append(kv, name+"Resources", resources)
}

// runClusterTagBackfill sweeps at start and then every interval until ctx is done. Each sweep
// logs one summary line; "complete" is logged once, after the first clean sweep.
func runClusterTagBackfill(ctx context.Context, o *ClusterTagOptions, c cloud.Cloud, client kubernetes.Interface, interval time.Duration) {
	completeLogged := false
	for {
		result := sweepClusterTags(ctx, o, c, client)
		klog.InfoS("cluster tag backfill sweep finished", result.summary()...)
		if result.complete() && !completeLogged {
			klog.InfoS("cluster tag backfill complete", "clusterName", o.ClusterName, "tagKeys", o.sweepTagKeys())
			completeLogged = true
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// sweepClusterTags visits the file system behind every FSx PersistentVolume in
// the cluster and adds the tags of sweepTags that it lacks. A file system this
// driver provisioned (the PV has the provisioned-by annotation and the file
// system's CSIVolumeName tag is the PV name) gets them in one request that
// carries only those tags. Any other file system that lacks them, or that
// carries another cluster's tag value, is reported for an administrator.
func sweepClusterTags(ctx context.Context, o *ClusterTagOptions, c cloud.Cloud, client kubernetes.Interface) *backfillResult {
	result := &backfillResult{}
	pvs, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(4).InfoS("cluster tag backfill could not list PersistentVolumes", "err", err)
		result.notChecked(fmt.Errorf("listing PersistentVolumes: %w", err))
		return result
	}
	want := o.sweepTags()
	keys := o.sweepTagKeys()
	seen := map[string]bool{}
	for _, pv := range pvs.Items {
		if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != DriverName {
			continue
		}
		fsID := pv.Spec.CSI.VolumeHandle
		if seen[fsID] {
			continue
		}
		seen[fsID] = true

		fs, err := c.DescribeFileSystem(ctx, fsID)
		if errors.Is(err, cloud.ErrNotFound) {
			klog.V(4).InfoS("cluster tag backfill: file system not found", "resource", fsID, "persistentVolume", pv.Name)
			continue
		}
		if err != nil {
			klog.V(4).InfoS("cluster tag backfill could not describe file system", "resource", fsID, "persistentVolume", pv.Name, "err", err)
			result.notChecked(err)
			continue
		}
		missing := false
		for key, value := range want {
			if fs.Tags[key] != value {
				missing = true
			}
		}
		if !missing {
			continue
		}
		otherCluster := false
		if o.backfill() {
			value, hasTag := fs.Tags[o.TagKey]
			otherCluster = hasTag && value != o.ClusterName
		}
		owned := pv.Annotations[provisionedByAnnotation] == DriverName && fs.Tags[cloud.VolumeNameTagKey] == pv.Name
		if otherCluster || !owned {
			result.needsAdmin(fs.FileSystemId, keys)
			continue
		}
		if err := c.TagFileSystem(ctx, fs.ResourceARN, want); err != nil {
			klog.V(4).InfoS("failed to add cluster tag to existing resource", "resource", fs.FileSystemId, "tagKeys", keys, "err", err)
			result.tagFailed(fs.FileSystemId, err)
			continue
		}
		klog.V(4).InfoS("added cluster tag to existing resource", "resource", fs.FileSystemId, "tagKeys", keys)
		result.tagged++
	}
	return result
}
