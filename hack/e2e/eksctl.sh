#!/bin/bash

set -euo pipefail

function eksctl_install() {
  INSTALL_PATH=${1}
  EKSCTL_VERSION=${2}
  if [[ ! -e ${INSTALL_PATH}/eksctl ]]; then
    EKSCTL_DOWNLOAD_URL="https://github.com/weaveworks/eksctl/releases/download/v${EKSCTL_VERSION}/eksctl_$(uname -s)_amd64.tar.gz"
    curl --silent --location "${EKSCTL_DOWNLOAD_URL}" | tar xz -C "${INSTALL_PATH}"
    chmod +x "${INSTALL_PATH}"/eksctl
  fi
}

function gomplate_install() {
  INSTALL_PATH=${1}
  GOMPLATE_VERSION=${2}
  if [[ ! -e ${INSTALL_PATH}/gomplate ]]; then
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$(uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) arch=$(uname -m) ;;
    esac
    curl --silent --fail --location -o "${INSTALL_PATH}/gomplate" \
      "https://github.com/hairyhenderson/gomplate/releases/download/${GOMPLATE_VERSION}/gomplate_${os}-${arch}"
    chmod +x "${INSTALL_PATH}"/gomplate
  fi
}

function eksctl_create_cluster() {
  CLUSTER_NAME=${1}
  BIN=${2}
  GOMPLATE_BIN=${3}
  REGION=${4}
  ZONES=${5}
  INSTANCE_TYPE=${6}
  NODE_COUNT=${7}
  K8S_VERSION=${8}
  CLUSTER_FILE=${9}
  KUBECONFIG=${10}
  TEMPLATE_FILE=${11}
  CONTROLLER_ROLE_ARN=${12}
  EKSCTL_ADMIN_ROLE=${13}

  CLUSTER_NAME="${CLUSTER_NAME//./-}"

  loudecho "Templating $CLUSTER_NAME to $CLUSTER_FILE"
  CLUSTER_NAME="${CLUSTER_NAME}" \
    REGION="${REGION}" \
    ZONES="${ZONES}" \
    INSTANCE_TYPE="${INSTANCE_TYPE}" \
    NODE_COUNT="${NODE_COUNT}" \
    K8S_VERSION="${K8S_VERSION}" \
    CONTROLLER_ROLE_ARN="${CONTROLLER_ROLE_ARN}" \
    ${GOMPLATE_BIN} -f "${TEMPLATE_FILE}" -o "${CLUSTER_FILE}"

  if eksctl_cluster_exists "${CLUSTER_NAME}" "${BIN}"; then
    loudecho "Upgrading cluster $CLUSTER_NAME with $CLUSTER_FILE"
    ${BIN} upgrade cluster -f "${CLUSTER_FILE}"
  else
    loudecho "Creating cluster $CLUSTER_NAME with $CLUSTER_FILE"
    ${BIN} create cluster -f "${CLUSTER_FILE}" --kubeconfig "${KUBECONFIG}"
  fi

  loudecho "Cluster ${CLUSTER_NAME} kubecfg written to ${KUBECONFIG}"

  loudecho "Getting cluster ${CLUSTER_NAME}"
  ${BIN} get cluster "${CLUSTER_NAME}"

  if [[ -n "$EKSCTL_ADMIN_ROLE" ]]; then
    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    ADMIN_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${EKSCTL_ADMIN_ROLE}"
    loudecho "Granting ${ADMIN_ARN} admin access to the cluster"
    ${BIN} create iamidentitymapping --cluster "${CLUSTER_NAME}" --arn "${ADMIN_ARN}" --group system:masters --username admin
  fi

  return $?
}

function eksctl_cluster_exists() {
  CLUSTER_NAME=${1}
  BIN=${2}
  set +e
  if ${BIN} get cluster "${CLUSTER_NAME}"; then
    set -e
    return 0
  else
    set -e
    return 1
  fi
}

function eksctl_delete_cluster() {
  BIN=${1}
  CLUSTER_NAME=${2}
  loudecho "Deleting cluster ${CLUSTER_NAME}"
  ${BIN} delete cluster "${CLUSTER_NAME}"
}
