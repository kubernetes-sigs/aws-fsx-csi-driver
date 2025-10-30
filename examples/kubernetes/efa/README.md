# EFA Example

This example demonstrates how to provision and configure an FSx Lustre file system with EFA (Elastic Fabric Adapter) enabled for high-performance networking, and tune Lustre client parameters after mount.

## Overview

The example provisions an FSx Lustre file system with EFA networking enabled (`efaEnabled: "true"`) for enhanced performance, then uses an init container to optimize Lustre client settings.

## Key Components

- **StorageClass**: Configures FSx Lustre with EFA enabled and high throughput settings
- **PersistentVolumeClaim**: Requests 4800Gi of storage using the EFA-enabled storage class  
- **Init Container**: Tunes Lustre client parameters for optimal performance
- **Application Pod**: Mounts the high-performance FSx volume at `/data`

