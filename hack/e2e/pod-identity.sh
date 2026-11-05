#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Manages the IAM role the controller uses through EKS Pod Identity on an eksctl
# cluster. EKS adds the eks-cluster-name and eks-cluster-arn session tags that the
# docs/iam/cluster-tag-*.json policies read. eksctl-cluster.yaml installs the Pod
# Identity Agent and associates this role with the controller service account.
#
#   pod-identity.sh create-role <role name>                   prints the role ARN; reuses an existing role
#   pod-identity.sh set-policies <role name> <policy file>...  replaces every inline policy
#   pod-identity.sh delete-role <role name>
#   pod-identity.sh wait-for-credentials <kubeconfig>         restarts the controller until it has credentials

set -euo pipefail

NAMESPACE=${NAMESPACE:-kube-system}

# The trust policy names no cluster, so the role can be created before the cluster.
function pod_identity_create_role() {
  local role=${1}
  local account arn
  # A rerun against the same TEST_ID and cluster reuses the role.
  if arn=$(aws iam get-role --role-name "${role}" --query Role.Arn --output text 2>/dev/null); then
    echo "${arn}"
    return 0
  fi
  account=$(aws sts get-caller-identity --query Account --output text)

  local trust
  trust=$(
    cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"Service": "pods.eks.amazonaws.com"},
      "Action": ["sts:AssumeRole", "sts:TagSession"],
      "Condition": {"StringEquals": {"aws:SourceAccount": "${account}"}}
    }
  ]
}
EOF
  )
  aws iam create-role --role-name "${role}" \
    --assume-role-policy-document "${trust}" \
    --query Role.Arn --output text
}

function pod_identity_set_policies() {
  local role=${1}
  shift
  local want=() file name
  for file in "$@"; do
    name=$(basename "${file}" .json)
    want+=("${name}")
    aws iam put-role-policy --role-name "${role}" --policy-name "${name}" --policy-document "file://${file}"
  done
  for name in $(aws iam list-role-policies --role-name "${role}" --query 'PolicyNames[]' --output text); do
    if [[ ! " ${want[*]} " == *" ${name} "* ]]; then
      aws iam delete-role-policy --role-name "${role}" --policy-name "${name}"
    fi
  done
}

# EKS injects Pod Identity credentials (AWS_CONTAINER_CREDENTIALS_FULL_URI) into a
# pod when it is created, but only once the association has taken effect. A
# controller pod created right after the cluster can miss it and then has no
# credentials, since pods can't reach IMDS. Restart the controller until its pod
# has them.
function pod_identity_wait_for_credentials() {
  local kubeconfig=${1} attempt uri
  for attempt in 1 2 3 4 5 6; do
    uri=$(kubectl --kubeconfig "${kubeconfig}" -n "${NAMESPACE}" get pods -l app=fsx-csi-controller \
      -o jsonpath='{.items[*].spec.containers[?(@.name=="fsx-plugin")].env[?(@.name=="AWS_CONTAINER_CREDENTIALS_FULL_URI")].value}')
    if [[ -n "${uri}" ]]; then
      return 0
    fi
    echo "controller has no Pod Identity credentials yet (attempt ${attempt}); restarting it" >&2
    sleep 10
    kubectl --kubeconfig "${kubeconfig}" -n "${NAMESPACE}" rollout restart deployment/fsx-csi-controller >&2
    kubectl --kubeconfig "${kubeconfig}" -n "${NAMESPACE}" rollout status deployment/fsx-csi-controller --timeout=5m >&2
  done
  echo "controller still has no Pod Identity credentials" >&2
  return 1
}

# The association is deleted with the cluster.
function pod_identity_delete_role() {
  local role=${1} name
  if ! aws iam get-role --role-name "${role}" >/dev/null 2>&1; then
    return 0
  fi
  for name in $(aws iam list-role-policies --role-name "${role}" --query 'PolicyNames[]' --output text); do
    aws iam delete-role-policy --role-name "${role}" --policy-name "${name}"
  done
  aws iam delete-role --role-name "${role}"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  case "${1:-}" in
  create-role) pod_identity_create_role "${2}" ;;
  set-policies) pod_identity_set_policies "${@:2}" ;;
  delete-role) pod_identity_delete_role "${2}" ;;
  wait-for-credentials) pod_identity_wait_for_credentials "${2}" ;;
  *)
    echo "usage: $0 create-role|set-policies|delete-role|wait-for-credentials ..." >&2
    exit 1
    ;;
  esac
fi
