# Cluster tag isolation

With `AmazonFSxFullAccess`, the controller of one Kubernetes cluster has IAM permissions to modify the FSx for Lustre
file systems of other clusters in the same AWS account. This page describes how to tag the file systems the driver
creates with the name of the cluster that owns them, and how to attach an IAM policy that limits each cluster's driver
to its own file systems.

The examples use EKS Pod Identity, account `111122223333`, region `us-west-2` and cluster `my-cluster`. For IAM roles for
service accounts (IRSA), see [IRSA and other credentials](#irsa-and-other-credentials).

- [Choosing a policy](#choosing-a-policy)
- [Moving off `AmazonFSxFullAccess`](#moving-off-amazonfsxfullaccess)
- [New clusters](#new-clusters)
- [Existing clusters](#existing-clusters)
- [What each mismatch does](#what-each-mismatch-does)
- [Settings](#settings)

## Choosing a policy

The controller tags every file system it creates `fsx.csi.aws.com/cluster=true`, and, when `--cluster-name` and
`--cluster-tag-key` are set, also `<cluster-tag-key>=<cluster-name>`.

| Policy | The controller can change | Use it when |
|---|---|---|
| [`docs/iam/shared-policy.json`](iam/shared-policy.json) | Every file system this driver created, in any cluster in the account | The clusters in the account share file systems, or you don't need them isolated |
| A copy of `shared-policy.json` with your own conditions | What your conditions allow | Some clusters share file systems and others don't |
| [`docs/iam/cluster-tag-policy.json`](iam/cluster-tag-policy.json) | Only file systems tagged for its own cluster | Each cluster should only reach its own file systems |
| `AmazonFSxFullAccess` (AWS managed) | Every FSx resource in the account, including file systems the driver never created | Not recommended |

The AWS managed policy `AmazonFSxFullAccess` also works, but it grants more than the driver needs including file systems the driver never created.
Move off it as described in [Moving off `AmazonFSxFullAccess`](#moving-off-amazonfsxfullaccess).

## The shared policy

[`docs/iam/shared-policy.json`](iam/shared-policy.json) limits the controller to file systems that this driver created:

* `fsx:CreateFileSystem` is allowed only when the request tags the new file system `fsx.csi.aws.com/cluster=true`.
* `fsx:TagResource`, `fsx:UpdateFileSystem` (expand) and `fsx:DeleteFileSystem` are allowed only on file systems that
  carry `fsx.csi.aws.com/cluster=true`.
* `fsx:DescribeFileSystems`, the FSx service-linked roles and `s3:ListBucket` are the same as in the tag-based policy
  (see the table below).

It reads no principal or session tags, so it works with EKS Pod Identity, IRSA, node roles and any other credentials. Every
cluster with this policy can change every file system that any cluster's driver created in the account. File systems an
administrator created without the tag, for example for static provisioning, can still be mounted, but the controller
can't expand or delete them.

## Moving off `AmazonFSxFullAccess`

The controller adds `fsx.csi.aws.com/cluster=true` to existing file systems it provisions, so
the shared policy covers them. In `disable` (the default) and `backfill`, it sweeps at startup and then every hour; see
[What the sweep does](#what-the-sweep-does).

1. Upgrade the driver, and keep `AmazonFSxFullAccess` attached.
2. Wait until every controller replica logs `cluster tag backfill complete`, as in
   [Confirm backfill is complete](#confirm-backfill-is-complete), with `TAG_KEY=fsx.csi.aws.com/cluster` and
   `CLUSTER=true`. File systems listed for an administrator weren't created by this driver; tag them
   `fsx.csi.aws.com/cluster=true` if the controller should expand or delete them.
3. Attach `shared-policy.json`, then detach `AmazonFSxFullAccess`.

To isolate clusters instead, follow [Existing clusters](#existing-clusters) with `AmazonFSxFullAccess` still attached.

## The tag-based policy

[`docs/iam/cluster-tag-policy.json`](iam/cluster-tag-policy.json) restricts the controller to file systems tagged for
its own cluster, as far as the FSx APIs allow:

* `fsx:CreateFileSystem` is allowed only when the request tags the new file system
  `eks:eks-cluster-name=<the caller's eks-cluster-name session tag>`.
* `fsx:UpdateFileSystem` (expand) and `fsx:DeleteFileSystem` are allowed only on file systems that already carry that tag
  with the caller's cluster name.
* `fsx:TagResource` is allowed only when the file system and the request both carry the caller's cluster tag. FSx
  authorizes the tags passed to `CreateFileSystem` as `fsx:TagResource`, and it has no condition key that limits a grant
  to tagging at creation. So the controller can also add other tags to its own cluster's file systems, as long as the
  request repeats its own cluster tag. It can never add, change or remove the cluster tag of another cluster's file
  system, and it cannot call `fsx:UntagResource`.

The policy needs no editing: it uses the `eks-cluster-name` and `eks-cluster-arn` session tags that EKS Pod Identity adds.
Credentials without those tags, such as IRSA or the node role, are denied.

**Roles per cluster.** With EKS Pod Identity, one IAM role and policy can serve every cluster in the account and still
keep them isolated. The Pod Identity trust policy does not name a cluster, and EKS sets the `eks-cluster-name` and
`eks-cluster-arn` session tags for whichever cluster's controller assumes the role, so each controller only reaches its
own cluster's file systems. Associate the same role with `fsx-csi-controller-sa` in each cluster.

Actions that cannot be scoped by tag:

| Action | Why | What it can still reach |
|---|---|---|
| `fsx:DescribeFileSystems` | No resource-level permissions | Metadata and tags of every FSx file system in the account, in the cluster's region |
| `iam:CreateServiceLinkedRole` for `fsx.amazonaws.com` | Account-level, global | Only the FSx service-linked role, which is the same for every cluster |
| `iam:CreateServiceLinkedRole`, `iam:AttachRolePolicy`, `iam:PutRolePolicy` on `role/aws-service-role/s3.data-source.lustre.fsx.amazonaws.com/*` | FSx creates one role per file system with a data repository, so the name is not known in advance; global | Any role under that service-linked role path, the same as `AmazonFSxFullAccess` |
| `s3:ListBucket` | Data repository buckets belong to you and carry no cluster tag | Any bucket in the same account, in the cluster's region, and only through FSx (`aws:ViaAWSService`). Narrow `Resource` to your data repository buckets if you can |

The controller never calls KMS. For a StorageClass `kmsKeyId`, the key policy must allow FSx to use the key, as it does
today.

**Region.** Every FSx and S3 statement requires the requested region to be the region in the `eks-cluster-arn` session
tag, so the controller can only act in its own cluster's region. The IAM statements are global, because IAM is a
global service.

**Account and partition.** ARNs use `*` for the partition, region and account, so the same file works in every
partition. FSx file systems have no resource-based policies, so a role cannot reach file systems in other accounts.
`s3:ListBucket` also requires `aws:ResourceAccount` to be the caller's own account. Cross-account data repository
buckets are not supported by this policy.

### IRSA and other credentials

The tag-based policy reads the `eks-cluster-name` and `eks-cluster-arn` session tags that EKS Pod Identity adds, so it
denies IRSA, node roles and other credentials without them. Use [`shared-policy.json`](iam/shared-policy.json) with
those credentials, or adapt the tag-based policy to read the cluster name from somewhere they provide, such as a tag on
the IAM role.

## The backfill policy

[`docs/iam/cluster-tag-backfill-policy.json`](iam/cluster-tag-backfill-policy.json) is attached together with
`cluster-tag-policy.json` only while `--cluster-tag-isolation=backfill` is set. It grants exactly one thing:
`fsx:TagResource` on a file system that has a `CSIVolumeName` tag (this driver created it) and no cluster tag yet, with a
request that sets the `eks:eks-cluster-name` tag to the caller's own cluster name and `fsx.csi.aws.com/cluster` to
`true`, and no other tags. Detach it after you set `enforce`.

`CSIVolumeName` does not name the cluster. While the backfill policy is attached, any cluster in the account whose role
has it could add its own cluster tag to another cluster's untagged file system. The controller itself only tags file
systems behind its own cluster's PersistentVolumes. Keep the backfill window short, or tag the file systems yourself
(see [Confirm backfill is complete](#confirm-backfill-is-complete)).

## Isolation covers the whole account

A cluster's file systems are protected only when every role for this controller in the account uses
`cluster-tag-policy.json`. A cluster whose controller uses `shared-policy.json` has IAM permissions to modify every file
system this driver created, and one that uses `AmazonFSxFullAccess` every file system in the account.

## New clusters

Every file system is tagged when it is created, so no backfill is needed.

1. Create the role and attach the policy:

   ```sh
   aws iam create-role --role-name my-cluster-fsx-csi-controller \
     --assume-role-policy-document file://pod-identity-trust-policy.json
   aws iam put-role-policy --role-name my-cluster-fsx-csi-controller --policy-name fsx-csi-cluster-tag \
     --policy-document file://docs/iam/cluster-tag-policy.json
   aws eks create-pod-identity-association --cluster-name my-cluster --namespace kube-system \
     --service-account fsx-csi-controller-sa \
     --role-arn arn:aws:iam::111122223333:role/my-cluster-fsx-csi-controller
   ```

2. Install the driver with the cluster name, the key and `enforce`:

   ```sh
   helm upgrade --install aws-fsx-csi-driver aws-fsx-csi-driver/aws-fsx-csi-driver --namespace kube-system \
     --set controller.clusterName=my-cluster \
     --set controller.clusterTagKey=eks:eks-cluster-name \
     --set controller.clusterTagIsolation=enforce
   ```

   Without Helm, pass the same values as controller flags:
   `--cluster-name=my-cluster --cluster-tag-key=eks:eks-cluster-name --cluster-tag-isolation=enforce`.

## Existing clusters

`--cluster-tag-isolation` must match the IAM policy attached to the controller. The controller cannot check this.

| State | IAM attached | Controller behavior |
|---|---|---|
| `disable` (default) | `AmazonFSxFullAccess` or `shared-policy.json` | No cluster isolation. Adds `fsx.csi.aws.com/cluster=true` to existing file systems this driver provisioned, and the cluster tag to new file systems when `--cluster-name` and `--cluster-tag-key` are set |
| `backfill` | `AmazonFSxFullAccess` (or `shared-policy.json`), `cluster-tag-policy.json` and `cluster-tag-backfill-policy.json` | Requires the cluster name and key. Adds the cluster tag and `fsx.csi.aws.com/cluster=true` to existing file systems this driver provisioned. Does not filter. |
| `enforce` | `cluster-tag-policy.json` (after the rollout below) | Requires the cluster name and key. Ignores file systems that lack this cluster's tag value: expanding or deleting one fails with `FailedPrecondition`, without an AWS call. Does not backfill. |

The value is a string and must be exactly `disable`, `backfill` or `enforce`; an empty value means `disable`. The
chart and the controller reject any other value. At startup the controller logs one line, `cluster tag isolation
settings`, with `clusterName`, `tagKey`, `clusterTagIsolation` and, in `backfill`, `backfillInterval`.

In `enforce`, each ignored file system is logged at info level as `ignoring resource not tagged for this cluster`, with
`resource` (the file system ID), `tagKey`, `clusterName` and `tagValue` (the value found, empty when the file system has
no cluster tag).

Roll out in this order. The steps name `AmazonFSxFullAccess`; if the controller uses `shared-policy.json`, keep that
one in its place. Keep it attached until `enforce` is set: the tag-based policy allows
nothing on file systems without this cluster's tag, and it needs the cluster name set before the controller can create
file systems. IAM allows an action when any attached policy does, so adding policies first changes nothing.

1. Attach `cluster-tag-policy.json` and `cluster-tag-backfill-policy.json` to the controller's role, **alongside**
   `AmazonFSxFullAccess`.
2. Set `backfill`. From now on, new file systems get the cluster tag too:

   ```sh
   helm upgrade aws-fsx-csi-driver aws-fsx-csi-driver/aws-fsx-csi-driver --namespace kube-system --reuse-values \
     --set controller.clusterName=my-cluster \
     --set controller.clusterTagKey=eks:eks-cluster-name \
     --set controller.clusterTagIsolation=backfill
   ```

3. Wait until every controller replica logs `cluster tag backfill complete`. Each replica sweeps on its own and logs
   this line once per process, after its first sweep with `failed`, `needsAdministrator` and `remaining` all zero.
   Check all replicas, not just the one `deploy/` picks:

   ```sh
   kubectl -n kube-system logs -l app=fsx-csi-controller -c fsx-plugin --prefix --tail=-1 |
     grep -E 'cluster tag backfill (complete|sweep finished)'
   ```

   Until then, tag every file system listed in `needsAdministratorResources` on the latest
   `cluster tag backfill sweep finished` line. The next sweep, at most an hour later, picks up your tags; restarting
   the controller sweeps at once. A restarted replica logs `complete` again only after its own clean sweep. See
   [Confirm backfill is complete](#confirm-backfill-is-complete) for a check that does not depend on logs.
4. Set `enforce`:

   ```sh
   helm upgrade aws-fsx-csi-driver aws-fsx-csi-driver/aws-fsx-csi-driver --namespace kube-system --reuse-values \
     --set controller.clusterTagIsolation=enforce
   ```

5. Detach `AmazonFSxFullAccess` and `cluster-tag-backfill-policy.json`. Only `cluster-tag-policy.json` stays, and
   the controller's IAM permissions are now limited to this cluster's file systems.

### What the sweep does

In the `disable` and `backfill` states the controller sweeps at startup and then every hour; in `enforce` it doesn't.
The sweep adds the driver's tags: `fsx.csi.aws.com/cluster=true` in `disable`, and that plus this cluster's tag in
`backfill`. It visits the file system behind every PersistentVolume of driver `fsx.csi.aws.com`:

* A file system that already has the tags is left alone.
* A file system this driver provisioned gets the missing tags, in one `TagResource` request that contains only the
  driver's tags. The driver provisioned it when the PersistentVolume has the annotation
  `pv.kubernetes.io/provisioned-by: fsx.csi.aws.com` and the file system's `CSIVolumeName` tag equals the
  PersistentVolume name.
* Any other file system without the tags is not changed, and is reported for an administrator. This covers
  statically provisioned file systems, and, in `backfill`, file systems tagged for another cluster.

Each controller replica runs its own sweep, so a line can appear once per replica.

Each sweep logs exactly one info line, `cluster tag backfill sweep finished`, with these fields:

| Field | Meaning |
|---|---|
| `tagged` | `TagResource` calls that succeeded in this sweep |
| `failed` | `TagResource` calls that failed in this sweep |
| `needsAdministrator` | Distinct file systems this cluster uses that an administrator must tag |
| `remaining` | File systems, or whole listings, the sweep could not check: a `DescribeFileSystems` call failed (1 per file system), or listing PersistentVolumes failed (1) |
| `needsAdministratorResources` | The first 20 file system IDs counted in `needsAdministrator`, with `needsAdministratorTotal` when there are more. Omitted when empty |
| `failedResources` | The first 20 file system IDs counted in `failed`, with `failedTotal` when there are more. Omitted when empty |
| `firstError` | The first error of the sweep. Only present when `failed` or `remaining` is above zero |

`cluster tag backfill complete`, with `clusterName` and `tagKeys` (the keys the sweep adds), is logged once per
controller process, after the
first sweep that ends with `failed`, `needsAdministrator` and `remaining` all zero. Later sweeps still log their
summary line, and never log `complete` again.

The lines for single file systems are logged only at verbosity 4 (`--v=4`, Helm `controller.logLevel: 4`), with the
fields `resource` (the file system ID) and `tagKeys`: `added cluster tag to existing resource`,
`failed to add cluster tag to existing resource` (also with `err`) and
`resource needs cluster tag from an administrator`. Also at verbosity 4: `cluster tag backfill could not describe file
system` and `cluster tag backfill could not list PersistentVolumes` (counted in `remaining`), and `cluster tag backfill:
file system not found` (with `resource` and `persistentVolume`) for a PersistentVolume whose file system no longer
exists. A missing file system is skipped and not counted, so it does not block `complete`.

### What each mismatch does

| Setting | Policy attached | Result |
|---|---|---|
| `backfill` | `cluster-tag-backfill-policy.json` missing | Every `TagResource` call fails: each sweep's summary line counts them in `failed`, lists them in `failedResources`, and shows `AccessDenied` in `firstError`. `cluster tag backfill complete` never appears. |
| `enforce` | `cluster-tag-backfill-policy.json` still attached | No effect. The controller does not backfill in `enforce`. Detach the policy. |
| `backfill` or `enforce` | `AmazonFSxFullAccess` still attached | Everything works, but IAM doesn't isolate: in `enforce` the controller skips other clusters' file systems, but its role could still change them. Detach `AmazonFSxFullAccess` after `enforce`. |
| `disable` | `cluster-tag-policy.json` without `AmazonFSxFullAccess`, before the cluster name is set | Creates fail with `AccessDenied`, and expanding or deleting an untagged file system is denied. This is what happens when `AmazonFSxFullAccess` is detached too early; reattach it and follow the steps above. |

### Confirm backfill is complete

Backfill is complete when every controller replica has logged `cluster tag backfill complete`. Read the latest summary
line of each replica; `needsAdministratorResources` lists the file systems an administrator must tag:

```sh
kubectl -n kube-system logs -l app=fsx-csi-controller -c fsx-plugin --prefix --tail=-1 |
  grep -E 'cluster tag backfill (complete|sweep finished)'
```

`--tail=-1` is needed because `kubectl logs -l` prints only the last 10 lines of each pod by default. If a summary line
has `needsAdministratorTotal`, more than 20 file systems are waiting: tag the listed ones and wait for the next sweep,
set `controller.logLevel: 4` to log each one as `resource needs cluster tag from an administrator`, or use the check
below.

These log lines come from a test rollout. They were captured with an earlier log format and are shown here in the
current one. The first sweep tagged one file system and found one that needs an administrator. After the administrator
tagged it, the next sweep, an hour later, logged completion:

```
I1021 23:18:58.611302       1 cluster_tag.go:114] "cluster tag isolation settings" clusterName="my-cluster" tagKey="eks:eks-cluster-name" clusterTagIsolation="backfill" backfillInterval="1h0m0s"
I1021 23:18:58.891728       1 cluster_tag.go:295] "cluster tag backfill sweep finished" tagged=1 failed=0 needsAdministrator=1 remaining=0 needsAdministratorResources=["fs-0e95ad8aed43a6f2f"]
I1022 00:18:59.088718       1 cluster_tag.go:295] "cluster tag backfill sweep finished" tagged=0 failed=0 needsAdministrator=0 remaining=0
I1022 00:18:59.088750       1 cluster_tag.go:297] "cluster tag backfill complete" clusterName="my-cluster" tagKeys=["eks:eks-cluster-name","fsx.csi.aws.com/cluster"]
```

The commands below use two shell variables. Set `TAG_KEY` to the `--cluster-tag-key` value:

```sh
TAG_KEY=eks:eks-cluster-name
CLUSTER=my-cluster
```

To list the file systems that still need tagging, check every file system referenced by the cluster's FSx
PersistentVolumes (`volumeHandle` is the file system ID) and print only those without `$TAG_KEY=$CLUSTER`, with the
value found (`None` when the file system has no cluster tag). Empty output means every file system is tagged;
`describe failed` means the call failed, for example because the file system was deleted. The IDs are described one
at a time because a single missing ID fails a batched `describe-file-systems` call:

```sh
kubectl get pv -o jsonpath='{range .items[?(@.spec.csi.driver=="fsx.csi.aws.com")]}{.spec.csi.volumeHandle}{"\n"}{end}' |
  sort -u | while read -r id; do
    aws fsx describe-file-systems --file-system-ids "$id" --output text \
      --query "FileSystems[?!(Tags[?Key=='$TAG_KEY' && Value=='$CLUSTER'])].[FileSystemId, Tags[?Key=='$TAG_KEY']|[0].Value]" ||
      printf '%s\tdescribe failed\n' "$id"
  done
```

A file system that already carries another cluster's value is shared with, or belongs to, that cluster: decide which
cluster owns it before you change the tag. To tag a file system, build its ARN from the ID
(`arn:aws:fsx:<region>:<account>:file-system/<ID>`):

```sh
aws resourcegroupstaggingapi tag-resources \
  --resource-arn-list arn:aws:fsx:us-west-2:111122223333:file-system/fs-0123456789abcdef0 \
  --tags "$TAG_KEY=$CLUSTER"
```

## Settings

The controller is configured with flags only.

| Flag | Helm value | Default | Description |
|---|---|---|---|
| `--cluster-name` | `controller.clusterName` | `""` | Name of the cluster. The value of the cluster tag. |
| `--cluster-tag-key` | `controller.clusterTagKey` | `""` | Key of the cluster tag. Empty writes no cluster tag. Use `eks:eks-cluster-name` with `cluster-tag-policy.json`. |
| `--cluster-tag-isolation` | `controller.clusterTagIsolation` | `disable` | `disable`, `backfill` or `enforce`, as described above. `backfill` and `enforce` fail at startup without `--cluster-name` and `--cluster-tag-key`. |
| `--extra-tags` | `controller.extraTags` (map) | `""` | Tags (`key1=value1,key2=value2`) added to every AWS resource the controller creates. Not used to find, filter or isolate resources, and not the cluster tag. A StorageClass `extraTags` value overrides these for the same key. Neither can set `CSIVolumeName`, `fsx.csi.aws.com/cluster` or the `--cluster-tag-key` key. |

Every new file system is tagged `fsx.csi.aws.com/cluster=true`. With `--cluster-name` and `--cluster-tag-key` set, it is
also tagged `<cluster-tag-key>=<cluster-name>`, in every state, including `disable`. A StorageClass whose `extraTags`
sets the cluster tag key, `fsx.csi.aws.com/cluster` or `CSIVolumeName` fails volume creation with `InvalidArgument`. An
`--extra-tags` value that sets any of them fails at startup, and so does a `--cluster-tag-key` of
`fsx.csi.aws.com/cluster`.
