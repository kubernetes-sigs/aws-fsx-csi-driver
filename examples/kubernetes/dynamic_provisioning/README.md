## Dynamic Provisioning Example
This example shows how to create a FSx for Lustre filesystem using persistence volume claim (PVC) and consumes it from a pod. 


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
  deploymentType: PERSISTENT_1
  storageType: HDD
```
* subnetId - the subnet ID that the FSx for Lustre filesystem should be created inside.
* securityGroupIds - a comma separated list of security group IDs that should be attached to the filesystem
* deploymentType (Optional) - FSx for Lustre supports four deployment types, SCRATCH_1, SCRATCH_2, PERSISTENT_1 and PERSISTENT_2. Default: SCRATCH_1.
* kmsKeyId (Optional) - for deployment types PERSISTENT_1 and PERSISTENT_2, customer can specify a KMS key to use.
* perUnitStorageThroughput (Optional) - for deployment type PERSISTENT_1 and PERSISTENT_2, customer can specify the storage throughput. Default: "200". Note that customer has to specify as a string here like "200" or "100" etc.
* storageType (Optional) - for deployment type PERSISTENT_1, customer can specify the storage type, either SSD or HDD. Default: "SSD"
* driveCacheType (Required if storageType is "HDD") - for HDD PERSISTENT_1, specify the type of drive cache, either NONE or READ.
* efaEnabled (Optional) - A boolean flag indicating whether Elastic Fabric Adapter (EFA) support is enabled for a PERSISTENT_2 file system with metadata configuration.
* metadataConfigurationMode (Optional) - The metadata configuration mode for provisioning Metadata IOPS for a PERSISTENT_2 file system, either "AUTOMATIC" or "USER_PROVISIONED".
* metadataIops (Required if metadataConfigurationMode is USER_PROVISIONED) - Specifies the number of Metadata IOPS to provision for the file system.
* automaticBackupRetentionDays (Optional) - The number of days to retain automatic backups. The default is to retain backups for 7 days. Setting this value to 0 disables the creation of automatic backups. The maximum retention period for backups is 35 days
* dailyAutomaticBackupStartTime (Optional) - The preferred time to take daily automatic backups, formatted HH:MM in the UTC time zone.
* copyTagsToBackups (Optional) - A boolean flag indicating whether tags for the file system should be copied to backups. This value defaults to false. If it's set to true, all tags for the file system are copied to all automatic and user-initiated backups where the user doesn't specify tags. If this value is true, and you specify one or more tags, only the specified tags are copied to backups. If you specify one or more tags when creating a user-initiated backup, no tags are copied from the file system, regardless of this value.
* dataCompressionType (Optional) - FSx for Lustre supports data compression via LZ4 algorithm. Compression is disabled when the value is set to NONE. The default value is NONE 
* weeklyMaintenanceStartTime (Optional) - The preferred start time to perform weekly maintenance, formatted d:HH:MM in the UTC time zone, where d is the weekday number, from 1 through 7, beginning with Monday and ending with Sunday. The default value is "7:09:00" (Sunday 09:00 UTC)
* fileSystemTypeVersion (Optional) - Sets the Lustre version of the Amazon FSx for Lustre file system to be created. Valid values are 2.10 and 2.12. The default value is "2.10"
* extraTags (Optional) - Tags that will be set on the FSx resource created in AWS, in the form of a comma separated list with each tag delimited by an equals sign (example - "Tag1=Value1,Tag2=Value2") . Default is a single tag with CSIVolumeName as the key and the generated volume name as it's value.

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
      storage: 6000Gi
```
Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up to 1200 GiB, 2400 GiB, or a multiple of 3600 GiB for SSD. If the storageType is specified as HDD, the storage capacity will be rounded up to 6000 GiB or a multiple of 6000 GiB if the perUnitStorageThroughput is 12, or rounded up to 1800 or a multiple of 1800 if the perUnitStorageThroughput is 40.

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/pod.yaml
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

### Notes for EFA enabled filesystems
* See [EKS userguide](https://docs.aws.amazon.com/eks/latest/userguide/node-efa.html) for creating an EFA supported cluster
* To configure EFA interfaces on EKS client nodes, consider adding the setup from [Configuring EFA clients](https://docs.aws.amazon.com/fsx/latest/LustreGuide/configure-efa-clients.html) to the `preBootstrapCommands` property of the nodegroup