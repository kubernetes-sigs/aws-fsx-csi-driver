
* This chart was tested on Helm v3 only
* If using k8s >1.17.x namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.
## Install chart
```shell script
helm install . --name aws-fsx-csi-driver
```

## Upgrade release
```shell script
helm upgrade aws-fsx-csi-driver \
    --install . \
    --version 0.1.0 \
    --namespace kube-system \
    -f values.yaml
```
