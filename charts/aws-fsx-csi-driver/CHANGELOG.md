# Helm chart

# v1.0.0
* Remove support for Helm 2
* Reorganize values to be more consistent with EFS and EBS helm charts
  * controllerService -> controller
  * nodeService -> node
* Add node.serviceAccount
* Add dnsPolicy and dnsConfig
* Add imagePullSecrets
* Remove extraArgs, securityContext, podSecurityContext 
* Bump sidecar images to support kubernetes >=1.20
* Require kubernetes >=1.17
