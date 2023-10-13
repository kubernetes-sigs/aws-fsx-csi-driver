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
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver/internal"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{}
)

// VolumeOperationAlreadyExists is message fmt returned to CO when there is another in-flight call on the given volumeID
const VolumeOperationAlreadyExists = "An operation with the given volume=%q is already in progress"

// Configurations for node taint removal retry mechanism
const (
	RemoveNotReadyTaintRetryTrue     = 11
	RemoveNotReadyNumTaintRetryFalse = 2
	RemoveNotReadyTaintPollDelay     = 1 * time.Second
	RemoveNotReadyTaintPollFactor    = 1.5
	RemoveNotReadyTimeout            = 90 * time.Second
)

type nodeService struct {
	metadata      cloud.MetadataService
	mounter       Mounter
	inFlight      *internal.InFlight
	driverOptions *DriverOptions
}

func newNodeService(driverOptions *DriverOptions) nodeService {
	klog.V(5).InfoS("[Debug] Retrieving node info from metadata service")

	region := os.Getenv("AWS_REGION")
	metadata, err := cloud.NewMetadataService(cloud.DefaultEC2MetadataClient, cloud.DefaultKubernetesAPIClient, region)
	if err != nil {
		panic(err)
	}
	klog.InfoS("regionFromSession Node service", "region", metadata.GetRegion())

	nodeMounter, err := newNodeMounter()
	if err != nil {
		panic(err)
	}

	// Remove taint from node to indicate driver startup success
	// This is done at the last possible moment to prevent race conditions or false positive removals
	err = removeNotReadyTaint(cloud.DefaultKubernetesAPIClient, driverOptions.retryTaintRemoval)
	if err != nil {
		klog.ErrorS(err, "Unexpected failure when attempting to remove node taint(s)")
	}

	return nodeService{
		metadata:      metadata,
		mounter:       nodeMounter,
		inFlight:      internal.NewInFlight(),
		driverOptions: driverOptions,
	}
}

func (d *nodeService) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *nodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *nodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(4).InfoS("NodePublishVolume: called with", "args", *req)

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	context := req.GetVolumeContext()
	dnsname := context[volumeContextDnsName]
	mountname := context[volumeContextMountName]

	if len(dnsname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "dnsname is not provided")
	}

	if len(mountname) == 0 {
		mountname = "fsx"
	}

	source := fmt.Sprintf("%s@tcp:/%s", dnsname, mountname)

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	if ok := d.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, VolumeOperationAlreadyExists, volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodePublishVolume: volume operation finished", "volumeId", volumeID)
		d.inFlight.Delete(volumeID)
	}()

	mountOptions := []string{}
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	if m := volCap.GetMount(); m != nil {
		hasOption := func(options []string, opt string) bool {
			for _, o := range options {
				if o == opt {
					return true
				}
			}
			return false
		}
		for _, f := range m.MountFlags {
			if !hasOption(mountOptions, f) {
				mountOptions = append(mountOptions, f)
			}
		}
	}
	klog.V(5).InfoS("NodePublishVolume: creating", "dir", target)
	if err := d.mounter.MakeDir(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", target, err)
	}

	//Checking if the target directory is already mounted with a volume.
	mounted, err := d.isMounted(source, target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if %q is mounted: %v", target, err)
	}
	if !mounted {
		klog.V(4).InfoS("NodePublishVolume: mounting", "source", source, "target", target, "mountOptions", mountOptions)
		if err := d.mounter.Mount(source, target, "lustre", mountOptions); err != nil {
			os.Remove(target)
			return nil, status.Errorf(codes.Internal, "Could not mount %q at %q: %v", source, target, err)
		}
		klog.V(5).InfoS("NodePublishVolume: was mounted", "target", target)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *nodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(4).InfoS("NodeUnpublishVolume: called", "args", *req)

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}
	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	if ok := d.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, VolumeOperationAlreadyExists, volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodeUnpublishVolume: volume operation finished", "volumeId", volumeID)
		d.inFlight.Delete(volumeID)
	}()

	// Check if the target is mounted before unmounting
	notMnt, _ := d.mounter.IsLikelyNotMountPoint(target)
	if notMnt {
		klog.V(5).InfoS("NodeUnpublishVolume: target path not mounted, skipping unmount", "target", target)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	klog.V(5).InfoS("NodeUnpublishVolume: unmounting", "target", target)
	err := d.mounter.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *nodeService) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *nodeService) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *nodeService) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(4).InfoS("NodeGetCapabilities: called", "args", *req)
	var caps []*csi.NodeServiceCapability
	for _, cap := range nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (d *nodeService) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(4).InfoS("NodeGetInfo: called", "args", *req)

	return &csi.NodeGetInfoResponse{
		NodeId: d.metadata.GetInstanceID(),
	}, nil
}

// isMounted checks if target is mounted. It does NOT return an error if target
// doesn't exist.
func (d *nodeService) isMounted(source string, target string) (bool, error) {
	/*
		Checking if it's a mount point using IsLikelyNotMountPoint. There are three different return values,
		1. true, err when the directory does not exist or corrupted.
		2. false, nil when the path is already mounted with a device.
		3. true, nil when the path is not mounted with any device.
	*/
	klog.V(4).InfoS(target)
	notMnt, err := d.mounter.IsLikelyNotMountPoint(target)
	if err != nil && !os.IsNotExist(err) {
		//Checking if the path exists and error is related to Corrupted Mount, in that case, the system could unmount and mount.
		_, pathErr := d.mounter.PathExists(target)
		if pathErr != nil && d.mounter.IsCorruptedMnt(pathErr) {
			klog.V(4).InfoS("NodePublishVolume: Target path is a corrupted mount. Trying to unmount.", "target", target)
			if mntErr := d.mounter.Unmount(target); mntErr != nil {
				return false, status.Errorf(codes.Internal, "Unable to unmount the target %q : %v", target, mntErr)
			}
			//After successful unmount, the device is ready to be mounted.
			return false, nil
		}
		return false, status.Errorf(codes.Internal, "Could not check if %q is a mount point: %v, %v", target, err, pathErr)
	}

	// Do not return os.IsNotExist error. Other errors were handled above.  The
	// Existence of the target should be checked by the caller explicitly and
	// independently because sometimes prior to mount it is expected not to exist
	// (in Windows, the target must NOT exist before a symlink is created at it)
	// and in others it is an error (in Linux, the target mount directory must
	// exist before mount is called on it)
	if err != nil && os.IsNotExist(err) {
		klog.V(5).InfoS("[Debug] NodePublishVolume: Target path does not exist", "target", target)
		return false, nil
	}

	if !notMnt {
		klog.V(4).InfoS("NodePublishVolume: Target path is already mounted", "target", target)
	}

	return !notMnt, nil
}

// Struct for JSON patch operations
type JSONPatch struct {
	OP    string      `json:"op,omitempty"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value"`
}

// removeNotReadyTaint removes the taint fsx.csi.aws.com/agent-not-ready from the local node
// This taint can be optionally applied by users to prevent startup race conditions such as
// https://github.com/kubernetes/kubernetes/issues/95911
func removeNotReadyTaint(k8sClient cloud.KubernetesAPIClient, retryTaintRemoval bool) error {
	nodeName := os.Getenv("CSI_NODE_NAME")
	if nodeName == "" {
		klog.V(4).InfoS("CSI_NODE_NAME missing, skipping taint removal")
		return nil
	}

	numSteps := RemoveNotReadyNumTaintRetryFalse
	if retryTaintRemoval {
		numSteps = RemoveNotReadyTaintRetryTrue
	}

	backoff := wait.Backoff{
		Duration: RemoveNotReadyTaintPollDelay,
		Factor:   RemoveNotReadyTaintPollFactor,
		Steps:    numSteps,
		Cap:      RemoveNotReadyTimeout,
	}

	var mostRecentError error

	removeTaint := func() (bool, error) {
		clientset, err := k8sClient()

		if err != nil {
			mostRecentError = err
			klog.V(4).InfoS("Failed to communicate with k8s API")
			return false, nil
		}

		node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			mostRecentError = err
			klog.ErrorS(err, "Error when collecting node information.")
			return false, nil
		}

		var taintsToKeep []corev1.Taint
		for _, taint := range node.Spec.Taints {
			if taint.Key != AgentNotReadyNodeTaintKey {
				taintsToKeep = append(taintsToKeep, taint)
			} else {
				klog.V(4).InfoS("Queued taint for removal", "key", taint.Key, "effect", taint.Effect)
			}
		}

		if len(taintsToKeep) == len(node.Spec.Taints) {
			mostRecentError = err
			klog.V(4).InfoS("No taints to remove on node, skipping taint removal")
			return true, nil
		}

		patchRemoveTaints := []JSONPatch{
			{
				OP:    "test",
				Path:  "/spec/taints",
				Value: node.Spec.Taints,
			},
			{
				OP:    "replace",
				Path:  "/spec/taints",
				Value: taintsToKeep,
			},
		}

		patch, err := json.Marshal(patchRemoveTaints)
		if err != nil {
			mostRecentError = err
			klog.ErrorS(err, "Error when marshalling taints.")
			return false, nil
		}

		_, err = clientset.CoreV1().Nodes().Patch(context.Background(), nodeName, k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			mostRecentError = err
			klog.ErrorS(err, "Error when executing patch.")
			return false, nil
		}
		klog.InfoS("Removed taint(s) from local node", "node", nodeName)
		mostRecentError = nil
		return true, nil
	}

	err := wait.ExponentialBackoff(backoff, removeTaint)
	if mostRecentError != nil {
		klog.ErrorS(err, "Node taint removal failed to complete within the specified number of attempts.")
		return mostRecentError
	}
	return nil
}
