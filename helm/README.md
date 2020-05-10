
## Amazon FSx for Lustre CSI Driver Helm Chart
For more info view [aws-fsx-csi-driver](https://github.com/kubernetes-sigs/aws-fsx-csi-driver)

## Prerequisites
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.
## Install chart
```shell script
helm install . --name aws-fsx-csi-driver
```

## Upgrade release
```shell script
helm upgrade aws-fsx-csi-driver \
    --install . \
    --version 0.1.0 \
    --namespace kube-system \
    -f values.yaml
```
## Uninstalling the Chart
```shell script
helm delete aws-fsx-csi-driver --namespace [NAMESPACE]
```
## Parameters

The following table lists the configurable parameters of the aws-fsx-csi-driver chart and their default values.

| Parameter                                             | Description                                                   | Default                                    |
| ------------------------------------------------------| ------------------------------------------------------------- | ------------------------------------------ |
| `controllerService.replicaCount`                      | Num of replicas for controller                                | `2`                                        |
| `controllerService.nodeSelector`                      | Controllers node selector                                     | `kubernetes.io/os: linux`             |
| `controllerService.podSecurityContext`                | Security context for controller pods                          | `{}`                                       |
|                                                       |                                                               |                                            |
| `controllerService.fsxPlugin.image.repository`        | aws-fsx-csi-driver image name                                 | `amazon/aws-fsx-csi-driver`                |
| `controllerService.fsxPlugin.image.tag`               | aws-fsx-csi-driver image tag                                  | `latest`                                   |
| `controllerService.fsxPlugin.image.pullPolicy`        | aws-fsx-csi-driver image pull policy                          | `IfNotPresent`                             |
| `controllerService.fsxPlugin.extraArgs`               | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--logtostderr --v=5`                      |
| `controllerService.fsxPlugin.securityContext`         | Security context for the container                            | `{}`                                       |
| `controllerService.fsxPlugin.resources`               | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                       |                                                               |                                            |
| `controllerService.csiProvisioner.image.repository`   | csi-provisioner image name                                    | `quay.io/k8scsi/csi-provisioner`           |
| `controllerService.csiProvisioner.image.tag`          | csi-provisioner image tag                                     | `v1.3.0`                                   |
| `controllerService.csiProvisioner.image.pullPolicy`   | csi-provisioner image pull policy                             | `IfNotPresent`                             |
| `controllerService.csiProvisioner.extraArgs`          | Extra arguments to be passed to csi-provisioner               | `--timeout=5m --v=5 --enable-leader-election --leader-election-type=leases`|
| `controllerService.csiProvisioner.securityContext`    | Security context for the container                            | `{}`                                       |
| `controllerService.csiProvisioner.resources`          | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                       |                                                               |                                            |
| `controllerService.nodeSelector`                      | Controllers node selector                                     | `kubernetes.io/os: linux`             |
| `nodeService.podSecurityContext`                      | Security context for controller pods                          | `{}`                                       |
|                                                       |                                                               |                                            |
| `nodeService.fsxPlugin.image.repository`              | aws-fsx-csi-driver image name                                 | `amazon/aws-fsx-csi-driver`                |
| `nodeService.fsxPlugin.image.tag`                     | aws-fsx-csi-driver image tag                                  | `latest`                                   |
| `nodeService.fsxPlugin.image.pullPolicy`              | aws-fsx-csi-driver image pull policy                          | `IfNotPresent`                             |
| `nodeService.fsxPlugin.extraArgs`                     | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--logtostderr --v=5`                      |
| `nodeService.fsxPlugin.securityContext`               | Security context for the container                            | `privileged: true`                         |
| `nodeService.fsxPlugin.resources`                     | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                       |                                                               |                                            |
| `nodeService.csiDriverRegistrar.image.repository`     | csi-node-driver-registrar image name                          | `quay.io/k8scsi/csi-node-driver-registrar` |
| `nodeService.csiDriverRegistrar.image.tag`            | csi-node-driver-registrar image tag                           | `v1.1.0`                                   |
| `nodeService.csiDriverRegistrar.image.pullPolicy`     | csi-node-driver-registrar image pull policy                   | `IfNotPresent`                             |
| `nodeService.csiDriverRegistrar.extraArgs`            | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--v=5`                                    |
| `nodeService.csiDriverRegistrar.securityContext`      | Security context for the container                            | `{}`                                       |
| `nodeService.csiDriverRegistrar.resources`            | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                       |                                                               |                                            |
| `nodeService.livenessProbe.image.repository`          | livenessprobe image name                                      | `quay.io/k8scsi/livenessprobe`             |
| `nodeService.livenessProbe.image.tag`                 | livenessprobe image tag                                       | `v1.1.0`                                   |
| `nodeService.livenessProbe.image.pullPolicy`          | livenessprobe image pull policy                               | `Always`                                   |
| `nodeService.livenessProbe.resources`                 | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                       |                                                               |                                            |
| `nameOverride`                                        | String to partially override aws-fsx-csi-driver.fullname      | `""`                                       |
| `fullnameOverride`                                    | String to fully override aws-fsx-csi-driver.fullname          | `""`                                       |
| `serviceAccount.create`                               | Specifies whether a service account should be created         | `true`                                     |
| `serviceAccount.annotations`                          | Additional Service Account annotations                        | `{}`                                       |
| `serviceAccount.name`                                 | Service Account name                                          | `fsx-csi-controller-sa`                    |
