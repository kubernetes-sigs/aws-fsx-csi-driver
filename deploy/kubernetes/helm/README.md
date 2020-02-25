
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

| Parameter                                      | Description                                                   | Default                                    |
| ---------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------ |
| `controller.replicaCount`                      | Num of replicas for controller                                | `2`                                        |
| `controller.nodeSelector`                      | Controllers node selector                                     | `beta.kubernetes.io/os: linux`             |
| `controller.podSecurityContext`                | Security context for controller pods                          | `{}`                                       |
|                                                |                                                               |                                            |
| `controller.fsxPlugin.image.repository`        | aws-fsx-csi-driver image name                                 | `amazon/aws-fsx-csi-driver`                |
| `controller.fsxPlugin.image.tag`               | aws-fsx-csi-driver image tag                                  | `latest`                                   |
| `controller.fsxPlugin.image.pullPolicy`        | aws-fsx-csi-driver image pull policy                          | `IfNotPresent`                             |
| `controller.fsxPlugin.extraArgs`               | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--logtostderr --v=5`                      |
| `controller.fsxPlugin.securityContext`         | Security context for the container                            | `{}`                                       |
| `controller.fsxPlugin.resources`               | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                |                                                               |                                            |
| `controller.csiProvisioner.image.repository`   | csi-provisioner image name                                    | `quay.io/k8scsi/csi-provisioner`           |
| `controller.csiProvisioner.image.tag`          | csi-provisioner image tag                                     | `v1.3.0`                                   |
| `controller.csiProvisioner.image.pullPolicy`   | csi-provisioner image pull policy                             | `IfNotPresent`                             |
| `controller.csiProvisioner.extraArgs`          | Extra arguments to be passed to csi-provisioner               | `--timeout=5m --v=5 --enable-leader-election --leader-election-type=leases`|
| `controller.csiProvisioner.securityContext`    | Security context for the container                            | `{}`                                       |
| `controller.csiProvisioner.resources`          | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                |                                                               |                                            |
| `daemonset.nodeSelector`                       | Controllers node selector                                     | `beta.kubernetes.io/os: linux`             |
| `daemonset.podSecurityContext`                 | Security context for controller pods                          | `{}`                                       |
|                                                |                                                               |                                            |
| `daemonset.fsxPlugin.image.repository`         | aws-fsx-csi-driver image name                                 | `amazon/aws-fsx-csi-driver`                |
| `daemonset.fsxPlugin.image.tag`                | aws-fsx-csi-driver image tag                                  | `latest`                                   |
| `daemonset.fsxPlugin.image.pullPolicy`         | aws-fsx-csi-driver image pull policy                          | `IfNotPresent`                             |
| `daemonset.fsxPlugin.extraArgs`                | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--logtostderr --v=5`                      |
| `daemonset.fsxPlugin.securityContext`          | Security context for the container                            | `privileged: true`                         |
| `daemonset.fsxPlugin.resources`                | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                |                                                               |                                            |
| `daemonset.csiDriverRegistrar.image.repository`| csi-node-driver-registrar image name                          | `quay.io/k8scsi/csi-node-driver-registrar` |
| `daemonset.csiDriverRegistrar.image.tag`       | csi-node-driver-registrar image tag                           | `v1.1.0`                                   |
| `daemonset.csiDriverRegistrar.image.pullPolicy`| csi-node-driver-registrar image pull policy                   | `IfNotPresent`                             |
| `daemonset.csiDriverRegistrar.extraArgs`       | Extra arguments to be passed to aws-fsx-csi-driver fsxPlugin  | `--v=5`                                    |
| `daemonset.csiDriverRegistrar.securityContext` | Security context for the container                            | `{}`                                       |
| `daemonset.csiDriverRegistrar.resources`       | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                |                                                               |                                            |
| `daemonset.livenessProbe.image.repository`     | livenessprobe image name                                      | `quay.io/k8scsi/livenessprobe`             |
| `daemonset.livenessProbe.image.tag`            | livenessprobe image tag                                       | `v1.1.0`                                   |
| `daemonset.livenessProbe.image.pullPolicy`     | livenessprobe image pull policy                               | `Always`                                   |
| `daemonset.livenessProbe.resources`            | CPU/Memory resource requests/limits                           | `{}`                                       |
|                                                |                                                               |                                            |
| `nameOverride`                                 | String to partially override aws-fsx-csi-driver.fullname      | `""`                                       |
| `fullnameOverride`                             | String to fully override aws-fsx-csi-driver.fullname          | `""`                                       |
| `serviceAccount.create`                        | Specifies whether a service account should be created         | `true`                                     |
| `serviceAccount.annotations`                   | Additional Service Account annotations                        | `{}`                                       |
| `serviceAccount.name`                          | Service Account name                                          | `fsx-csi-controller-sa`                    |
