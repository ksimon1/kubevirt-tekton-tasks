#!/usr/bin/env bash

set -ex

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
REPO_DIR="$(realpath "${SCRIPT_DIR}/..")"

source "${SCRIPT_DIR}/common.sh"

IMAGE_REGISTRY=""
IN_CLUSTER_IMAGE_REGISTRY=""

if [[ "${IS_OKD}" == "true" ]]; then
  oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
  IMAGE_REGISTRY="$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')"
  IN_CLUSTER_IMAGE_REGISTRY="image-registry.openshift-image-registry.svc:5000"
  # wait for the route
  sleep 5

  podman login -u kubeadmin -p "$(oc whoami -t)" --tls-verify=false "$IMAGE_REGISTRY"
elif [[ "${IS_MINIKUBE}" == "true" ]]; then
  if ! minikube addons list | grep -q "registry .*enabled"; then
     echo "minikube should have registry addon enabled" >&2
     exit 2
  fi
  IMAGE_REGISTRY="$(minikube ip):5000"
  IN_CLUSTER_IMAGE_REGISTRY="$(kubectl get service registry -n kube-system --output 'jsonpath={.spec.clusterIP}')"
else
  echo "only minikube or OKD is supported" >&2
  exit 3
fi

IMAGE_NAME_AND_TAG="tekton-tasks:latest"
echo ${DEPLOY_NAMESPACE}
export IMAGE="${IMAGE_REGISTRY}/${DEPLOY_NAMESPACE}/${IMAGE_NAME_AND_TAG}"
podman build -f "build/Containerfile" -t "${IMAGE}" .
podman push "${IMAGE}" --tls-verify=false

# set inside-cluster registry
export IMAGE="${IN_CLUSTER_IMAGE_REGISTRY}/${DEPLOY_NAMESPACE}/${IMAGE_NAME_AND_TAG}"
export TEKTON_TASKS_IMAGE="${IMAGE}"

IMAGE_NAME_AND_TAG="tekton-tasks-disk-virt:latest"
export IMAGE="${IMAGE_REGISTRY}/${DEPLOY_NAMESPACE}/${IMAGE_NAME_AND_TAG}"
podman build -f "build/Containerfile.DiskVirt" -t "${IMAGE}" .
podman push "${IMAGE}" --tls-verify=false

# set inside-cluster registry
export IMAGE="${IN_CLUSTER_IMAGE_REGISTRY}/${DEPLOY_NAMESPACE}/${IMAGE_NAME_AND_TAG}"
export TEKTON_TASKS_DISK_VIRT_IMAGE="${IMAGE}"

"${REPO_DIR}/scripts/deploy-tasks.sh" "$@"
