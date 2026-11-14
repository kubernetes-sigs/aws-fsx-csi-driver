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
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/internal"
	driverMocks "sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/mocks"
)

var (
	volumeID = "voltest"
)

func TestNodePublishVolume(t *testing.T) {

	var (
		dnsname       = "fs-0a2d0632b5ff567e9.fsx.us-west-2.amazonaws.com"
		mountname     = "random"
		targetPath    = "/target/path"
		targetPathAlt = "/target/alt_path"
		stdVolCap     = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		}
	)

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}
				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: missing mountname for static provisioning, default 'fsx' used",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}
				source := dnsname + "@tcp:/fsx"

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName: dnsname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with read only mount",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
					Readonly:         true,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Eq([]string{"ro"})).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with flock mount options",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: &csi.VolumeCapability{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{
								MountFlags: []string{"flock"},
							},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Eq([]string{"flock"})).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing dns name",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing target path",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing volume capability",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					TargetPath: targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: unsupported volume capability",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: &csi.VolumeCapability{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
						},
					},
					TargetPath: targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mounter failed to MakeDir",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				err := fmt.Errorf("failed to MakeDir")
				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(err)

				_, err = driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mounter failed to Mount",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				source := dnsname + "@tcp:/" + mountname
				err := fmt.Errorf("failed to Mount")
				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(err)

				_, err = driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail another operation in-flight on given volumeId-targetPath",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				req := &csi.NodePublishVolumeRequest{
					VolumeId: volumeID,
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				rpcKey := fmt.Sprintf("%s-%s", volumeID, targetPath)

				awsDriver.inFlight.Insert(rpcKey)
				_, err := awsDriver.NodePublishVolume(context.TODO(), req)
				expectErr(t, err, codes.Aborted)
			},
		},
		{
			name: "success: operation in-flight with different volumeId-targetPath",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				rpcKeyAlt := fmt.Sprintf("%s-%s", volumeID, targetPathAlt)

				awsDriver.inFlight.Insert(rpcKeyAlt)

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := awsDriver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: caps the Lustre read cache after mounting",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				lliteDir := t.TempDir()
				maxCachedMBFile := writeFakeLlite(t, lliteDir, "fsx-ffff0001", fakeMaxCachedMB)
				driver := &nodeService{
					mounter:            mockMounter,
					inFlight:           internal.NewInFlight(),
					lustreCacheLimiter: newTestLustreCacheLimiter(lliteDir, map[string]string{targetPath: "fsx-ffff0001"}),
				}
				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				assert.Equal(t, "102400\n", readFile(t, maxCachedMBFile))
			},
		},
		{
			name: "success: mounts even when the Lustre read cache cannot be capped",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:            mockMounter,
					inFlight:           internal.NewInFlight(),
					lustreCacheLimiter: newTestLustreCacheLimiter(t.TempDir(), map[string]string{}),
				}
				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						volumeContextDnsName:   dnsname,
						volumeContextMountName: mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodeUnpublishVolume(t *testing.T) {

	var (
		targetPath    = "/target/path"
		targetPathAlt = "/target/alt_path"
	)

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: -1},
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(nil)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodeUnpublishVolume is failed: %v", err)
				}
			},
		},
		{
			name: "success: target already unmounted",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodeUnpublishVolume is failed: %v", err)
				}
			},
		},
		{
			name: "fail: targetPath is missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId: "volumeId",
				}

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodeUnpublishVolume is not failed: %v", err)
				}
			},
		},
		{
			name: "fail: mounter failed to umount",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: -1},
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mountErr := fmt.Errorf("Unmount failed")
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(mountErr)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodeUnpublishVolume is not failed: %v", err)
				}
			},
		},
		{
			name: "fail another operation in-flight on given volumeId-targetPath",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
				}

				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   volumeID,
					TargetPath: targetPath,
				}

				rpcKey := fmt.Sprintf("%s-%s", volumeID, targetPath)

				awsDriver.inFlight.Insert(rpcKey)
				_, err := awsDriver.NodeUnpublishVolume(context.TODO(), req)
				expectErr(t, err, codes.Aborted)
			},
		},
		{
			name: "success: operation in-flight with different volumeId-targetPath",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: -1},
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				rpcKeyAlt := fmt.Sprintf("%s-%s", volumeID, targetPathAlt)
				awsDriver.inFlight.Insert(rpcKeyAlt)

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(nil)

				_, err := awsDriver.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodeUnpublishVolume is failed: %v", err)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

// TestNodeUnpublishVolumeForcefulUnmount covers the forceful-unmount paths. The
// bug being guarded against is a umount(8) that never returns: because
// NodeUnpublishVolume holds a per-volume in-flight lock for its whole duration, a
// permanently blocked unmount wedges the volume until the driver pod restarts. So
// each case asserts the RPC *returns* rather than only what it returns.
func TestNodeUnpublishVolumeForcefulUnmount(t *testing.T) {
	const targetPath = "/target/path"

	// Keep polling fast so the tests do not sit through real intervals.
	origPollInterval := unmountPollInterval
	unmountPollInterval = 10 * time.Millisecond
	defer func() { unmountPollInterval = origPollInterval }()

	// unpublish runs NodeUnpublishVolume and fails the test if it blocks, which is
	// exactly the symptom customers hit.
	unpublish := func(t *testing.T, driver *nodeService) error {
		t.Helper()
		errCh := make(chan error, 1)
		go func() {
			_, err := driver.NodeUnpublishVolume(context.Background(), &csi.NodeUnpublishVolumeRequest{
				VolumeId:   "volumeId",
				TargetPath: targetPath,
			})
			errCh <- err
		}()
		select {
		case err := <-errCh:
			return err
		case <-time.After(30 * time.Second):
			t.Fatal("NodeUnpublishVolume did not return; the in-flight lock would be held indefinitely")
			return nil
		}
	}

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			// forceful-unmount-timeout=0 skips the normal unmount entirely.
			name: "success: timeout zero forces unmount immediately",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: 0},
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mockMounter.EXPECT().UnmountWithForce(gomock.Eq(targetPath), gomock.Eq(time.Duration(0))).Return(nil)
				// No plain Unmount: gomock fails the test if one is attempted.

				if err := unpublish(t, driver); err != nil {
					t.Fatalf("NodeUnpublishVolume failed: %v", err)
				}
			},
		},
		{
			// The regression this whole change exists for: the normal unmount never
			// returns, but the mount point is gone, so polling must notice and let the
			// RPC finish instead of blocking on the stuck umount forever.
			name: "success: hanging unmount released when mount point disappears",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: 10 * time.Second},
				}

				// Mounted at entry and on the first poll, unmounted on later polls.
				gomock.InOrder(
					mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil).Times(2),
					mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil).MinTimes(1),
				)
				// Unmount never returns, mimicking a umount blocked in the kernel.
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).DoAndReturn(func(string) error {
					<-make(chan struct{})
					return nil
				})

				if err := unpublish(t, driver); err != nil {
					t.Fatalf("NodeUnpublishVolume failed: %v", err)
				}
			},
		},
		{
			// IsLikelyNotMountPoint stats the path, so a target removed by kubelet (or
			// by a filesystem that drops the namespace entry before umount returns)
			// reports (true, ENOENT), not (true, nil). Treating that as still-mounted
			// is what made the poll loop useless.
			name: "success: hanging unmount released when target path vanishes",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: 10 * time.Second},
				}

				gomock.InOrder(
					mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil).Times(2),
					mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).
						Return(true, os.ErrNotExist).MinTimes(1),
				)
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).DoAndReturn(func(string) error {
					<-make(chan struct{})
					return nil
				})

				if err := unpublish(t, driver); err != nil {
					t.Fatalf("NodeUnpublishVolume failed: %v", err)
				}
			},
		},
		{
			// Still mounted and unmount still stuck: escalate to umount -f.
			name: "success: hanging unmount escalates to forced unmount",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				timeout := 50 * time.Millisecond
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: timeout},
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil).AnyTimes()
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).DoAndReturn(func(string) error {
					<-make(chan struct{})
					return nil
				})
				mockMounter.EXPECT().UnmountWithForce(gomock.Eq(targetPath), gomock.Eq(timeout)).Return(nil)

				if err := unpublish(t, driver); err != nil {
					t.Fatalf("NodeUnpublishVolume failed: %v", err)
				}
			},
		},
		{
			// umount -f has no timeout upstream, so it can hang too. When it does, the
			// RPC must still return an error rather than hold the lock, letting kubelet
			// retry.
			name: "fail: hanging forced unmount is abandoned instead of held",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				timeout := 50 * time.Millisecond
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: timeout},
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil).AnyTimes()
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).DoAndReturn(func(string) error {
					<-make(chan struct{})
					return nil
				})
				mockMounter.EXPECT().UnmountWithForce(gomock.Eq(targetPath), gomock.Any()).
					DoAndReturn(func(string, time.Duration) error {
						<-make(chan struct{})
						return nil
					})

				if err := unpublish(t, driver); err == nil {
					t.Fatal("NodeUnpublishVolume succeeded despite a forced unmount that never returned")
				}

				// The lock must be free so kubelet's retry is not rejected with Aborted.
				rpcKey := fmt.Sprintf("%s-%s", "volumeId", targetPath)
				if ok := driver.inFlight.Insert(rpcKey); !ok {
					t.Fatal("in-flight lock still held after NodeUnpublishVolume returned")
				}
			},
		},
		{
			// Feature disabled: behavior must be byte-for-byte the old blocking path.
			name: "success: negative timeout uses plain blocking unmount",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				defer mockCtl.Finish()

				mockMounter := driverMocks.NewMockMounter(mockCtl)
				driver := &nodeService{
					mounter:       mockMounter,
					inFlight:      internal.NewInFlight(),
					driverOptions: &DriverOptions{forcefulUnmountTimeout: -1},
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(nil)

				if err := unpublish(t, driver); err != nil {
					t.Fatalf("NodeUnpublishVolume failed: %v", err)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestRemoveNotReadyTaint(t *testing.T) {
	nodeName := "test-node-123"
	testCases := []struct {
		name      string
		setup     func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error)
		expResult error
	}{
		{
			name: "missing CSI_NODE_NAME",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				return func() (kubernetes.Interface, error) {
					t.Fatalf("Unexpected call to k8s client getter")
					return nil, nil
				}
			},
			expResult: nil,
		},
		{
			name: "failed to setup k8s client",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				t.Setenv("CSI_NODE_NAME", nodeName)
				return func() (kubernetes.Interface, error) {
					return nil, fmt.Errorf("Failed setup!")
				}
			},
			expResult: nil,
		},
		{
			name: "failed to get node",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				t.Setenv("CSI_NODE_NAME", nodeName)
				getNodeMock, _ := getNodeMock(mockCtl, nodeName, nil, fmt.Errorf("Failed to get node!"))

				return func() (kubernetes.Interface, error) {
					return getNodeMock, nil
				}
			},
			expResult: fmt.Errorf("Failed to get node!"),
		},
		{
			name: "no taints to remove",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				t.Setenv("CSI_NODE_NAME", nodeName)
				getNodeMock, _ := getNodeMock(mockCtl, nodeName, &corev1.Node{}, nil)

				return func() (kubernetes.Interface, error) {
					return getNodeMock, nil
				}
			},
			expResult: nil,
		},
		{
			name: "failed to patch node",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				t.Setenv("CSI_NODE_NAME", nodeName)
				getNodeMock, mockNode := getNodeMock(mockCtl, nodeName, &corev1.Node{
					Spec: corev1.NodeSpec{
						Taints: []corev1.Taint{
							{
								Key:    AgentNotReadyNodeTaintKey,
								Effect: "NoExecute",
							},
						},
					},
				}, nil)
				mockNode.EXPECT().Patch(gomock.Any(), gomock.Eq(nodeName), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("Failed to patch node!"))

				return func() (kubernetes.Interface, error) {
					return getNodeMock, nil
				}
			},
			expResult: fmt.Errorf("Failed to patch node!"),
		},
		{
			name: "success",
			setup: func(t *testing.T, mockCtl *gomock.Controller) func() (kubernetes.Interface, error) {
				t.Setenv("CSI_NODE_NAME", nodeName)
				getNodeMock, mockNode := getNodeMock(mockCtl, nodeName, &corev1.Node{
					Spec: corev1.NodeSpec{
						Taints: []corev1.Taint{
							{
								Key:    AgentNotReadyNodeTaintKey,
								Effect: "NoSchedule",
							},
						},
					},
				}, nil)
				mockNode.EXPECT().Patch(gomock.Any(), gomock.Eq(nodeName), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

				return func() (kubernetes.Interface, error) {
					return getNodeMock, nil
				}
			},
			expResult: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			k8sClientGetter := tc.setup(t, mockCtl)
			result := removeNotReadyTaint(k8sClientGetter)

			if !reflect.DeepEqual(result, tc.expResult) {
				t.Fatalf("Expected result `%v`, got result `%v`", tc.expResult, result)
			}
		})
	}
}

func TestRemoveTaintInBackground(t *testing.T) {
	mockRemovalCount := 0
	mockRemovalFunc := func(_ cloud.KubernetesAPIClient) error {
		mockRemovalCount += 1
		if mockRemovalCount == 3 {
			return nil
		} else {
			return fmt.Errorf("Taint removal failed!")
		}
	}

	removeTaintInBackground(nil, mockRemovalFunc)
	assert.Equal(t, mockRemovalCount, 3)
}

func getNodeMock(mockCtl *gomock.Controller, nodeName string, returnNode *corev1.Node, returnError error) (kubernetes.Interface, *driverMocks.MockNodeInterface) {
	mockClient := driverMocks.NewMockKubernetesClient(mockCtl)
	mockCoreV1 := driverMocks.NewMockCoreV1Interface(mockCtl)
	mockNode := driverMocks.NewMockNodeInterface(mockCtl)

	mockClient.EXPECT().CoreV1().Return(mockCoreV1).MinTimes(1)
	mockCoreV1.EXPECT().Nodes().Return(mockNode).MinTimes(1)
	mockNode.EXPECT().Get(gomock.Any(), gomock.Eq(nodeName), gomock.Any()).Return(returnNode, returnError).MinTimes(1)

	return mockClient, mockNode
}

func expectErr(t *testing.T, actualErr error, expectedCode codes.Code) {
	if actualErr == nil {
		t.Fatalf("Expect error but got no error")
	}

	status, ok := status.FromError(actualErr)
	if !ok {
		t.Fatalf("Failed to get error status code from error: %v", actualErr)
	}

	if status.Code() != expectedCode {
		t.Fatalf("Expected error code %d, got %d message %s", expectedCode, status.Code(), status.Message())
	}
}
