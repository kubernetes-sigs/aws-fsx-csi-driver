## Multiple Pods Read Write Many 
This example shows how to create a static provisioned FSx for Lustre PV and access it from multiple pods with RWX access mode.

### Edit Persistent Volume
Edit persistent volume using sample [spec](pv.yaml):
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
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: fsx-sc
  csi:
    driver: fsx.csi.aws.com
    volumeHandle: [FileSystemId]
    volumeAttributes:
      dnsname: [DNSName] 
```
Replace `volumeHandle` with `FileSystemId` and `dnsname` with `DNSName`. Note that the access mode is `RWX` which means the PV can be read and write from multiple pods.

You can get both `FileSystemId` and `DNSName` using AWS CLI:

```
aws fsx describe-file-systems
```

### Deploy the Application
Create PV, persistence volume claim (PVC), storageclass and the pods that consume the PV:
```
kubectl apply -f examples/kubernetes/static_provisioning/storageclass.yaml
kubectl apply -f examples/kubernetes/static_provisioning/pv.yaml
kubectl apply -f examples/kubernetes/static_provisioning/claim.yaml
kubectl apply -f examples/kubernetes/static_provisioning/pod1.yaml
kubectl apply -f examples/kubernetes/static_provisioning/pod2.yaml
```

Both pod1 and pod2 are writing to the same FSx for Lustre filesystem at the same time.

### Check the Application uses FSx for Lustre filesystem
After the objects are created, verify that pod is running:

```
kubectl get pods
```

Also verify that data is written onto FSx for Luster filesystem:

```
kubectl exec -ti app1 -- tail -f /data/out1.txt
kubectl exec -ti app2 -- tail -f /data/out2.txt
```
