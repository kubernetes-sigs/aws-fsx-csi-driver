# Helm chart

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
