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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
	cloudMock "sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud/mocks"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/internal"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				driver := &nodeService{
					metadata: mockMetadata,
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					metadata: mockMetadata,
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

				mockMetadata := cloudMock.NewMockMetadataService(mockCtl)
				mockMounter := driverMocks.NewMockMounter(mockCtl)

				awsDriver := &nodeService{
					metadata: mockMetadata,
					mounter:  mockMounter,
					inFlight: internal.NewInFlight(),
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
