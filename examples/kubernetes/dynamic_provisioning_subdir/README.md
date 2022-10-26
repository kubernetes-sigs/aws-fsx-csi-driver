## Dynamic Provisioning with Subdirectory mode

This example shows how to create a subdirectory on the existing FSx for Lustre file system using persistence volume claim (PVC) and consumes it from a pod.

### Edit [StorageClass](./specs/storageclass.yaml)

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-subdir-sc
provisioner: fsx.csi.aws.com
parameters:
  dnsname: [DNSName]
  mountname: [MountName]
  basedir: dynamic_provisioning
mountOptions:
  - flock
```

* dnsname - The DNS name of existing FSx for Lustre file system
* mountname - The mountname of existing FSx for Lustre file system
* basedir - Parent path for subdirectory that created by dynamic provisioning. If this parameter is empty or not set, subdirectory is created under the root path of the file system.

Replace `dnsname` with `DNSName` and `mountname` with `MountName`. You can get both `DNSName` and `MountName` using AWS CLI:

```sh
>> aws fsx describe-file-systems
```

### Edit [Persistent Volume Claim Spec](./specs/claim.yaml)

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fsx-claim
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: fsx-subdir-sc
  resources:
    requests:
      storage: 100Gi
```

`spec.resource.requests.storage` is ignored on subdir provisioning, so any value is OK.

### Deploy the Application

Create PVC, storageclass and the pod that consumes the PV:

```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/pod.yaml
```

### Check the Application uses FSx for Lustre file system

After the objects are created, verify that pod is running:

```sh
>> kubectl get pods
```

Also verify that data is written onto FSx for Luster file system:

```sh
>> kubectl exec -ti fsx-app -- tail -f /data/out.txt
```
