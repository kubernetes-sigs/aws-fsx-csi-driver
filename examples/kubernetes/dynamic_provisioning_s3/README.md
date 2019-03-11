## Dynamic Provisioning with data repository Example
This example shows how to create a FSx for Lustre filesystem using persistence volume claim (PVC) with data repository integration.

Amazon FSx for Lustre is deeply integrated with Amazon S3. This integration means that you can seamlessly access the objects stored in your Amazon S3 buckets from applications mounting your Amazon FSx for Lustre file system. Please check [Using Data Repositories](https://docs.aws.amazon.com/fsx/latest/LustreGuide/fsx-data-repositories.html) for details.

### Edit [StorageClass](./specs/storageclass.yaml)
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc
provisioner: fsx.csi.aws.com
parameters:
  subnetId: subnet-056da83524edbe641
  securityGroupIds: sg-086f61ea73388fb6b
  s3ImportPath: s3://dl-benchmark-dataset
  s3ExportPath: s3://dl-benchmark-dataset/export
```
* subnetId - the subnet ID that the FSx for Lustre filesystem should be created inside.
* securityGroupIds - a comman separated list of security group IDs that should be attached to the filesystem
* s3ImportPath(Optional) - S3 data repository you want to copy from S3 to persistent volume
* s3ExportPath(Optional) - S3 data repository you want to export new or modified files from persistent volume to S3

Note:
- S3Bucket in s3ImportPath and s3ExportPath must be same, otherwise aws-fsx-csi-driver can not create FSx for lustre successfully.
- S3ExportPath can not be given without specifying S3ImportPath.
- S3ImportPath can stand alone and a random path will be created automatically like `s3://dl-benchmark-result/FSxLustre20190308T012310Z`

### Edit [Persistent Volume Claim Spec](./specs/claim.yaml)
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fsx-claim
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: fsx-sc
  resources:
    requests:
      storage: 5Gi
```
Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up multiplication of 3600GiB.

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/pod.yaml
```

### Use Case 1: Acccess S3 files in readonly mode, no write back.
If you only want to import data and read it with any modification and creation. You can skip `s3ExportPath` parameter in your `storageclass.yaml` configuration.

You can see S3 files have been downloaded in the persistent volume.

```
kubectl exec -it fsx-app ls /data
```

### Use case 2: Sync back new created files to s3ExportPath

Pod `fsx-app` create a file `out.txt` in mounted volume, you can run following command to check this file.

```
kubectl exec -ti fsx-app -- tail -f /data/out.txt
```

### Use case 3: Sync back modified S3 files to s3ExportPath
Making change to S3 files is allowed but original files in S3 won't be updated. Instead, FSx allows you to sync these changes back to s3ExportPath.

### Sync files
New created and modified files won't be synced back to S3 automatically. In order to sync files to `S3ExportPath`, you need to install lustre client in your container image and manually run following command to force sync up.

```
sudo lfs hsm_archive /data/out.txt
sudo lfs hsm_run /data/out.txt
```
