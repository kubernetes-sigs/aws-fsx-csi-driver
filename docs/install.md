# Installation

## Prerequisites

* Kubernetes Version >= 1.20

* If you are using a self managed cluster, ensure the flag `--allow-privileged=true` for `kube-apiserver`.

## Installation
### Set up driver permissions
The driver controller requires IAM permissions to create, update, tag and delete FSx for Lustre file systems on the user's behalf.
Only the controller service account (`fsx-csi-controller-sa`) needs AWS credentials. The node DaemonSet makes no FSx API calls.

The EKS Pod Identity example below attaches the tag-based policy [`cluster-tag-policy.json`](iam/cluster-tag-policy.json), which limits each cluster's controller to the file systems tagged for that cluster.
See [Cluster tag isolation](cluster-tag-isolation.md) for how the policy works.
If clusters in the account share file systems, attach [`shared-policy.json`](iam/shared-policy.json) instead and skip the cluster tag settings: it limits the controller to file systems this driver created, in any cluster. See [Choosing a policy](cluster-tag-isolation.md#choosing-a-policy).
If your cluster is in the AWS GovCloud Regions, replace `arn:aws:` with `arn:aws-us-gov:`. Likewise, if your cluster is in the AWS China Regions, replace `arn:aws:` with `arn:aws-cn:`.

> [!IMPORTANT]
> These steps are for new clusters. A cluster that already has file systems provisioned by the driver must migrate first:
> those file systems have no cluster tag, so `enforce` ignores them and the tag-based policy denies expanding or deleting them.
> Follow [Existing clusters](cluster-tag-isolation.md#existing-clusters) to backfill the cluster tag before you set `enforce`.

#### Amazon EKS: EKS Pod Identity (Recommended)
With Pod Identity, one IAM role and policy can be shared by every cluster in the account, and each cluster's controller is still limited to its own file systems.
Create the role once and repeat only the association (step 2) and step 3 for each cluster. See [Roles per cluster](cluster-tag-isolation.md#the-tag-based-policy).

1. Install the [EKS Pod Identity Agent](https://docs.aws.amazon.com/eks/latest/userguide/pod-id-agent-setup.html) add-on, if it is not already installed:
   ```sh
   aws eks create-addon --cluster-name my-cluster --addon-name eks-pod-identity-agent
   ```

2. Create a role with the standard [EKS Pod Identity trust policy](https://docs.aws.amazon.com/eks/latest/userguide/pod-id-role.html), attach the policy, and associate the role with the controller service account:
   ```sh
   aws iam create-role --role-name my-cluster-fsx-csi-controller \
     --assume-role-policy-document file://pod-identity-trust-policy.json
   aws iam put-role-policy --role-name my-cluster-fsx-csi-controller --policy-name fsx-csi-cluster-tag \
     --policy-document file://docs/iam/cluster-tag-policy.json
   aws eks create-pod-identity-association --cluster-name my-cluster --namespace kube-system \
     --service-account fsx-csi-controller-sa \
     --role-arn arn:aws:iam::111122223333:role/my-cluster-fsx-csi-controller
   ```

3. When you [deploy the driver](#deploy-driver), set the cluster tag options:
   ```sh
   helm upgrade --install aws-fsx-csi-driver aws-fsx-csi-driver/aws-fsx-csi-driver --namespace kube-system \
     --set controller.clusterName=my-cluster \
     --set controller.clusterTagKey=eks:eks-cluster-name \
     --set controller.clusterTagIsolation=enforce
   ```
   Without Helm, pass the same values as controller flags:
   `--cluster-name=my-cluster --cluster-tag-key=eks:eks-cluster-name --cluster-tag-isolation=enforce`.

#### Other credentials
On a self-managed cluster, or with IRSA or another credential type, give the controller [`docs/iam/shared-policy.json`](iam/shared-policy.json): it needs no Pod Identity session tags.
See [IRSA and other credentials](cluster-tag-isolation.md#irsa-and-other-credentials).

### Configure driver toleration settings
By default, the driver controller tolerates taint `CriticalAddonsOnly` and has `tolerationSeconds` configured as `300`; and the driver node tolerates all taints.
If you don't want to deploy the driver node on all nodes, please set Helm `Value.node.tolerateAllTaints` to false before deployment.
Add policies to `Value.node.tolerations` to configure customized toleration for nodes.

### Configure node startup taint
There are potential race conditions on node startup (especially when a node is first joining the cluster) where pods/processes that rely on the FSx for Lustre CSI Driver can act on a node before the FSx for Lustre CSI Driver is able to start up and become fully ready. To combat this, the FSx for Lustre CSI Driver contains a feature to automatically remove a taint from the node on startup. Users can taint their nodes when they join the cluster and/or on startup, to prevent other pods from running and/or being scheduled on the node prior to the FSx for Lustre CSI Driver becoming ready.

This feature is activated by default, and cluster administrators should use the taint `fsx.csi.aws.com/agent-not-ready:NoExecute` (any effect will work, but `NoExecute` is recommended). For example, EKS Managed Node Groups [support automatically tainting nodes](https://docs.aws.amazon.com/eks/latest/userguide/node-taints-managed-node-groups.html).

### Configure Lustre client userspace installation
To have the driver node install the Lustre client userspace tools on Amazon Linux 2 and Amazon Linux 2023 hosts, set Helm `Value.node.lustreClientInstaller.enabled` to true before deployment.

### Deploy driver
You may deploy the FSx for Lustre CSI driver via Kustomize or Helm

#### Kustomize
```sh
kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.11"
```

*Note: Using the master branch to deploy the driver is not supported as the master branch may contain upcoming features incompatible with the currently released stable version of the driver.*

#### Helm
- Add the `aws-fsx-csi-driver` Helm repository.
```sh
helm repo add aws-fsx-csi-driver https://kubernetes-sigs.github.io/aws-fsx-csi-driver
helm repo update
```

- Install the latest release of the driver.
```sh
helm upgrade --install aws-fsx-csi-driver \
    --namespace kube-system \
    aws-fsx-csi-driver/aws-fsx-csi-driver
```

Review the [configuration values](https://github.com/kubernetes-sigs/aws-fsx-openzfs-csi-driver/blob/master/charts/aws-fsx-csi-driver/values.yaml) for the Helm chart.

#### Once the driver has been deployed, verify the pods are running:
```sh
kubectl get pods -n kube-system -l app.kubernetes.io/name=aws-fsx-csi-driver
```

#### EKS Auto Mode

[EKS Auto Mode](https://docs.aws.amazon.com/eks/latest/userguide/automode.html)  extends AWS management of Kubernetes clusters beyond the cluster itself, to allow AWS to also set up and manage the infrastructure that enables the smooth operation of your workloads. Deploying support for Amazon FSx for Lustre CSI Driver onto an Auto Mode cluster requires a few specific changes from a general install.

- Specify the AWS region in the controller configuration
- Use a newer Lustre filesystem version than the default which is compatible with Auto Mode's newer kernel

As a [best practice](https://docs.aws.amazon.com/eks/latest/best-practices/identity-and-access-management.html#_identities_and_credentials_for_eks_pods_recommendations), Auto Mode blocks access to IMDS for Pods that don't use host networking. To eliminate the need for the FSX controller to use IMDS, you need to configure the region for the controller.

```sh
helm upgrade --install aws-fsx-csi-driver \
    --namespace kube-system \
    aws-fsx-csi-driver/aws-fsx-csi-driver \
    --set controller.region=us-west-2
```

The supported Lustre filesystem versions [varies](https://docs.aws.amazon.com/fsx/latest/LustreGuide/lustre-client-matrix.html) based on the Linux Kernel version of the Node. The current latest filesystem version which is compatible with Auto Mode is 2.15. This is configured in the FSX storage class:

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fsx-sc
provisioner: fsx.csi.aws.com
parameters:
  fileSystemTypeVersion: "2.15"
  subnetId: subnet-1234567890abcdef0
  securityGroupIds: sg-1234567890abcdef0
```
