#!/usr/bin/env bash

set -e

if [ -z "${IMG_TAG}" ]; then
  echo "IMG_TAG is not defined"
  exit 1
fi

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"

source "${SCRIPT_DIR}/release-var.sh"
source "${SCRIPT_DIR}/common.sh"

for TASK_NAME in ${TASK_NAMES[*]}; do
    if echo "${TASK_NAME}" | grep -vqE "^(${EXCLUDED_NON_IMAGE_MODULES})$"; then
    if [ ! -d  "${TASK_NAME}" ]; then
        continue
    fi
    visit "${TASK_NAME}"
        IMAGE_NAME_AND_TAG="tekton-task-${TASK_NAME}:${IMG_TAG}"
        export IMAGE="${REGISTRY}/${REPOSITORY}/${IMAGE_NAME_AND_TAG}"
        podman push "${IMAGE}" --tls-verify=false
    leave
    fi
done
