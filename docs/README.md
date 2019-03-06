[![Build Status](https://travis-ci.org/aws/csi-driver-amazon-fsx.svg?branch=master)](https://travis-ci.org/aws/csi-driver-amazon-fsx)

**WARNING**: This driver is in pre ALPHA currently. This means that there may potentially be backwards compatible breaking changes moving forward. Do NOT use this driver in a production environment in its current state.

**DISCLAIMER**: This is not an officially supported Amazon product

## Amazon FSx for Lustre CSI Driver
### Overview

The [Amazon FSx for Lustre](https://aws.amazon.com/fsx/lustre/) Container Storage Interface (CSI) Driver implements [CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md) specification for chntainer orchestrators (CO) to manage lifecycle of Amazon FSx for Lustre filesystems.

### CSI Specification Compability Matrix
| AWS FSx for Lustre CSI Driver \ CSI Version       | v0.3.0| v1.0.0 |
|---------------------------------------------------|-------|--------|
| master branch                                     | yes   | no     |

### Features
The following CSI interfaces are implemented:
* Controller Service: CreateVolume, DeleteVolume, ControllerGetCapabilities, ValidateVolumeCapabilities
* Node Service: NodePublishVolume, NodeUnpublishVolume, NodeGetCapabilities, NodeGetInfo, NodeGetId
* Identity Service: GetPluginInfo, GetPluginCapabilities, Probe

## FSx for Lustre CSI Driver on Kubernetes
Following sections are Kubernetes specific. If you are Kubernetes user, use followings for driver features, installation steps and examples.

### Kubernetes Version Compability Matrix
| AWS FSx for Lustre CSI Driver \ Kubernetes Version| v1.11 | v1.12 | v1.13 |
|---------------------------------------------------|-------|-------|-------|
| master branch                                     | yes   | yes   | yes   |

### Features
* Static provisioning - FSx for Lustre file system needs to be created manually first, then it could be mounted inside container as a volume using the Driver.
* Dynamic provisioning - uses persistence volume claim (PVC) to let the Kuberenetes to create the FSx for Lustre filesystem for you and consumes the volume from inside container.

### Installation
Checkout the project:
```sh
>> git clone https://github.com/aws/aws-fsx-csi-driver.git
>> cd aws-fsx-csi-driver
```

Edit the [secret manifest](../deploy/kubernetes/secret.yaml) using your favorite text editor. The secret should have enough permission to create FSx for Lustre filesystem. Then deploy the secret:

```sh
>> kubectl apply -f deploy/kubernetes/secret.yaml
```

Then deploy the driver:

```sh
>> kubectl apply -f deploy/kubernetes/controller.yaml
>> kubectl apply -f deploy/kubernetes/node.yaml
```

### Examples
Before the example, you need to:
* Get yourself familiar with how to setup Kubernetes on AWS and [create FSx for Lustre filesystem](https://docs.aws.amazon.com/fsx/latest/LustreGuide/getting-started.html#getting-started-step1) if you are using static provisioning.
* When creating FSx for Lustre file system, make sure it is accessible from Kuberenetes cluster. This can be achieved by creating FSx for lustre filesystem inside the same VPC as Kubernetes cluster or using VPC peering.
* Install FSx for Lustre CSI driver following the [Installation](README.md#Installation) steps.

#### Example links
* [Static provisioning](../examples/kubernetes/static_provisioning/README.md)
* [Dynamic provisioning](../examples/kubernetes/dynamic_provisioning/README.md)
* [Accessing the filesystem from multiple pods](../examples/kubernetes/multiple_pods/README.md)

## Development
Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.11.4+

### Dependency
Dependencies are managed through go module. To build the project, first turn on go mod using `export GO111MODULE=on`, to build the project run: `make`

### Testing
* To execute all unit tests, run: `make test`
* To execute sanity tests, run: `make test-sanity`

## License
This library is licensed under the Apache 2.0 License. 
