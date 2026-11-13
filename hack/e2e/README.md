# Usage

run.sh will build and push a driver image, create a kops cluster, helm install the driver pointing to the built image, run ginkgo tests, then clean everything up.

See below for an example.

KOPS_STATE_FILE is an S3 bucket you have write access to.

TEST_ID is a token used for idempotency.

For more details, see the script itself.

For more examples, see the top-level Makefile.

```
TEST_PATH=./tests/e2e/... \
TEST_ID=18512 \
CLEAN=false \
KOPS_STATE_FILE=s3://mattwon \
AWS_REGION=us-west-2 \
AWS_AVAILABILITY_ZONES=us-west-2a \
GINKGO_FOCUS=".*" \
GINKGO_NODES=1 \
./hack/e2e/run.sh
```

## Cluster tag isolation tests

With `CLUSTER_TYPE=eksctl` (`make test-e2e-eksctl`; `make test-e2e` stays on kops), run.sh first creates an IAM role
`fsx-csi-e2e-$TEST_ID` with the Pod Identity trust policy (`pod-identity.sh create-role`), starting with
`iam/legacy-policy.json`. It then renders `eksctl-cluster.yaml` with gomplate and creates the cluster from it. The
config installs the EKS Pod Identity Agent add-on, associates the role with `kube-system/fsx-csi-controller-sa`, and
blocks pod access to IMDS. EKS adds the `eks-cluster-name` and `eks-cluster-arn` session tags that the tag-based
policies read.

After the other tests, run.sh runs the `[fsx-csi-isolation]` specs in `tests/e2e/isolation_test.go` alone. They walk
the rollout in `docs/cluster-tag-isolation.md`, swapping the role's policies with `pod-identity.sh set-policies` and
the controller settings with `helm-upgrade.sh`:

1. `disable` with `docs/iam/shared-policy.json`: a new file system has `fsx.csi.aws.com/cluster=true` and no cluster
   tag.
2. `backfill` with `docs/iam/cluster-tag-policy.json` and `cluster-tag-backfill-policy.json` added alongside the
   shared policy, as in the documented rollout: the file system gets
   `eks:eks-cluster-name=<cluster>`, and the controller logs `cluster tag backfill complete`.
3. `enforce`, then detach the shared and backfill policies, leaving `cluster-tag-policy.json`: a new file system is
   tagged at creation.
4. Negative, controller: deleting a PersistentVolume for a file system tagged for another cluster fails with
   `FailedPrecondition`, and the file system is left alone.
5. Negative, IAM: an aws-cli pod with the controller's service account, and so its role, can describe this cluster's
   file system, but tagging or deleting the other cluster's is denied.
6. Deleting the claims deletes this cluster's file systems, including the backfilled one.

Set `ISOLATION_E2E=false` to skip them. run.sh deletes the role when `CLEAN=true`; the association is deleted with the
cluster.
