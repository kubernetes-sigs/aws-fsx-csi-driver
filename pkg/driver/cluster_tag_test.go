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
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/internal"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/mocks"
)

const (
	testTagKey      = "example.com/cluster"
	testClusterName = "my-cluster"
)

func TestNewClusterTagOptions(t *testing.T) {
	testCases := []struct {
		name                                   string
		clusterName, tagKey, isolation, extras string
		want                                   *ClusterTagOptions
		wantErr                                string
	}{
		{name: "defaults", want: &ClusterTagOptions{Isolation: ClusterTagIsolationDisable}},
		{name: "disable with name and key", clusterName: testClusterName, tagKey: testTagKey, isolation: "disable",
			want: &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationDisable}},
		{name: "backfill", clusterName: testClusterName, tagKey: testTagKey, isolation: "backfill",
			want: &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}},
		{name: "enforce", clusterName: testClusterName, tagKey: testTagKey, isolation: "enforce", extras: "team=a,env=b",
			want: &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationEnforce}},
		{name: "backfill without cluster name", tagKey: testTagKey, isolation: "backfill", wantErr: "requires --cluster-name and --cluster-tag-key"},
		{name: "enforce without tag key", clusterName: testClusterName, isolation: "enforce", wantErr: "requires --cluster-name and --cluster-tag-key"},
		{name: "empty is disable", clusterName: testClusterName, tagKey: testTagKey, isolation: "",
			want: &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationDisable}},
		{name: "false", isolation: "false", wantErr: invalidIsolationError("false")},
		{name: "true", isolation: "true", wantErr: invalidIsolationError("true")},
		{name: "on", isolation: "on", wantErr: invalidIsolationError("on")},
		{name: "off", isolation: "off", wantErr: invalidIsolationError("off")},
		{name: "uppercase OFF", isolation: "OFF", wantErr: invalidIsolationError("OFF")},
		{name: "leading space", isolation: " disable", wantErr: invalidIsolationError(" disable")},
		{name: "unknown state", isolation: "strict", wantErr: invalidIsolationError("strict")},
		{name: "reserved key CSIVolumeName", clusterName: testClusterName, tagKey: "CSIVolumeName", wantErr: "reserved"},
		{name: "reserved key aws prefix", clusterName: testClusterName, tagKey: "aws:cluster", wantErr: "reserved"},
		{name: "reserved shared key", clusterName: testClusterName, tagKey: cloud.SharedTagKey, wantErr: "reserved"},
		{name: "extra tags set the shared tag", extras: cloud.SharedTagKey + "=false", wantErr: "invalid --extra-tags"},
		{name: "cluster name with separator", clusterName: "a=b", tagKey: testTagKey, wantErr: "invalid --cluster-name"},
		{name: "extra tags set the cluster tag", clusterName: testClusterName, tagKey: testTagKey, extras: "team=a," + testTagKey + "=other",
			wantErr: "invalid --extra-tags"},
		{name: "extra tags set CSIVolumeName", extras: "CSIVolumeName=x", wantErr: "invalid --extra-tags"},
		{name: "extra tags may use the key when no cluster tag is written", tagKey: testTagKey, extras: testTagKey + "=x",
			want: &ClusterTagOptions{TagKey: testTagKey, Isolation: ClusterTagIsolationDisable}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewClusterTagOptions(tc.clusterName, tc.tagKey, tc.isolation, tc.extras)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got error %v, want one containing %q", err, tc.wantErr)
				}
				if strings.HasPrefix(tc.wantErr, "invalid --cluster-tag-isolation ") && err.Error() != tc.wantErr {
					t.Fatalf("got error %q, want exactly %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func invalidIsolationError(v string) string {
	return `invalid --cluster-tag-isolation "` + v + `": must be "disable", "backfill" or "enforce"`
}

func TestCreateTags(t *testing.T) {
	tagging := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationDisable}
	testCases := []struct {
		name        string
		opts        *ClusterTagOptions
		optionsTags []string
		scTags      []string
		want        []string
		wantErr     bool
	}{
		{name: "no cluster tag", opts: &ClusterTagOptions{}, optionsTags: []string{"a=1"}, scTags: []string{"b=2"}, want: []string{"a=1", "b=2"}},
		{name: "nil options", opts: nil, optionsTags: []string{"a=1"}, want: []string{"a=1"}},
		{name: "StorageClass overrides extra tags", opts: &ClusterTagOptions{}, optionsTags: []string{"a=1", "b=1"}, scTags: []string{"a=2"},
			want: []string{"a=2", "b=1"}},
		{name: "cluster tag added last", opts: tagging, optionsTags: []string{"a=1"}, scTags: []string{"b=2"},
			want: []string{"a=1", "b=2", testTagKey + "=" + testClusterName}},
		{name: "cluster tag only", opts: tagging, want: []string{testTagKey + "=" + testClusterName}},
		{name: "StorageClass may not set the cluster tag", opts: tagging, scTags: []string{testTagKey + "=other"}, wantErr: true},
		{name: "StorageClass may not set CSIVolumeName", opts: &ClusterTagOptions{}, scTags: []string{"CSIVolumeName=x"}, wantErr: true},
		{name: "StorageClass may not set the shared tag", opts: &ClusterTagOptions{}, scTags: []string{cloud.SharedTagKey + "=false"}, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.opts.createTags(tc.optionsTags, tc.scTags)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func clusterTagCreateRequest(extraTags string) *csi.CreateVolumeRequest {
	params := map[string]string{volumeParamsSubnetId: "subnet-1", volumeParamsSecurityGroupIds: "sg-1"}
	if extraTags != "" {
		params[volumeParamsExtraTags] = extraTags
	}
	return &csi.CreateVolumeRequest{
		Name: "pvc-1",
		VolumeCapabilities: []*csi.VolumeCapability{{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER},
		}},
		Parameters: params,
	}
}

// Every state writes the cluster tag on new file systems.
func TestCreateVolumeWritesClusterTagInEveryState(t *testing.T) {
	for _, isolation := range []string{ClusterTagIsolationDisable, ClusterTagIsolationBackfill, ClusterTagIsolationEnforce} {
		t.Run(isolation, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			mockCloud := mocks.NewMockCloud(mockCtl)
			opts, err := NewClusterTagOptions(testClusterName, testTagKey, isolation, "team=a")
			if err != nil {
				t.Fatal(err)
			}
			d := controllerService{cloud: mockCloud, inFlight: internal.NewInFlight(),
				driverOptions: &DriverOptions{extraTags: "team=a", clusterTag: opts}}
			fs := &cloud.FileSystem{FileSystemId: "fs-1", CapacityGiB: 1200}
			mockCloud.EXPECT().GetFileSystemIdByVolumeName("pvc-1").Return("", false)
			mockCloud.EXPECT().CreateFileSystem(gomock.Any(), "pvc-1", gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, o *cloud.FileSystemOptions) (*cloud.FileSystem, error) {
					want := []string{"team=b", "sc=1", testTagKey + "=" + testClusterName}
					if !reflect.DeepEqual(o.ExtraTags, want) {
						t.Errorf("tags %v, want %v", o.ExtraTags, want)
					}
					return fs, nil
				})
			mockCloud.EXPECT().WaitForFileSystemAvailable(gomock.Any(), "fs-1").Return(nil)
			if _, err := d.CreateVolume(context.Background(), clusterTagCreateRequest("team=b,sc=1")); err != nil {
				t.Fatalf("CreateVolume: %v", err)
			}
		})
	}
}

func TestCreateVolumeRejectsStorageClassClusterTag(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockCloud := mocks.NewMockCloud(mockCtl)
	opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationDisable}
	d := controllerService{cloud: mockCloud, inFlight: internal.NewInFlight(), driverOptions: &DriverOptions{clusterTag: opts}}
	mockCloud.EXPECT().GetFileSystemIdByVolumeName("pvc-1").Return("", false)
	_, err := d.CreateVolume(context.Background(), clusterTagCreateRequest(testTagKey+"=other"))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", err)
	}
}

func TestDeleteVolumeClusterTagIsolation(t *testing.T) {
	own := &cloud.FileSystem{FileSystemId: "fs-1", Tags: map[string]string{testTagKey: testClusterName}}
	other := &cloud.FileSystem{FileSystemId: "fs-1", Tags: map[string]string{testTagKey: "other-cluster"}}
	untagged := &cloud.FileSystem{FileSystemId: "fs-1", Tags: map[string]string{cloud.VolumeNameTagKey: "pvc-1"}}
	testCases := []struct {
		name      string
		isolation string
		describe  *cloud.FileSystem
		descErr   error
		delete    bool
		wantCode  codes.Code
	}{
		{name: "disable does not describe", isolation: ClusterTagIsolationDisable, delete: true, wantCode: codes.OK},
		{name: "backfill does not describe", isolation: ClusterTagIsolationBackfill, delete: true, wantCode: codes.OK},
		{name: "enforce deletes own file system", isolation: ClusterTagIsolationEnforce, describe: own, delete: true, wantCode: codes.OK},
		{name: "enforce ignores another cluster's file system", isolation: ClusterTagIsolationEnforce, describe: other, wantCode: codes.FailedPrecondition},
		{name: "enforce ignores untagged file system", isolation: ClusterTagIsolationEnforce, describe: untagged, wantCode: codes.FailedPrecondition},
		{name: "enforce treats not found as deleted", isolation: ClusterTagIsolationEnforce, descErr: cloud.ErrNotFound, wantCode: codes.OK},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			mockCloud := mocks.NewMockCloud(mockCtl)
			opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: tc.isolation}
			d := controllerService{cloud: mockCloud, inFlight: internal.NewInFlight(), driverOptions: &DriverOptions{clusterTag: opts}}
			if tc.describe != nil || tc.descErr != nil {
				mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-1").Return(tc.describe, tc.descErr)
			}
			if tc.delete {
				mockCloud.EXPECT().DeleteFileSystem(gomock.Any(), "fs-1").Return(nil)
			}
			_, err := d.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{VolumeId: "fs-1"})
			if status.Code(err) != tc.wantCode {
				t.Fatalf("got %v, want code %v", err, tc.wantCode)
			}
		})
	}
}

func TestExpandVolumeClusterTagIsolation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		tags     map[string]string
		wantCode codes.Code
	}{
		{name: "own file system", tags: map[string]string{testTagKey: testClusterName}, wantCode: codes.OK},
		{name: "another cluster's file system", tags: map[string]string{testTagKey: "other-cluster"}, wantCode: codes.FailedPrecondition},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			mockCloud := mocks.NewMockCloud(mockCtl)
			opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationEnforce}
			d := controllerService{cloud: mockCloud, inFlight: internal.NewInFlight(), driverOptions: &DriverOptions{clusterTag: opts}}
			fs := &cloud.FileSystem{FileSystemId: "fs-1", CapacityGiB: 1200, DeploymentType: "SCRATCH_2", StorageType: "SSD", Tags: tc.tags}
			mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-1").Return(fs, nil)
			if tc.wantCode == codes.OK {
				mockCloud.EXPECT().ResizeFileSystem(gomock.Any(), "fs-1", int32(2400)).Return(int32(2400), nil)
				mockCloud.EXPECT().WaitForFileSystemResize(gomock.Any(), "fs-1", int32(2400)).Return(nil)
			}
			_, err := d.ControllerExpandVolume(context.Background(), &csi.ControllerExpandVolumeRequest{
				VolumeId: "fs-1", CapacityRange: &csi.CapacityRange{RequiredBytes: 2400 * 1024 * 1024 * 1024}})
			if status.Code(err) != tc.wantCode {
				t.Fatalf("got %v, want code %v", err, tc.wantCode)
			}
		})
	}
}

func fsxPV(name, fsID string, provisioned bool) *corev1.PersistentVolume {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeSpec{PersistentVolumeSource: corev1.PersistentVolumeSource{
			CSI: &corev1.CSIPersistentVolumeSource{Driver: DriverName, VolumeHandle: fsID}}},
	}
	if provisioned {
		pv.Annotations = map[string]string{provisionedByAnnotation: DriverName}
	}
	return pv
}

func fsWithTags(id string, tags map[string]string) *cloud.FileSystem {
	return &cloud.FileSystem{FileSystemId: id, ResourceARN: "arn:aws:fsx:us-east-1:111122223333:file-system/" + id, Tags: tags}
}

func TestSweepClusterTags(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockCloud := mocks.NewMockCloud(mockCtl)
	otherDriver := fsxPV("pv-ebs", "vol-1", true)
	otherDriver.Spec.CSI.Driver = "ebs.csi.aws.com"
	client := fake.NewSimpleClientset(
		fsxPV("pvc-legacy", "fs-legacy", true),       // owned, untagged: tagged
		fsxPV("pvc-legacy-copy", "fs-legacy", false), // same file system again: visited once
		fsxPV("pvc-tagged", "fs-tagged", true),       // already tagged: nothing to do
		fsxPV("pvc-other", "fs-other", true),         // tagged for another cluster: administrator
		fsxPV("static", "fs-static", false),          // statically provisioned: administrator
		fsxPV("pvc-mismatch", "fs-mismatch", true),   // CSIVolumeName does not match: administrator
		fsxPV("pvc-gone", "fs-gone", true),           // deleted file system: skipped
		fsxPV("pvc-error", "fs-error", true),         // describe fails: remaining
		fsxPV("pvc-denied", "fs-denied", true),       // tagging fails: failed
		otherDriver,
	)
	opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}

	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-legacy").Return(fsWithTags("fs-legacy", map[string]string{cloud.VolumeNameTagKey: "pvc-legacy"}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-tagged").Return(fsWithTags("fs-tagged", map[string]string{cloud.VolumeNameTagKey: "pvc-tagged", testTagKey: testClusterName, cloud.SharedTagKey: cloud.SharedTagValue}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-other").Return(fsWithTags("fs-other", map[string]string{cloud.VolumeNameTagKey: "pvc-other", testTagKey: "other"}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-static").Return(fsWithTags("fs-static", map[string]string{}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-mismatch").Return(fsWithTags("fs-mismatch", map[string]string{cloud.VolumeNameTagKey: "pvc-x"}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-gone").Return(nil, cloud.ErrNotFound)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-error").Return(nil, errors.New("throttled"))
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-denied").Return(fsWithTags("fs-denied", map[string]string{cloud.VolumeNameTagKey: "pvc-denied"}), nil)
	// One request with both driver tags.
	mockCloud.EXPECT().TagFileSystem(gomock.Any(), fsWithTags("fs-legacy", nil).ResourceARN, map[string]string{cloud.SharedTagKey: cloud.SharedTagValue, testTagKey: testClusterName}).Return(nil)
	mockCloud.EXPECT().TagFileSystem(gomock.Any(), fsWithTags("fs-denied", nil).ResourceARN, map[string]string{cloud.SharedTagKey: cloud.SharedTagValue, testTagKey: testClusterName}).Return(errors.New("AccessDenied"))

	got := sweepClusterTags(context.Background(), opts, mockCloud, client)
	// A failed tag call counts in failed only; a failed describe counts in remaining.
	if counts := [4]int{got.tagged, got.failed, got.needsAdministrator, got.remaining}; counts != [4]int{1, 1, 3, 1} {
		t.Fatalf("got tagged, failed, needsAdministrator, remaining = %v, want [1 1 3 1]", counts)
	}
	if want := []string{"fs-mismatch", "fs-other", "fs-static"}; !reflect.DeepEqual(got.needsAdministratorResources, want) {
		t.Fatalf("needsAdministratorResources = %v, want %v", got.needsAdministratorResources, want)
	}
	if want := []string{"fs-denied"}; !reflect.DeepEqual(got.failedResources, want) {
		t.Fatalf("failedResources = %v, want %v", got.failedResources, want)
	}
	if got.firstError == nil || got.firstError.Error() != "AccessDenied" {
		t.Fatalf("firstError = %v, want AccessDenied (pvc-denied is listed before pvc-error)", got.firstError)
	}
	if got.complete() {
		t.Fatal("sweep with failures reported complete")
	}
}

func TestSweepSharedTagInDisable(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockCloud := mocks.NewMockCloud(mockCtl)
	client := fake.NewSimpleClientset(
		fsxPV("pvc-legacy", "fs-legacy", true), // owned, no shared tag: tagged
		fsxPV("pvc-shared", "fs-shared", true), // already has the shared tag: nothing to do
		fsxPV("pvc-other", "fs-other", true),   // another cluster's tag doesn't matter in disable: tagged
		fsxPV("static", "fs-static", false),    // statically provisioned: administrator
	)
	opts := &ClusterTagOptions{Isolation: ClusterTagIsolationDisable}
	sharedOnly := map[string]string{cloud.SharedTagKey: cloud.SharedTagValue}

	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-legacy").Return(fsWithTags("fs-legacy", map[string]string{cloud.VolumeNameTagKey: "pvc-legacy"}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-shared").Return(fsWithTags("fs-shared", map[string]string{cloud.VolumeNameTagKey: "pvc-shared", cloud.SharedTagKey: cloud.SharedTagValue}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-other").Return(fsWithTags("fs-other", map[string]string{cloud.VolumeNameTagKey: "pvc-other", testTagKey: "other"}), nil)
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-static").Return(fsWithTags("fs-static", map[string]string{}), nil)
	mockCloud.EXPECT().TagFileSystem(gomock.Any(), fsWithTags("fs-legacy", nil).ResourceARN, sharedOnly).Return(nil)
	mockCloud.EXPECT().TagFileSystem(gomock.Any(), fsWithTags("fs-other", nil).ResourceARN, sharedOnly).Return(nil)

	got := sweepClusterTags(context.Background(), opts, mockCloud, client)
	if counts := [4]int{got.tagged, got.failed, got.needsAdministrator, got.remaining}; counts != [4]int{2, 0, 1, 0} {
		t.Fatalf("got tagged, failed, needsAdministrator, remaining = %v, want [2 0 1 0]", counts)
	}
	if want := []string{"fs-static"}; !reflect.DeepEqual(got.needsAdministratorResources, want) {
		t.Fatalf("needsAdministratorResources = %v, want %v", got.needsAdministratorResources, want)
	}
}

func TestSweepsExceptInEnforce(t *testing.T) {
	for isolation, want := range map[string]bool{ClusterTagIsolationDisable: true, ClusterTagIsolationBackfill: true, ClusterTagIsolationEnforce: false} {
		if got := (&ClusterTagOptions{Isolation: isolation}).sweeps(); got != want {
			t.Errorf("%s: sweeps() = %v, want %v", isolation, got, want)
		}
	}
}

// syncBuffer is a goroutine-safe log sink.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureKlog(t *testing.T) *syncBuffer {
	t.Helper()
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)
	_ = fs.Set("logtostderr", "false")
	_ = fs.Set("alsologtostderr", "false")
	buf := &syncBuffer{}
	klog.SetOutput(buf)
	t.Cleanup(func() {
		klog.Flush()
		_ = fs.Set("logtostderr", "true")
	})
	return buf
}

func runSweeps(t *testing.T, mockCloud *mocks.MockCloud, sweeps int) string {
	t.Helper()
	buf := captureKlog(t)
	client := fake.NewSimpleClientset(fsxPV("pvc-legacy", "fs-legacy", true))
	opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runClusterTagBackfill(ctx, opts, mockCloud, client, time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(10 * time.Second)
	for strings.Count(buf.String(), "cluster tag backfill sweep finished") < sweeps && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	klog.Flush()
	return buf.String()
}

func TestBackfillCompleteOnlyWhenTrue(t *testing.T) {
	t.Run("failed tagging never reports complete", func(t *testing.T) {
		mockCtl := gomock.NewController(t)
		mockCloud := mocks.NewMockCloud(mockCtl)
		mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-legacy").Return(
			fsWithTags("fs-legacy", map[string]string{cloud.VolumeNameTagKey: "pvc-legacy"}), nil).AnyTimes()
		mockCloud.EXPECT().TagFileSystem(gomock.Any(), gomock.Any(), map[string]string{cloud.SharedTagKey: cloud.SharedTagValue, testTagKey: testClusterName}).Return(errors.New("AccessDenied")).AnyTimes()
		logs := runSweeps(t, mockCloud, 3)
		if !strings.Contains(logs, `"cluster tag backfill sweep finished" tagged=0 failed=1 needsAdministrator=0 remaining=0 failedResources=["fs-legacy"] firstError="AccessDenied"`) {
			t.Fatalf("missing failure summary:\n%s", logs)
		}
		if strings.Contains(logs, "cluster tag backfill complete") {
			t.Fatalf("complete logged although tagging failed:\n%s", logs)
		}
	})
	t.Run("complete is logged once", func(t *testing.T) {
		mockCtl := gomock.NewController(t)
		mockCloud := mocks.NewMockCloud(mockCtl)
		tags := map[string]string{cloud.VolumeNameTagKey: "pvc-legacy"}
		var mu sync.Mutex
		mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-legacy").DoAndReturn(func(context.Context, string) (*cloud.FileSystem, error) {
			mu.Lock()
			defer mu.Unlock()
			copied := map[string]string{}
			for k, v := range tags {
				copied[k] = v
			}
			return fsWithTags("fs-legacy", copied), nil
		}).AnyTimes()
		mockCloud.EXPECT().TagFileSystem(gomock.Any(), gomock.Any(), map[string]string{cloud.SharedTagKey: cloud.SharedTagValue, testTagKey: testClusterName}).DoAndReturn(func(_ context.Context, _ string, add map[string]string) error {
			mu.Lock()
			defer mu.Unlock()
			for k, v := range add {
				tags[k] = v
			}
			return nil
		}).Times(1)
		logs := runSweeps(t, mockCloud, 3)
		if !strings.Contains(logs, `"cluster tag backfill sweep finished" tagged=1 failed=0 needsAdministrator=0 remaining=0`) {
			t.Fatalf("missing tag summary:\n%s", logs)
		}
		if n := strings.Count(logs, "cluster tag backfill complete"); n != 1 {
			t.Fatalf("complete logged %d times:\n%s", n, logs)
		}
	})
}

// logMessages returns the structured part (after "] ") of each klog line in out.
func logMessages(out string) []string {
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if _, msg, ok := strings.Cut(l, "] "); ok {
			lines = append(lines, msg)
		}
	}
	return lines
}

func TestClusterTagBackfillLogLines(t *testing.T) {
	buf := captureKlog(t)
	mockCloud := mocks.NewMockCloud(gomock.NewController(t))
	var pvs []runtime.Object
	var admin []string
	for i := 0; i < 22; i++ {
		id := fmt.Sprintf("fs-static-%02d", i)
		pvs = append(pvs, fsxPV("static-"+id, id, false))
		if i < 20 {
			admin = append(admin, id)
		}
		mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), id).Return(fsWithTags(id, map[string]string{}), nil)
	}
	pvs = append(pvs, fsxPV("pvc-denied", "fs-denied", true))
	mockCloud.EXPECT().DescribeFileSystem(gomock.Any(), "fs-denied").Return(fsWithTags("fs-denied", map[string]string{cloud.VolumeNameTagKey: "pvc-denied"}), nil)
	mockCloud.EXPECT().TagFileSystem(gomock.Any(), gomock.Any(), map[string]string{cloud.SharedTagKey: cloud.SharedTagValue, testTagKey: testClusterName}).Return(errors.New("AccessDenied"))
	opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}
	ctx := context.Background()

	// Sweep 1: 22 file systems for an administrator (list capped at 20) and one failed tag call.
	r := sweepClusterTags(ctx, opts, mockCloud, fake.NewSimpleClientset(pvs...))
	klog.InfoS("cluster tag backfill sweep finished", r.summary()...)
	klog.Flush()
	want := `"cluster tag backfill sweep finished" tagged=0 failed=1 needsAdministrator=22 remaining=0 needsAdministratorResources=["` +
		strings.Join(admin, `","`) + `"] needsAdministratorTotal=22 failedResources=["fs-denied"] firstError="AccessDenied"`
	if got := logMessages(buf.String()); !reflect.DeepEqual(got, []string{want}) {
		t.Fatalf("log lines:\n got %q\nwant %q", got, want)
	}
}

func TestClusterTagBackfillCompleteOncePerProcess(t *testing.T) {
	mockCloud := mocks.NewMockCloud(gomock.NewController(t))
	client := fake.NewSimpleClientset()
	// The third PersistentVolume list fails; the others succeed with no PVs.
	var mu sync.Mutex
	calls := 0
	client.PrependReactor("list", "persistentvolumes", func(k8stesting.Action) (bool, runtime.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == 3 {
			return true, nil, errors.New("connection refused")
		}
		return false, nil, nil
	})
	buf := captureKlog(t)
	opts := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runClusterTagBackfill(ctx, opts, mockCloud, client, time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(10 * time.Second)
	for strings.Count(buf.String(), "cluster tag backfill sweep finished") < 4 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	klog.Flush()
	got := logMessages(buf.String())
	if len(got) < 5 {
		t.Fatalf("expected at least 5 lines, got %q", got)
	}
	want := []string{
		`"cluster tag backfill sweep finished" tagged=0 failed=0 needsAdministrator=0 remaining=0`,
		`"cluster tag backfill complete" clusterName="my-cluster" tagKeys=["example.com/cluster","fsx.csi.aws.com/cluster"]`,
		`"cluster tag backfill sweep finished" tagged=0 failed=0 needsAdministrator=0 remaining=0`,
		`"cluster tag backfill sweep finished" tagged=0 failed=0 needsAdministrator=0 remaining=1 firstError="listing PersistentVolumes: connection refused"`,
		`"cluster tag backfill sweep finished" tagged=0 failed=0 needsAdministrator=0 remaining=0`,
	}
	if !reflect.DeepEqual(got[:5], want) {
		t.Fatalf("log lines:\n got %q\nwant %q", got[:5], want)
	}
	if n := strings.Count(buf.String(), "cluster tag backfill complete"); n != 1 {
		t.Fatalf("complete logged %d times", n)
	}
}

func TestClusterTagSettingsAndEnforceLogLines(t *testing.T) {
	buf := captureKlog(t)
	(&ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationBackfill}).logSettings()
	(&ClusterTagOptions{Isolation: ClusterTagIsolationDisable}).logSettings()
	o := &ClusterTagOptions{ClusterName: testClusterName, TagKey: testTagKey, Isolation: ClusterTagIsolationEnforce}
	_ = o.checkOwnCluster(fsWithTags("fs-other", map[string]string{testTagKey: "other"}))
	err := o.checkOwnCluster(fsWithTags("fs-untagged", nil))
	klog.Flush()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("got %v, want FailedPrecondition", err)
	}
	want := []string{
		`"cluster tag isolation settings" clusterName="my-cluster" tagKey="example.com/cluster" clusterTagIsolation="backfill" backfillInterval="1h0m0s"`,
		`"cluster tag isolation settings" clusterName="" tagKey="" clusterTagIsolation="disable" backfillInterval="1h0m0s"`,
		`"ignoring resource not tagged for this cluster" resource="fs-other" tagKey="example.com/cluster" clusterName="my-cluster" tagValue="other"`,
		`"ignoring resource not tagged for this cluster" resource="fs-untagged" tagKey="example.com/cluster" clusterName="my-cluster" tagValue=""`,
	}
	if got := logMessages(buf.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("log lines:\n got %q\nwant %q", got, want)
	}
}
