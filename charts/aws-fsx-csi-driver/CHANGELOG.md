# Helm chart

# v1.5.0
* Use driver 0.9.0

# v1.4.4
* Use driver 0.8.3

# v1.4.3
* Added option to configure `fsGroupPolicy` on the CSIDriver object. Adding such a configuration allows kubelet to change ownership of every file in the volume at mount time.
Documentation on fsGroupPolicy can be found [here](https://kubernetes-csi.github.io/docs/support-fsgroup.html).

**Side-note**: Setting fsGroupPolicy to `File` in for configurations that mount the disk on multiple nodes as the same time can lead to race-conditions and subsequently deadlocks, unless if **every** Pod mounting the volume has the same *securityContext* which includes the setting `fsGroupChangePolicy: "OnRootMismatch"`

# v1.4.2
* Use driver 0.8.2

# v1.4.1
* Use driver 0.8.1

# v1.4.0
* Use driver 0.8.0

# v1.3.2
* Update ECR sidecars to 1-18-13

# v1.3.1
* Use driver 0.7.1

# v1.3.0
* Use driver 0.7.0

# v1.2.0
* Use driver 0.6.0
* Add sidecar for storage scaling (external-resizer)

# v1.1.0
* Use driver 0.5.0

# v1.0.0
* Remove support for Helm 2
* Reorganize values to be more consistent with EFS and EBS helm charts
  * controllerService -> controller
  * nodeService -> node
* Add node.serviceAccount
* Add dnsPolicy and dnsConfig
* Add imagePullSecrets
* Add controller.tolerations, node.tolerations, and node.tolerateAllTaints
* Remove extraArgs, securityContext, podSecurityContext 
* Bump sidecar images to support kubernetes >=1.20
* Require kubernetes >=1.17
