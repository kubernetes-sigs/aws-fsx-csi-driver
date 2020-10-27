## Static Provisioning Example
This example shows how to make a pre-created FSx for Lustre filesystem mounted inside container.
It uses the optional subpath parameter to map a sub-folder of the FSx for Lustre filesystem into the cluster, allowing one filesystem to be used independently by many volumes.

### Edit [Persistent Volume Spec](./specs/pv.yaml)
```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: fsx-pv
spec:
  capacity:
    storage: 1200Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  mountOptions:
    - flock
  persistentVolumeReclaimPolicy: Recycle
  csi:
    driver: fsx.csi.aws.com
    volumeHandle: [FileSystemId]
    volumeAttributes:
      dnsname: [DNSName]
      mountname: [MountName]
      subpath: someDirName
```
Replace `volumeHandle` with `FileSystemId`, `dnsname` with `DNSName` and `mountname` with `MountName`. You can get both `FileSystemId`, `DNSName` and `MountName` using AWS CLI:
When the optional subpath parameter is set, this will create a folder with the name from subpath in the root directory of the FSx filesystem.

```sh
>> aws fsx describe-file-systems
```

### Deploy the Application
Create PV, persistent volume claim (PVC), and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/pv.yaml
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/pod.yaml
```

### Check the Application uses FSx for Lustre filesystem
After the objects are created, verify that pod is running:

```sh
>> kubectl get pods
```

Also verify that data is written onto FSx for Luster filesystem:

```sh
>> kubectl exec -ti fsx-app -- tail -f /data/out.txt
```
