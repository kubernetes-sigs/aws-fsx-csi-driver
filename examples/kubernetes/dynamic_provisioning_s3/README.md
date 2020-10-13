## Dynamic Provisioning with Data Repository
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
  autoImportPolicy: NONE
  s3ImportPath: s3://ml-training-data-000
  s3ExportPath: s3://ml-training-data-000/export
  deploymentType: SCRATCH_2
```
* subnetId - the subnet ID that the FSx for Lustre filesystem should be created inside.
* securityGroupIds - a common separated list of security group IDs that should be attached to the filesystem.
* autoImportPolicy - the policy FSx will follow that determines how the filesystem is automatically updated with changes made in the linked data repository. For a list of acceptable policies, please view the official FSx for Lustre documentation: https://docs.aws.amazon.com/fsx/latest/APIReference/API_CreateFileSystemLustreConfiguration.html
* s3ImportPath(Optional) - S3 data repository you want to copy from S3 to persistent volume.
* s3ExportPath(Optional) - S3 data repository you want to export new or modified files from persistent volume to S3.
* deploymentType (Optional) - FSx for Lustre supports three deployment types, SCRATCH_1, SCRATCH_2 and PERSISTENT_1. Default: SCRATCH_1.
* kmsKeyId (Optional) - for deployment type PERSISTENT_1, customer can specify a KMS key to use.
* perUnitStorageThroughput (Optional) - for deployment type PERSISTENT_1, customer can specify the storage throughput. Default: "200". Note that customer has to specify as a string here like "200" or "100" etc.

Note:
- S3 Bucket in s3ImportPath and s3ExportPath must be same, otherwise the driver can not create FSx for lustre successfully.
- s3ImportPath can stand alone and a random path will be created automatically like `s3://ml-training-data-000/FSxLustre20190308T012310Z`.
- s3ExportPath can not be given without specifying S3ImportPath.
- autoImportPolicy can not be given without specifying S3ImportPath.

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
      storage: 1200Gi
```
Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up to 1200 GiB, 2400 GiB, or a multiple of 3600 GiB.

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/pod.yaml
```

### Use Case 1: Acccess S3 files from Lustre filesystem
If you only want to import data and read it without any modification and creation. You can skip `s3ExportPath` parameter in your `storageclass.yaml` configuration.

You can see S3 files are visible in the persistent volume.

```
>> kubectl exec -it fsx-app ls /data
```

### Use case 2: Archive files to s3ExportPath
For new files and modified files, you can use lustre user space tool to archive the data back to s3 on `s3ExportPath`.

Pod `fsx-app` create a file `out.txt` in mounted volume, run following command to check this file:

```sh
>> kubectl exec -ti fsx-app -- tail -f /data/out.txt
```

Export the file back to S3 using:
```sh
>> kubectl exec -ti fsx-app -- lfs hsm_archive /data/out.txt
```

## Notes
* New created files won't be synced back to S3 automatically. In order to sync files to `s3ExportPath`, you need to install lustre client in your container image and manually run following command to force sync up using `lfs hsm_archive`. And the container should run in priviledged mode with `CAP_SYS_ADMIN` capability.
* This example uses lifecycle hook to install lustre client for demostration purpose, a normal approach will be building a container image with lustre client.
