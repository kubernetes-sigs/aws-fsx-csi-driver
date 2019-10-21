## Tuning Lustre Max Memory Cache
This example shows how to set lustre `llite.*.max_cached_mb` using init container. Lustre client interacts with lustre kernel module for data caching at host level. Since the cache resides in kernel space, it won't be counted toward application container's memory limit. Sometimes it is desireable to reduce the lustre cache size to limit memory consumption at host level. In this example, the max cache size is set to 32MB, but other values may be selected depending on what makes sense for the workload.

### Edit [Pod](./specs/pod.yaml)
```
apiVersion: v1
kind: Pod
metadata:
  name: fsx-app
spec:
  initContainers:
  - name: set-lustre-cache
    image: amazon/aws-fsx-csi-driver:latest
    securityContext:
      privileged: true
    command: ["/sbin/lctl"]
    args: ["set_param", "llite.*.max_cached_mb=32"]
  containers:
  - name: app
    image: amazonlinux:2
    command: ["/bin/sh"]
    args: ["-c", "sleep 999999"]
    volumeMounts:
    - name: persistent-storage
      mountPath: /data
  volumes:
  - name: persistent-storage
    persistentVolumeClaim:
      claimName: fsx-claim
```
The `fsx-app` pod has an init container that sets `llite.*.max_cached_mb` using `lctl`.

## Notes
* The aws-fsx-csi-driver image is reused in the init container for the `lctl` command. You could chose your own container image for this purpose as long as the lustre client user space tools `lctl` is available inside the image.
* The init container needs to be privileged as required by `lctl`
