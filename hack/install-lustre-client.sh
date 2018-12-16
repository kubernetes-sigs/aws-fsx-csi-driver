#!/bin/sh

set -euo pipefail

curl -L https://s3-us-west-2.amazonaws.com/lustre-raghu/lustre-client-2.10.52_58_g0fedb01_dirty-1.amzn1.x86_64.rpm -o lustre-client.rpm
rpm2cpio lustre-client.rpm | cpio -idmv
rm lustre-client.rpm
