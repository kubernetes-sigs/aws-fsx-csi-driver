# Driver Options
There are a couple of driver options that can be passed as arguments when starting the driver container.

| Option argument             | value sample                                      | default                                             | Description                                                                                 |
|-----------------------------|---------------------------------------------------|-----------------------------------------------------|---------------------------------------------------------------------------------------------|
| endpoint                    | tcp://127.0.0.1:10000/                            | unix:///var/lib/csi/sockets/pluginproxy/csi.sock    | The socket on which the driver will listen for CSI RPCs                                     |
| extra-tags                  | key1=value1,key2=value2                           |                                                     | Tags specified in the controller spec are attached to each dynamically provisioned resource |
| logging-format              | json                                              | text                                                | Sets the log format. Permitted formats: text, json                                          |

# StorageClass Parameters

StorageClass parameters are used to configure FSx for Lustre filesystems during dynamic provisioning. These parameters are specified in the `parameters` section of a StorageClass resource.

## Common Parameters

These parameters apply to all storage types:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subnetId` | string | Yes | The subnet ID where the FSx filesystem should be created |
| `securityGroupIds` | string | Yes | Comma-separated list of security group IDs to attach to the filesystem |
| `storageType` | string | No | Storage type: `SSD`, `HDD`, or `INTELLIGENT_TIERING`. Default: `SSD` |
| `deploymentType` | string | No | Deployment type: `SCRATCH_1`, `SCRATCH_2`, `PERSISTENT_1`, or `PERSISTENT_2`. Default: `SCRATCH_1` |
| `kmsKeyId` | string | No | KMS key ID for encryption (PERSISTENT_1 and PERSISTENT_2 only) |
| `automaticBackupRetentionDays` | string | No | Number of days to retain automatic backups (0-35). Default: `7` |
| `dailyAutomaticBackupStartTime` | string | No | Preferred time for daily backups in UTC (HH:MM format) |
| `copyTagsToBackups` | string | No | Copy filesystem tags to backups: `true` or `false`. Default: `false` |
| `dataCompressionType` | string | No | Data compression: `NONE` or `LZ4`. Default: `NONE` |
| `weeklyMaintenanceStartTime` | string | No | Preferred weekly maintenance window (d:HH:MM format, UTC). Default: `7:09:00` |
| `fileSystemTypeVersion` | string | No | Lustre version: `2.10` or `2.12`. Default: `2.10` |
| `extraTags` | string | No | Additional tags in format: `Tag1=Value1,Tag2=Value2` |

## PERSISTENT_1 and PERSISTENT_2 Parameters

These parameters apply to PERSISTENT_1 and PERSISTENT_2 deployment types:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `perUnitStorageThroughput` | string | No | Storage throughput per unit (MB/s/TiB): `12`, `40`, `50`, `100`, `125`, `250`, `500`, `1000`. Default: `200` |
| `driveCacheType` | string | Conditional | Drive cache type for HDD storage: `NONE` or `READ`. Required if `storageType` is `HDD` |

## PERSISTENT_2 Metadata Configuration

These parameters apply to PERSISTENT_2 deployment type for metadata configuration:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `efaEnabled` | string | No | Enable Elastic Fabric Adapter (EFA): `true` or `false`. Requires metadata configuration |
| `metadataConfigurationMode` | string | No | Metadata configuration mode: `AUTOMATIC` or `USER_PROVISIONED` |
| `metadataIops` | string | Conditional | Metadata IOPS to provision. Required if `metadataConfigurationMode` is `USER_PROVISIONED` |

## INTELLIGENT_TIERING Parameters

INTELLIGENT_TIERING is a storage type where AWS automatically manages storage capacity. It requires specific configuration parameters and only supports PERSISTENT_2 deployment type.

### Required Parameters

| Parameter | Type | Required | Valid Values | Description |
|-----------|------|----------|--------------|-------------|
| `storageType` | string | Yes | `INTELLIGENT_TIERING` | Must be set to enable INTELLIGENT_TIERING |
| `throughputCapacity` | string | Yes | Multiples of 4000 (e.g., `4000`, `8000`, `12000`) | Sustained throughput capacity in MB/s |

### Optional Parameters

| Parameter | Type | Required | Valid Values | Default | Description |
|-----------|------|----------|--------------|---------|-------------|
| `deploymentType` | string | No | `PERSISTENT_2` | `PERSISTENT_2` | Automatically set to PERSISTENT_2 (only valid value) |
| `metadataIops` | string | No | `6000`, `12000` | `6000` | Metadata IOPS for the filesystem |
| `dataReadCacheSizingMode` | string | No | `NO_CACHE`, `PROPORTIONAL_TO_THROUGHPUT_CAPACITY`, `USER_PROVISIONED` | `PROPORTIONAL_TO_THROUGHPUT_CAPACITY` | SSD cache sizing mode |
| `dataReadCacheSizeGiB` | string | Conditional | >= 32 | - | Cache size in GiB. Required when `dataReadCacheSizingMode` is `USER_PROVISIONED` |
| `skipFinalBackup` | string | No | `true`, `false` | `false` | Skip final backup on deletion. Set to `true` to speed up deletion (useful for test environments) |

### INTELLIGENT_TIERING Notes

- **Storage Capacity**: Do not specify `storageCapacity` in the PVC. AWS manages capacity automatically. Any value specified in the PVC is ignored.
- **Deployment Type**: Only `PERSISTENT_2` is supported. The driver automatically sets this if not specified.
- **Throughput Capacity**: Must be a multiple of 4000 MB/s (e.g., 4000, 8000, 12000, 16000).
- **Metadata IOPS**: Choose `6000` for standard workloads or `12000` for metadata-intensive workloads.
- **Data Read Cache Modes**:
  - `PROPORTIONAL_TO_THROUGHPUT_CAPACITY` (default): Cache size scales with throughput capacity
  - `NO_CACHE`: No SSD cache (lowest cost, suitable for write-heavy workloads)
  - `USER_PROVISIONED`: Specify exact cache size (minimum 32 GiB)
- **Skip Final Backup**: Set `skipFinalBackup: "true"` to skip the final backup on deletion, which significantly speeds up filesystem deletion. This is particularly useful for test environments. By default, AWS FSx takes a final backup before deletion.

### INTELLIGENT_TIERING Examples

**Basic Configuration (defaults):**
```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  subnetId: subnet-xxxxx
  securityGroupIds: sg-xxxxx
```

**High Metadata IOPS:**
```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "8000"
  metadataIops: "12000"
  subnetId: subnet-xxxxx
  securityGroupIds: sg-xxxxx
```

**No Cache (cost-optimized):**
```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  dataReadCacheSizingMode: NO_CACHE
  subnetId: subnet-xxxxx
  securityGroupIds: sg-xxxxx
```

**Custom Cache Size:**
```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  dataReadCacheSizingMode: USER_PROVISIONED
  dataReadCacheSizeGiB: "1000"
  subnetId: subnet-xxxxx
  securityGroupIds: sg-xxxxx
```

For complete examples, see [examples/kubernetes/intelligent_tiering/](../examples/kubernetes/intelligent_tiering/).

## IAM Permissions

The following IAM permissions are required for the FSx CSI Driver to manage filesystems:

### Basic Permissions (All Storage Types)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "fsx:CreateFileSystem",
        "fsx:DeleteFileSystem",
        "fsx:DescribeFileSystems",
        "fsx:TagResource"
      ],
      "Resource": "*"
    }
  ]
}
```

### INTELLIGENT_TIERING Permissions

No additional IAM permissions are required for INTELLIGENT_TIERING beyond the basic permissions listed above. The same permissions used for other storage types apply to INTELLIGENT_TIERING filesystems.

### Managed Policy

The AWS managed policy `AmazonFSxFullAccess` includes all necessary permissions for the FSx CSI Driver, including INTELLIGENT_TIERING support.
