[![Build Status](https://travis-ci.org/kubernetes-sigs/aws-fsx-csi-driver.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/aws-fsx-csi-driver)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/aws-fsx-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/aws-fsx-csi-driver?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-sigs/aws-fsx-csi-driver)](https://goreportcard.com/report/github.com/kubernetes-sigs/aws-fsx-csi-driver)

## Amazon FSx for Lustre CSI Driver
### Overview

The [Amazon FSx for Lustre](https://aws.amazon.com/fsx/lustre/) Container Storage Interface (CSI) Driver implements [CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md) specification for container orchestrators (CO) to manage lifecycle of Amazon FSx for Lustre filesystems.

### CSI Specification Compability Matrix
| AWS FSx for Lustre CSI Driver \ CSI Version       | v0.3.0| v1.1.0 |
|---------------------------------------------------|-------|--------|
| master branch                                     | no    | yes    |
| v0.4.0                                            | no    | yes    |
| v0.3.0                                            | no    | yes    |
| v0.2.0                                            | no    | yes    |
| v0.1.0                                            | yes   | no     |

### Features
The following CSI interfaces are implemented:
* Controller Service: CreateVolume, DeleteVolume, ControllerGetCapabilities, ValidateVolumeCapabilities
* Node Service: NodePublishVolume, NodeUnpublishVolume, NodeGetCapabilities, NodeGetInfo, NodeGetId
* Identity Service: GetPluginInfo, GetPluginCapabilities, Probe

## FSx for Lustre CSI Driver on Kubernetes
Following sections are Kubernetes specific. If you are Kubernetes user, use followings for driver features, installation steps and examples.

### Kubernetes Version Compability Matrix
| AWS FSx for Lustre CSI Driver \ Kubernetes Version| v1.11 | v1.12 | v1.13 | v1.14 | v1.15+ |
|---------------------------------------------------|-------|-------|-------|-------|--------|
| master branch                                     | no    | no    | no    | yes   | yes    |
| v0.4.0                                            | no    | no    | no    | yes   | yes    |
| v0.3.0                                            | no    | no    | no    | yes   | yes    |
| v0.2.0                                            | no    | no    | no    | yes   | yes    |
| v0.1.0                                            | yes   | yes   | yes   | no    | no     |

### Container Images
|FSx CSI Driver Version     | Image                               |
|---------------------------|-------------------------------------|
|master branch              |amazon/aws-fsx-csi-driver:latest     |
|v0.4.0                     |amazon/aws-fsx-csi-driver:v0.4.0     |
|v0.3.0                     |amazon/aws-fsx-csi-driver:v0.3.0     |
|v0.2.0                     |amazon/aws-fsx-csi-driver:v0.2.0     |
|v0.1.0                     |amazon/aws-fsx-csi-driver:v0.1.0     |

### Features
* Static provisioning - FSx for Lustre file system needs to be created manually first, then it could be mounted inside container as a volume using the Driver.
* Dynamic provisioning - uses persistent volume claim (PVC) to let the Kuberenetes to create the FSx for Lustre filesystem for you and consumes the volume from inside container.
* Mount options - mount options can be specified in storageclass to define how the volume should be mounted.

**Notes**:
* For dynamically provisioned volumes, only one subnet is allowed inside storageclass's `parameters.subnetId`. This is a [limitation](https://docs.aws.amazon.com/fsx/latest/APIReference/API_CreateFileSystem.html#FSx-CreateFileSystem-request-SubnetIds) that is enforced by FSx for Lustre.

### Installation
#### Set up driver permission
The driver requires IAM permission to talk to Amazon FSx for Lustre service to create/delete the filesystem on user's behalf. There are several methods to grant driver IAM permission:
* Using secret object - create an IAM user with proper permission, put that user's credentials in [secret manifest](../deploy/kubernetes/secret.yaml) then deploy the secret.

```sh
curl https://raw.githubusercontent.com/kubernetes-sigs/aws-fsx-csi-driver/master/deploy/kubernetes/secret.yaml > secret.yaml
# Edit the secret with user credentials
kubectl apply -f secret.yaml
```

* Using worker node instance profile - grant all the worker nodes with proper permission by attach policy to the instance profile of the worker.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreateServiceLinkedRole",
        "iam:AttachRolePolicy",
        "iam:PutRolePolicy"
       ],
      "Resource": "arn:aws:iam::*:role/aws-service-role/s3.data-source.lustre.fsx.amazonaws.com/*"
    },
    {
      "Action":"iam:CreateServiceLinkedRole",
      "Effect":"Allow",
      "Resource":"*",
      "Condition":{
        "StringLike":{
          "iam:AWSServiceName":[
            "fsx.amazonaws.com"
          ]
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "fsx:CreateFileSystem",
        "fsx:DeleteFileSystem",
        "fsx:DescribeFileSystems"
      ],
      "Resource": ["*"]
    }
  ]
}
```

#### Deploy driver
```sh
kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=master"
```

Alternatively, you could also install the driver using helm:
```sh
helm repo add aws-fsx-csi-driver https://kubernetes-sigs.github.io/aws-fsx-csi-driver/
helm install aws-fsx-csi-driver aws-fsx-csi-driver/aws-fsx-csi-driver
```
### Examples
Before the example, you need to:
* Get yourself familiar with how to setup Kubernetes on AWS and [create FSx for Lustre filesystem](https://docs.aws.amazon.com/fsx/latest/LustreGuide/getting-started.html#getting-started-step1) if you are using static provisioning.
* When creating FSx for Lustre file system, make sure its VPC is accessible from Kuberenetes cluster's VPC and network traffic is allowed by security group.
  * For FSx for Lustre VPC, you can either create FSx for lustre filesystem inside the same VPC as Kubernetes cluster or using VPC peering.
  * For security group, make sure port 988 is allowed for the security groups that are attached the lustre filesystem ENI.
* Install FSx for Lustre CSI driver following the [Installation](README.md#Installation) steps.

#### Example links
* [Static provisioning](../examples/kubernetes/static_provisioning/README.md)
* [Dynamic provisioning](../examples/kubernetes/dynamic_provisioning/README.md)
* [Dynamic provisioning with S3 integration](../examples/kubernetes/dynamic_provisioning_s3/README.md)
* [Accessing the filesystem from multiple pods](../examples/kubernetes/multiple_pods/README.md)

## Development
Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.13.4+

### Dependency
Dependencies are managed through go module. To build the project, first turn on go mod using `export GO111MODULE=on`, to build the project run: `make`

### Testing
* To execute all unit tests, run: `make test`
* To execute sanity tests, run: `make test-sanity`
* To execute e2e tests, run: `make test-e2e`

## License
This library is licensed under the Apache 2.0 License.
