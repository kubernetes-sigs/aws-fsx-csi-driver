## Dynamic Provisioning Example
This example shows how to use an existing FSx for Lustre filesystem using persistence volume claim (PVC) and consumes it from a pod.
Each PVC will create a separate folder at the root of the filesystem.
Update the parameters to match your filesystem.

### Edit [StorageClass](./specs/storageclass.yaml)
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc-0d3a
provisioner: fsx.csi.aws.com
parameters:
  fileSystemId: fs-0d3a9c1160a2ef4ad
mountOptions:
  - flock
```

### Edit [Persistent Volume Claim Spec](./specs/claim.yaml)
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fsx-claim
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: fsx-sc-03da
  resources:
    requests:
      storage: 6000Gi
```
NOTE: `spec.resource.requests.storage` is not used at this time. It must be managed outside of kubernetes.

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_existing_fs/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_existing_fs/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_existing_fs/specs/pod.yaml
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
