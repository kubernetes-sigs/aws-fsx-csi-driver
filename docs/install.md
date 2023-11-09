# Installation

## Prerequisites

* Kubernetes Version >= 1.20

* If you are using a self managed cluster, ensure the flag `--allow-privileged=true` for `kube-apiserver`.

## Installation
### Set up driver permissions
The driver requires IAM permissions to interact with the Amazon FSx for Lustre service to create/delete file systems and volumes on the user's behalf.
There are several methods to grant the driver IAM permissions:
* Using [IAM roles for ServiceAccounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) (**Recommended**) - Create a Kubernetes service account for the driver and attach the AmazonFSxFullAccess AWS-managed policy to it with the following command. If your cluster is in the AWS GovCloud Regions, then replace arn:aws: with arn:aws-us-gov. Likewise, if your cluster is in the AWS China Regions, replace arn:aws: with arn:aws-cn:
```sh

export cluster_name=my-csi-fsx-cluster
export region_code=region-code

eksctl create iamserviceaccount \
    --name fsx-csi-controller-sa \
    --namespace kube-system \
    --cluster $cluster_name \
    --attach-policy-arn arn:aws:iam::aws:policy/AmazonFSxFullAccess \
    --approve \
    --role-name AmazonEKSFSxLustreCSIDriverFullAccess \
    --region $region_code
```

* Using IAM [instance profile](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html) - Create the following IAM policy and attach the policy to the instance profile IAM role of your cluster's worker nodes.
  See [here](https://docs.aws.amazon.com/eks/latest/userguide/create-node-role.html) for guidelines on how to access your EKS node IAM role.
```sh
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreateServiceLinkedRole",
        "iam:AttachRolePolicy",
        "iam:PutRolePolicy"
       ],
      "Resource": "arn:aws:iam::*:role/aws-service-role/s3.data-source.lustre.fsx.amazonaws.com/*"
    },
    {
      "Action":"iam:CreateServiceLinkedRole",
      "Effect":"Allow",
      "Resource":"*",
      "Condition":{
        "StringLike":{
          "iam:AWSServiceName":[
            "fsx.amazonaws.com"
          ]
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "fsx:CreateFileSystem",
        "fsx:DeleteFileSystem",
        "fsx:DescribeFileSystems",
        "fsx:TagResource"
      ],
      "Resource": ["*"]
    }
  ]
}
```



### Configure driver toleration settings
By default, the driver controller tolerates taint `CriticalAddonsOnly` and has `tolerationSeconds` configured as `300`; and the driver node tolerates all taints.
If you don't want to deploy the driver node on all nodes, please set Helm `Value.node.tolerateAllTaints` to false before deployment.
Add policies to `Value.node.tolerations` to configure customized toleration for nodes.

### Configure node startup taint
There are potential race conditions on node startup (especially when a node is first joining the cluster) where pods/processes that rely on the FSx for Lustre CSI Driver can act on a node before the FSx for Lustre CSI Driver is able to start up and become fully ready. To combat this, the FSx for Lustre CSI Driver contains a feature to automatically remove a taint from the node on startup. Users can taint their nodes when they join the cluster and/or on startup, to prevent other pods from running and/or being scheduled on the node prior to the FSx for Lustre CSI Driver becoming ready.

This feature is activated by default, and cluster administrators should use the taint `fsx.csi.aws.com/agent-not-ready:NoExecute` (any effect will work, but `NoExecute` is recommended). For example, EKS Managed Node Groups [support automatically tainting nodes](https://docs.aws.amazon.com/eks/latest/userguide/node-taints-managed-node-groups.html).

### Deploy driver
You may deploy the FSx for Lustre CSI driver via Kustomize or Helm

#### Kustomize
```sh
kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.1"
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