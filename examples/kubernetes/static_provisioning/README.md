## Static Provisioning Example

This example shows how to make a pre-created FSx for Lustre filesystem mounted inside container.

### Edit [Persistent Volume Spec](./specs/pv.yaml)

```yaml
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
    # mount root of filesystem
    volumeHandle: [FileSystemId]
    # mount subdir of filesystem
    # volumeHandle: [DNSName:MountName:basedir:subdir:uuid]
    volumeAttributes:
      dnsname: [DNSName]
      mountname: [MountName]
      # set the basedir and subdir if you wanted to mount subdir
      # basedir: base
      # subdir: sub
```

Replace `volumeHandle` with `FileSystemId`, `dnsname` with `DNSName` and `mountname` with `MountName`. You can get both `FileSystemId`, `DNSName` and `MountName` using AWS CLI:

```sh
>> aws fsx describe-file-systems
```

If you wanted to mount subdir of file system, you can do by setting `basedir` and `subdir` to volumeAttributes.
See follow section for detail.

### volumeAttributes and mounted source

The table below shows the relationship between volumeAttributes and mounted source.

Notice: You have to create subdir on file system before creating PV.

| dnsname    | mountname | basedir | subdir | mounted source               |
| :--------- | :-------- | :------ | :----- | :--------------------------- |
| fs-xxx.com |           |         |        | fs-xxx.com@tcp:/fsx          |
| fs-xxx.com |           | base    | sub    | fs-xxx.com@tcp:/fsx/base/sub |
| fs-xxx.com | abc       |         |        | fs-xxx.com@tcp:/abc          |
| fs-xxx.com | abc       | base    |        | fs-xxx.com@tcp:/abc/base     |
| fs-xxx.com | abc       | base    | sub    | fs-xxx.com@tcp:/abc/base/sub |
| fs-xxx.com | abc       |         | sub    | fs-xxx.com@tcp:/abc/sub      |

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
