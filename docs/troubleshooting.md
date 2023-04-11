## Troubleshooting CSI Driver (Common Issues)

If you’re experiencing issues with the Lustre CSI Driver, always ensure that you’re using the [latest available Lustre CSI Driver](https://github.com/kubernetes-sigs/aws-fsx-csi-driver#csi-specification-compatibility-matrix). 
You can check which version of the CSI Driver you’re running in the pods on your cluster by checking the fsx-plugin container image version in your cluster’s fsx-csi-node or fsx-csi-controller pods either via AWS EKS console or by calling `kubectl describe pod <pod-name>`. 

If you’re not using the latest image, please upgrade the CSI Driver image you’re currently using on the pods in your cluster and see if the issue persists.


### Troubleshooting Issues

For common Lustre issues, you can refer to the [AWS Lustre troubleshooting guide](https://docs.aws.amazon.com/fsx/latest/LustreGuide/troubleshooting.html) for more details as it includes mitigations for common problems with FSx for Lustre.

#### Issue: Pod is stuck in ContainerCreating when trying to mount a volume.

##### Characteristics:

1. The underlying file system has a large number of files
2. When calling `kubectl get pod <pod-name>` you see an error message similar to this:
```
Warning  FailedMount  kubelet    Unable to attach or mount volumes: unmounted volumes=[fsx-volume-name], unattached volumes=[fsx-volume-name]: timed out waiting for the condition
```

##### Likely Cause:
Volume ownership is being set recursively on every file in the volume, which prevents the pod from mounting the volume for an extended period of time. See https://github.com/kubernetes/kubernetes/issues/69699

##### Mitigation:
[Per Kubernetes documentation](https://kubernetes.io/blog/2020/12/14/kubernetes-release-1.20-fsgroupchangepolicy-fsgrouppolicy/#allow-users-to-skip-recursive-permission-changes-on-mount): “When configuring a pod’s security context, set fsGroupChangePolicy to "OnRootMismatch" so if the root of the volume already has the correct permissions, the recursive permission change can be skipped." After setting this policy, terminate the pod stuck in ContainerCreating and drain the node. Pod-level mounting on the new node should no longer have issues mounting if the volume root has the correct permissions.

For more information on configuring securityContext, see https://kubernetes.io/docs/tasks/configure-pod-container/security-context/.

  

#### Issue: Pods fail to mount file system with the following error:

```
mount.lustre: mount ipaddress@tcp:/mountname at /fsx failed: Input/output error
Is the MGS running?
```

##### Likely Cause:
FSx for Lustre rejects packets where the source port is neither 988 nor in the range 1018–1023. It may be that kube-proxy is redirecting the packet to a different port.

##### Mitigation:
Run netstat -tlpna to confirm whether there are TCP connections established with a source port outside of the range 1018–1023.  If there are such connections, enable SNAT to avoid redirecting packets to a different port. For more information, see https://docs.aws.amazon.com/eks/latest/userguide/external-snat.html

  

#### Issue: Pods are stuck in terminating when the [cluster autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler) is scaling down resources.

##### Likely Cause:
A [July 2021 autoscaler change](https://github.com/kubernetes/autoscaler/pull/4172) introduced a [known issue](https://github.com/kubernetes/autoscaler/issues/5240) where daemonset pods are evicted at the same time as non-daemonset pods, which can cause a race condition where when daemonset pods are evicted prior to the non-daemonset pods, the non-daemonset pods are unable to unmount gracefully and are stuck in terminating.

##### Mitigation:
Annotate Daemonset pods with `“cluster-autoscaler.kubernetes.io/enable-ds-eviction": "false"` , which will prevent the Daemonset pods from being evicted on resource scale-down and allow for the non-DS pods to unmount properly. This can be done by annotating the [pod spec of the Daemonset file](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#how-can-i-enabledisable-eviction-for-a-specific-daemonset).


