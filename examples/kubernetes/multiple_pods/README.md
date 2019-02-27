## Multiple Pods Read Write Many 
This example shows how to create a dynamically provisioned FSx for Lustre PV and access it from multiple pods with `ReadWriteMany` access mode. If you are using static provisioning, following steps to setup static provisioned PV with access mode set to `ReadWriteMany` and the rest of steps of consuming the volume from pods are similar.

### Edit [StorageClass](storageclass.yaml)
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc
provisioner: fsx.csi.aws.com
parameters:
  subnetId: subnet-056da83524edbe641
  securityGroupIds: sg-086f61ea73388fb6b
```
* subnetId - the subnet ID that the FSx for Lustre filesystem should be created inside.
* securityGroupIds - a comman separated list of security group IDs that should be attached to the filesystem

### Deploy the Application
Create PV, persistence volume claim (PVC), storageclass and the pods that consume the PV:
```sh
>> kubectl apply -f examples/kubernetes/multiple_pods/storageclass.yaml
>> kubectl apply -f examples/kubernetes/multiple_pods/claim.yaml
>> kubectl apply -f examples/kubernetes/multiple_pods/pod1.yaml
>> kubectl apply -f examples/kubernetes/multiple_pods/pod2.yaml
```

Both pod1 and pod2 are writing to the same FSx for Lustre filesystem at the same time.

### Check the Application uses FSx for Lustre filesystem
After the objects are created, verify that pod is running:

```sh
>> kubectl get pods
```

Also verify that data is written onto FSx for Luster filesystem:

```sh
>> kubectl exec -ti app1 -- tail -f /data/out1.txt
>> kubectl exec -ti app2 -- tail -f /data/out2.txt
```
