[![Build Status](https://travis-ci.org/aws/aws-fsx-csi-driver.svg?branch=master)](https://travis-ci.org/aws/aws-fsx-csi-driver)

**WARNING**: This driver is in ALPHA currently. This means that there may potentially be backwards compatible breaking changes moving forward. Do NOT use this driver in a production environment in its current state.

**DISCLAIMER**: This is not an officially supported Amazon product

## Amazon FSx for Lustre CSI Driver
### Overview

The [Amazon FSx for Lustre](https://aws.amazon.com/fsx/lustre/) Container Storage Interface (CSI) Driver provides a [CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md) interface used by container orchestrators to manage the lifecycle of Amazon FSx for lustre volumes.

This driver is in alpha stage. Basic volume operations that are functional include NodePublishVolume/NodeUnpublishVolume.

### CSI Specification Compability Matrix
| AWS FSx for Lustre CSI Driver \ CSI Version       | v0.3.0| v1.0.0 |
|---------------------------------------------------|-------|--------|
| master branch                                     | yes   | no     |

### Kubernetes Version Compability Matrix
| AWS FSx for Lustre CSI Driver \ Kubernetes Version| v1.12 | v1.13 |
|---------------------------------------------------|-------|-------|
| master branch                                     | yes   | yes   |

## Features
Currently only static provisioning is supported. With static provisioning, a FSx for lustre file system needs to be created manually first, then it could be mounted inside container as a persistence volume (PV) using AWS FSx for Lustre CSI Driver. 

## Examples
This example shows how to make a FSx for Lustre filesystem mounted inside container. Before this, get yourself familiar with how to setup kubernetes on AWS and [create FSx for Lustre filesystem](https://docs.aws.amazon.com/fsx/latest/LustreGuide/getting-started.html#getting-started-step1). And when creating FSx for Lustre file system, make sure it is created inside the same VPC as kuberentes cluster or it is accessible through VPC peering.

Once kubernetes cluster and FSx for lustre file system is created, modify secret manifest file using [secret.yaml](../deploy/kubernetes/secret.yaml). 

Then create the secret object:
```
kubectl apply -f deploy/kubernetes/secret.yaml 
```

Deploy AWS FSx for lustre CSI driver:

```
kubectl apply -f https://raw.githubusercontent.com/aws/aws-fsx-csi-driver/master/deploy/kubernetes/attacher.yaml 
kubectl apply -f https://raw.githubusercontent.com/aws/aws-fsx-csi-driver/master/deploy/kubernetes/node.yaml
```

Edit the [persistence volume manifest file](../deploy/kubernetes/sample_app/pv.yaml):
```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: fsx-pv
spec:
  capacity:
    storage: 5Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: fsx-sc
  csi:
    driver: fsx.csi.aws.com
    volumeHandle: [FileSystemId]
    volumeAttributes:
      dnsname: [DNSName] 
```
Replace `volumeHandle` with `FileSystemId` and `dnsname` with `DNSName`. You can get both `FileSystemId` and `DNSName` using AWS CLI:

```
aws fsx describe-file-systems
```

Then create PV, persistence volume claim (PVC) and storage class:
```
kubectl apply -f deploy/kubernetes/sample_app/storageclass.yaml
kubectl apply -f deploy/kubernetes/sample_app/pv.yaml
kubectl apply -f deploy/kubernetes/sample_app/claim.yaml
kubectl apply -f deploy/kubernetes/sample_app/pod.yaml
```

After the objects are created, verify that pod name app is running:

```
kubectl get pods
```

Also verify that data is written onto FSx for luster:

```
kubectl exec -ti app tail -f /data/out.txt
```

## Development
Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.11.2+

### Testing
To execute all unit tests, run: `make test`

## License
This library is licensed under the Apache 2.0 License. 
