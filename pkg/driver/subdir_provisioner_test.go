package driver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/mocks"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/util"
)

func TestCreateVolumeBySubDirProvisioner(t *testing.T) {
	var (
		endpoint    = "endpoint"
		volumeName  = "volumeName"
		dnsName     = "test.fsx.us-west-2.amazoawd.com"
		mountName   = "random"
		baseDir     = "base"
		volCapacity = util.GiBToBytes(100)
		stdVolCap   = &csi.VolumeCapability{
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
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   dnsName,
						volumeParamsMountName: mountName,
						volumeParamsBaseDir:   baseDir,
					},
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s/%s", dnsName, mountName, baseDir)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if expected := strings.Join([]string{dnsName, mountName, baseDir, volumeName, volumeName}, volumeIDSeparator); resp.Volume.VolumeId != expected {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, expected)
				}

				if resp.Volume.CapacityBytes != volCapacity {
					t.Fatalf("resp.Volume.CapacityGiB mismatches. actual: %v expected: %v", resp.Volume.CapacityBytes, volCapacity)
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				basedir, exists := resp.Volume.VolumeContext[volumeContextBaseDir]
				if !exists {
					t.Fatal("baseDir is missing")
				}

				if basedir != baseDir {
					t.Fatalf("baseDir mismatches. actual: %v expected: %v", basedir, baseDir)
				}

				subdir, exists := resp.Volume.VolumeContext[volumeContextSubDir]
				if !exists {
					t.Fatal("subdir is missing")
				}

				if subdir != volumeName {
					t.Fatalf("subDir mismatches. actual: %v expected: %v", subdir, volumeName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with empty baseDir",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   dnsName,
						volumeParamsMountName: mountName,
						volumeParamsBaseDir:   "",
					},
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s", dnsName, mountName)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if expected := strings.Join([]string{dnsName, mountName, "", volumeName, volumeName}, volumeIDSeparator); resp.Volume.VolumeId != expected {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, expected)
				}

				if resp.Volume.CapacityBytes != volCapacity {
					t.Fatalf("resp.Volume.CapacityGiB mismatches. actual: %v expected: %v", resp.Volume.CapacityBytes, volCapacity)
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				basedir, exists := resp.Volume.VolumeContext[volumeContextBaseDir]
				if !exists {
					t.Fatal("baseDir is missing")
				}

				if basedir != "" {
					t.Fatalf("baseDir has to be empty. actual: %v", basedir)
				}

				subdir, exists := resp.Volume.VolumeContext[volumeContextSubDir]
				if !exists {
					t.Fatal("subdir is missing")
				}

				if subdir != volumeName {
					t.Fatalf("subDir mismatches. actual: %v expected: %v", subdir, volumeName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal without baseDir",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   dnsName,
						volumeParamsMountName: mountName,
					},
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s", dnsName, mountName)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if expected := strings.Join([]string{dnsName, mountName, "", volumeName, volumeName}, volumeIDSeparator); resp.Volume.VolumeId != expected {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, expected)
				}

				if resp.Volume.CapacityBytes != volCapacity {
					t.Fatalf("resp.Volume.CapacityGiB mismatches. actual: %v expected: %v", resp.Volume.CapacityBytes, volCapacity)
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				basedir, exists := resp.Volume.VolumeContext[volumeContextBaseDir]
				if !exists {
					t.Fatal("baseDir is missing")
				}

				if basedir != "" {
					t.Fatalf("baseDir has to be empty. actual: %v", basedir)
				}

				subdir, exists := resp.Volume.VolumeContext[volumeContextSubDir]
				if !exists {
					t.Fatal("subdir is missing")
				}

				if subdir != volumeName {
					t.Fatalf("subDir mismatches. actual: %v expected: %v", subdir, volumeName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with kubernetes external provisioner's parameter key",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:                            dnsName,
						volumeParamsMountName:                          mountName,
						volumeParamsBaseDir:                            baseDir,
						kubernetesExternalProvisionerKeyPrefix + "any": "any",
					},
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s/%s", dnsName, mountName, baseDir)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				resp, err := driver.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("CreateVolume is failed: %v", err)
				}

				if resp.Volume == nil {
					t.Fatal("resp.Volume is nil")
				}

				if expected := strings.Join([]string{dnsName, mountName, baseDir, volumeName, volumeName}, volumeIDSeparator); resp.Volume.VolumeId != expected {
					t.Fatalf("VolumeId mismatches. actual: %v expected: %v", resp.Volume.VolumeId, expected)
				}

				if resp.Volume.CapacityBytes != volCapacity {
					t.Fatalf("resp.Volume.CapacityGiB mismatches. actual: %v expected: %v", resp.Volume.CapacityBytes, volCapacity)
				}

				dnsname, exists := resp.Volume.VolumeContext[volumeContextDnsName]
				if !exists {
					t.Fatal("dnsname is missing")
				}

				if dnsname != dnsName {
					t.Fatalf("dnsname mismatches. actual: %v expected: %v", dnsname, dnsName)
				}

				mountname, exists := resp.Volume.VolumeContext[volumeContextMountName]
				if !exists {
					t.Fatal("mountname is missing")
				}

				if mountname != mountName {
					t.Fatalf("mountname mismatches. actual: %v expected: %v", mountname, mountName)
				}

				basedir, exists := resp.Volume.VolumeContext[volumeContextBaseDir]
				if !exists {
					t.Fatal("baseDir is missing")
				}

				if basedir != baseDir {
					t.Fatalf("baseDir mismatches. actual: %v expected: %v", basedir, baseDir)
				}

				subdir, exists := resp.Volume.VolumeContext[volumeContextSubDir]
				if !exists {
					t.Fatal("subdir is missing")
				}

				if subdir != volumeName {
					t.Fatalf("subDir mismatches. actual: %v expected: %v", subdir, volumeName)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: dnsname is empty",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   "",
						volumeParamsMountName: mountName,
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mountname is empty",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:    dnsName,
						volumeContextMountName: "",
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mountname missing",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName: dnsName,
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: invalid parameter is passed",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   dnsName,
						volumeParamsMountName: mountName,
						"invalidParam":        "invalid",
					},
				}

				ctx := context.Background()
				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mkdir after mounted",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				req := &csi.CreateVolumeRequest{
					Name:          volumeName,
					CapacityRange: &csi.CapacityRange{RequiredBytes: volCapacity},
					VolumeCapabilities: []*csi.VolumeCapability{
						stdVolCap,
					},
					Parameters: map[string]string{
						volumeParamsDnsName:   dnsName,
						volumeParamsMountName: mountName,
						volumeParamsBaseDir:   baseDir,
					},
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s/%s", dnsName, mountName, baseDir)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(&os.PathError{Op: "mkdir", Path: gomock.Any().String(), Err: syscall.ENOTDIR})
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				_, err := driver.CreateVolume(ctx, req)
				if err == nil {
					t.Fatal("CreateVolume is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDeleteVolumeBySubDirProvisioner(t *testing.T) {
	var (
		endpoint   = "endpoint"
		volumeName = "volumeName"
		dnsName    = "test.fsx.us-west-2.amazoawd.com"
		mountName  = "random"
		baseDir    = "base"
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				volumeID := strings.Join([]string{dnsName, mountName, baseDir, volumeName, volumeName}, volumeIDSeparator)
				req := &csi.DeleteVolumeRequest{
					VolumeId: volumeID,
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s/%s", dnsName, mountName, baseDir)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with empty baseDir",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				volumeID := strings.Join([]string{dnsName, mountName, "", volumeName, volumeName}, volumeIDSeparator)
				req := &csi.DeleteVolumeRequest{
					VolumeId: volumeID,
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s", dnsName, mountName)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "success: normal with empty subDir",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				volumeID := strings.Join([]string{dnsName, mountName, baseDir, "", volumeName}, volumeIDSeparator)
				req := &csi.DeleteVolumeRequest{
					VolumeId: volumeID,
				}

				ctx := context.Background()
				source := fmt.Sprintf("%s@tcp:/%s/%s", dnsName, mountName, baseDir)

				mockMounter.EXPECT().MakeDir(gomock.Not("")).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(true, nil)
				mockMounter.EXPECT().Mount(source, gomock.Not(""), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Not("")).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Not("")).Return(nil)

				_, err := driver.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("DeleteVolume is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: dnsname missing in volume ID",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				volumeID := strings.Join([]string{"", mountName, baseDir, volumeName, volumeName}, volumeIDSeparator)
				req := &csi.DeleteVolumeRequest{
					VolumeId: volumeID,
				}
				deleteError := status.Error(codes.InvalidArgument, "dnsname is not provided")

				ctx := context.Background()
				_, err := driver.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatal("DeleteVolume is not failed")
				}
				if err.Error() != deleteError.Error() {
					t.Fatalf("DeleteVolume returned error [%v], expected [%v]", err, deleteError)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: mountname missing in volume ID",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtl)

				driver := &Driver{
					endpoint: endpoint,
					mounter:  mockMounter,
				}

				volumeID := strings.Join([]string{dnsName, "", baseDir, volumeName, volumeName}, volumeIDSeparator)
				req := &csi.DeleteVolumeRequest{
					VolumeId: volumeID,
				}
				deleteError := status.Error(codes.InvalidArgument, "mountname is not provided")

				ctx := context.Background()
				_, err := driver.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatal("DeleteVolume is not failed")
				}
				if err.Error() != deleteError.Error() {
					t.Fatalf("DeleteVolume returned error [%v], expected [%v]", err, deleteError)
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
