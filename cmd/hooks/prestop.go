// Copyright 2025 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hooks

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver"
)

/*
When a node is terminated without first unmounting clients, this leaves stranded clients server side, which in turn need to be
evicted by the Lustre filesystem.

This PreStop lifecycle hook aims to ensure that before the node (and the CSI driver node pod running on it) is shut down,
there are no more pods with PVCs on the node, thereby indicating that all volumes have been successfully unmounted and detached.

No unnecessary delay is added to the termination workflow, as the PreStop hook logic is only executed when the node is being drained
(thus preventing delays in termination where the node pod is killed due to a rolling restart, or during driver upgrades, but the workload pods are expected to be running).
If the PreStop hook hangs during its execution, the driver node pod will be forcefully terminated after terminationGracePeriodSeconds.
*/

const clusterAutoscalerTaint = "ToBeDeletedByClusterAutoscaler"
const v1KarpenterTaint = "karpenter.sh/disrupted"
const v1beta1KarpenterTaint = "karpenter.sh/disruption"

// drainTaints includes taints used by K8s or autoscalers that signify node draining or pod eviction
var drainTaints = map[string]struct{}{
	v1.TaintNodeUnschedulable: {}, // Kubernetes common eviction taint (kubectl drain)
	clusterAutoscalerTaint:    {},
	v1KarpenterTaint:          {},
	v1beta1KarpenterTaint:     {},
}

func PreStop(clientset kubernetes.Interface) error {
	klog.InfoS("PreStop: executing PreStop lifecycle hook")

	nodeName := os.Getenv("CSI_NODE_NAME")
	if nodeName == "" {
		return fmt.Errorf("PreStop: CSI_NODE_NAME missing")
	}

	node, err := fetchNode(clientset, nodeName)
	if err != nil {
		return err
	}

	if isNodeBeingDrained(node) {
		klog.InfoS("PreStop: node is being drained, checking for remaining pods with PVCs", "node", nodeName)
		return waitForPodShutdowns(clientset, nodeName)
	}

	klog.InfoS("PreStop: node is not being drained, skipping pods check", "node", nodeName)
	return nil
}

func fetchNode(clientset kubernetes.Interface, nodeName string) (*v1.Node, error) {
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetchNode: failed to retrieve node information: %w", err)
	}
	return node, nil
}

// isNodeBeingDrained returns true if node resource has a known drain/eviction taint.
func isNodeBeingDrained(node *v1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if _, isDrainTaint := drainTaints[taint.Key]; isDrainTaint {
			return true
		}
	}
	return false
}

func waitForPodShutdowns(clientset kubernetes.Interface, nodeName string) error {
	allVolumesUnmounted := make(chan struct{})

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("spec.nodeName=%s", nodeName)
		}))
	informer := factory.Core().V1().Pods().Informer()

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			klog.V(5).InfoS("DeleteFunc: Pod deleted", "node", nodeName)
			if err := checkActivePods(clientset, nodeName, allVolumesUnmounted); err != nil {
				klog.ErrorS(err, "DeleteFunc: error checking active pods on the node")
			}

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			klog.V(5).InfoS("UpdateFunc: Pod updated", "node", nodeName)
			if err := checkActivePods(clientset, nodeName, allVolumesUnmounted); err != nil {
				klog.ErrorS(err, "UpdateFunc: error checking active pods on the node")
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler to Node informer: %w", err)
	}

	go informer.Run(allVolumesUnmounted)

	if err := checkActivePods(clientset, nodeName, allVolumesUnmounted); err != nil {
		klog.ErrorS(err, "waitForPodShutdowns: error checking active pods on the node")
	}

	<-allVolumesUnmounted
	klog.InfoS("waitForPodShutdowns: finished waiting for active pods on the node. preStopHook completed")
	return nil
}

func checkActivePods(clientset kubernetes.Interface, nodeName string, allVolumesUnmounted chan struct{}) error {
	podList, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return fmt.Errorf("checkActivePods: failed to get podList: %w", err)
	}
	for _, pod := range podList.Items {
		// Temporary workaround until FieldSelector filters properly: https://github.com/kubernetes/client-go/issues/1350
		if pod.Spec.NodeName == nodeName {
			for _, vol := range pod.Spec.Volumes {

				if vol.PersistentVolumeClaim == nil {
					continue
				}

				pvcName := vol.PersistentVolumeClaim.ClaimName
				pvc, err := clientset.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("checkActivePods: failed to get pv %s: %w", pvcName, err)
				}
				pvName := pvc.Spec.VolumeName
				pv, err := clientset.CoreV1().PersistentVolumes().Get(context.Background(), pvName, metav1.GetOptions{})

				if err != nil {
					return fmt.Errorf("checkActivePods: failed to get pv %s: %w", pvName, err)
				}

				if pv.Spec.CSI != nil && pv.Spec.CSI.Driver == driver.DriverName {
					klog.InfoS("checkActivePods: not ready to exit, found PV associated with pod", "PV", pvName, "node", nodeName)
					return nil
				}
			}
		}
	}

	close(allVolumesUnmounted)
	klog.V(5).Info("checkActivePods: no pods associated with PVCs identified", "node", nodeName)
	return nil
}
