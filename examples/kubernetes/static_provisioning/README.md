## Static Provisioning Example
This example shows how to make a pre-created FSx for Lustre filesystem mounted inside container. 

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
```
Replace `volumeHandle` with `FileSystemId` and `dnsname` with `DNSName`. You can get both `FileSystemId` and `DNSName` using AWS CLI:

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
