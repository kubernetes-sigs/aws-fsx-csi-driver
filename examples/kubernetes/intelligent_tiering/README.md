# INTELLIGENT_TIERING Dynamic Provisioning Example

This example shows how to create an FSx for Lustre filesystem with INTELLIGENT_TIERING storage type using a PersistentVolumeClaim (PVC) and consume it from a pod.

## Overview

INTELLIGENT_TIERING is a storage type where AWS automatically manages storage capacity. This eliminates the need to specify storage capacity in your PVC, as the filesystem scales automatically based on your data.

### Key Features

- **Automatic Capacity Management**: AWS manages storage capacity automatically
- **Configurable Metadata IOPS**: Choose between 6000 or 12000 IOPS for metadata operations
- **Flexible Data Read Cache**: Configure SSD cache sizing to balance performance and cost
- **PERSISTENT_2 Deployment**: Uses the latest deployment type with enhanced features

## Prerequisites

- Kubernetes cluster with the FSx CSI Driver installed
- Subnet ID in your VPC
- Security group ID(s) for filesystem access
- IAM permissions for FSx operations (AmazonFSxFullAccess or equivalent)

## Examples

This directory contains five example configurations:

1. **Basic** - Default configuration with minimal parameters
2. **High Metadata IOPS** - Optimized for metadata-intensive workloads
3. **No Cache** - Cost-optimized configuration without SSD cache
4. **Custom Cache** - User-defined cache size for specific performance needs
5. **Skip Final Backup** - Test/development configuration that skips final backup on deletion for faster cleanup

## Basic Example

### 1. Edit [StorageClass](./specs/storageclass-basic.yaml)

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc-intelligent-tiering
provisioner: fsx.csi.aws.com
parameters:
  subnetId: subnet-0eabfaa81fb22bcaf
  securityGroupIds: sg-068000ccf82dfba88
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
```

**Required Parameters:**
- `subnetId` - The subnet ID where the filesystem should be created
- `securityGroupIds` - Comma-separated list of security group IDs
- `storageType` - Must be set to `INTELLIGENT_TIERING`
- `throughputCapacity` - Sustained throughput in MB/s (must be multiple of 4000)

**Optional Parameters:**
- `deploymentType` - Automatically set to `PERSISTENT_2` (only valid value)
- `metadataIops` - Metadata IOPS: `6000` (default) or `12000`
- `dataReadCacheSizingMode` - Cache sizing mode (see below)
- `dataReadCacheSizeGiB` - Cache size in GiB (required for USER_PROVISIONED mode)

### 2. Create [PersistentVolumeClaim](./specs/claim.yaml)

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fsx-claim-intelligent-tiering
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: fsx-sc-intelligent-tiering
  resources:
    requests:
      storage: 1200Gi
```

**Note**: The `storage` value is ignored for INTELLIGENT_TIERING. AWS manages capacity automatically. You can specify any valid value (e.g., 1200Gi) to satisfy Kubernetes requirements.

### 3. Deploy the Application

```sh
kubectl apply -f examples/kubernetes/intelligent_tiering/specs/storageclass-basic.yaml
kubectl apply -f examples/kubernetes/intelligent_tiering/specs/claim.yaml
kubectl apply -f examples/kubernetes/intelligent_tiering/specs/pod.yaml
```

### 4. Verify the Deployment

Check that the pod is running:
```sh
kubectl get pods
```

Verify data is written to the filesystem:
```sh
kubectl exec -ti fsx-app -- tail -f /data/out.txt
```

## Configuration Options

### Data Read Cache Sizing Modes

The `dataReadCacheSizingMode` parameter controls how SSD cache is provisioned:

- **`PROPORTIONAL_TO_THROUGHPUT_CAPACITY`** (default) - Cache size scales with throughput capacity
- **`NO_CACHE`** - No SSD cache (lowest cost, suitable for write-heavy workloads)
- **`USER_PROVISIONED`** - Specify exact cache size (requires `dataReadCacheSizeGiB` parameter)

### Throughput Capacity

Valid values are multiples of 4000 MB/s:
- `4000` - 4000 MB/s
- `8000` - 8000 MB/s
- `12000` - 12000 MB/s
- And higher multiples of 4000

### Metadata IOPS

Choose based on your workload's metadata operation requirements:
- `6000` - Standard metadata performance (default)
- `12000` - High metadata performance (for metadata-intensive workloads)

## Advanced Examples

### High Metadata IOPS Configuration

For workloads with intensive metadata operations (many small files, frequent directory listings):

```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "8000"
  metadataIops: "12000"
```

See [storageclass-high-metadata-iops.yaml](./specs/storageclass-high-metadata-iops.yaml)

### No Cache Configuration

For cost-optimized, write-heavy workloads that don't benefit from read caching:

```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  dataReadCacheSizingMode: NO_CACHE
```

See [storageclass-no-cache.yaml](./specs/storageclass-no-cache.yaml)

### Custom Cache Configuration

For precise control over cache size:

```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  dataReadCacheSizingMode: USER_PROVISIONED
  dataReadCacheSizeGiB: "1000"
```

See [storageclass-custom-cache.yaml](./specs/storageclass-custom-cache.yaml)

### Skip Final Backup Configuration (Test/Development)

For test and development environments where you want faster filesystem deletion:

```yaml
parameters:
  storageType: INTELLIGENT_TIERING
  throughputCapacity: "4000"
  skipFinalBackup: "true"
```

**Warning**: This skips the final backup on deletion. Only use this in test/development environments where you don't need to preserve data.

See [storageclass-skip-final-backup.yaml](./specs/storageclass-skip-final-backup.yaml)

## IAM Permissions

No additional IAM permissions are required beyond standard FSx operations. The following permissions are sufficient:

- `fsx:CreateFileSystem`
- `fsx:DescribeFileSystems`
- `fsx:DeleteFileSystem`
- `fsx:TagResource`

These are included in the `AmazonFSxFullAccess` managed policy.

## Troubleshooting

### Common Issues

1. **"throughputCapacity is required for INTELLIGENT_TIERING storage type"**
   - Ensure `throughputCapacity` parameter is specified in the StorageClass

2. **"throughputCapacity must be a multiple of 4000"**
   - Use values like 4000, 8000, 12000, etc.

3. **"dataReadCacheSizeGiB is required when dataReadCacheSizingMode is USER_PROVISIONED"**
   - Add `dataReadCacheSizeGiB` parameter when using USER_PROVISIONED mode

4. **"dataReadCacheSizeGiB must be at least 32 GiB"**
   - Increase the cache size to at least 32 GiB

## Additional Resources

- [FSx for Lustre Documentation](https://docs.aws.amazon.com/fsx/latest/LustreGuide/)
- [CSI Driver Options](../../../docs/options.md)
- [Troubleshooting Guide](../../../docs/troubleshooting.md)
