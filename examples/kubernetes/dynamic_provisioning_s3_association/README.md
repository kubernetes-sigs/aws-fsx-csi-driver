## Dynamic Provisioning with Data Repository Associations
This example shows how to create a FSx for Lustre filesystem using persistence volume claim (PVC) with data repository associations integration.

Please note that data repository associations are supported on FSx for Lustre 2.12 and 2.15 file systems, excluding scratch_1 deployment type.

This integration means that you can seamlessly access the objects stored in your Amazon S3 buckets from applications mounting your Amazon FSx for Lustre file system. Please check [Using Data Repositories](https://docs.aws.amazon.com/fsx/latest/LustreGuide/fsx-data-repositories.html) for details.

### Edit [StorageClass](./specs/storageclass.yaml)
```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc
provisioner: fsx.csi.aws.com
parameters:
  subnetId: subnet-056da83524edbe641
  securityGroupIds: sg-086f61ea73388fb6b
  deploymentType: PERSISTENT_2
  perUnitStorageThroughput: "125"
  dataRepositoryAssociations: |
    - batchImportMetaDataOnCreate: true
      dataRepositoryPath: s3://ml-training-data-000
      fileSystemPath: /ml-training-data-000
      s3:
        autoExportPolicy:
          events: ["NEW", "CHANGED", "DELETED" ]
        autoImportPolicy:
          events: ["NEW", "CHANGED", "DELETED" ]
    - batchImportMetaDataOnCreate: true
      dataRepositoryPath: s3://ml-training-data-000
      fileSystemPath: /ml-training-data-000
      s3:
        autoExportPolicy:
          events: ["NEW", "CHANGED", "DELETED" ]
        autoImportPolicy:
          events: ["NEW", "CHANGED", "DELETED" ]
```
* subnetId - the subnet ID that the FSx for Lustre filesystem should be created inside.
* securityGroupIds - a common separated list of security group IDs that should be attached to the filesystem.
* dataRepositoryAssociations - a list of data repository association configurations in yaml to associate with the filesystem. See [./specs/storageclass.yaml](./specs/storageclass.yaml) for details.
* deploymentType (Optional) - FSx for Lustre supports four deployment types, SCRATCH_1, SCRATCH_2, PERSISTENT_1 and PERSISTENT_2. Default: SCRATCH_1. However, data repository association can't be used with SCRATCH_1 deploymentType
* kmsKeyId (Optional) - for deployment types PERSISTENT_1 and PERSISTENT_2, customer can specify a KMS key to use.
* perUnitStorageThroughput (Optional) - for deployment type PERSISTENT_1 and PERSISTENT_2, customer can specify the storage throughput. Default: "200". Note that customer has to specify as a string here like "200" or "100" etc. For PERSISTENT_2 SSD storage, valid values are 125, 250, 500, 1000.
* storageType (Optional) - for deployment type PERSISTENT_1, customer can specify the storage type, either SSD or HDD. Default: "SSD". For PERSISTENT_2 SSD storage, only "SSD" is allowed.
* driveCacheType (Required if storageType is "HDD") - for HDD PERSISTENT_1, specify the type of drive cache, either NONE or READ.
* dataCompressionType (Optional) - FSx for Lustre supports data compression via LZ4 algorithm. Compression is disabled when the value is set to NONE. The default value is NONE

Note:
- `dataRepositoryAssociations` can not be used with `s3ImportPath, s3ExportPath, autoImportPolicy` described in [Dynamic Provisioning with Data Repository](../dynamic_provisioning_s3/)

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
Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up to 1200 GiB, 2400 GiB, or a multiple of 3600 GiB for SSD. If the storageType is specified as HDD, the storage capacity will be rounded up to 6000 GiB or a multiple of 6000 GiB if the perUnitStorageThroughput is 12, or rounded up to 1800 or a multiple of 1800 if the perUnitStorageThroughput is 40.

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning_s3/specs/pod.yaml
```

### Use Case 1: Access S3 files from Lustre filesystem
If you only want to import data and read it without any modification and creation.

You can see S3 files are visible in the persistent volume.

```
>> kubectl exec -it fsx-app ls /data
```

### Use case 2: Export changes to S3
For new files and modified/deleted files, these are automatically export to the linked bucket.

Pod `fsx-app` create a file `out.txt` in mounted volume, then, you can see the file in S3 bucket:

```sh
>> kubectl exec -ti fsx-app -- sh -c 'echo "test" > /data/out.txt'
```
