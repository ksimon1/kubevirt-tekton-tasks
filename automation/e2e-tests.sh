#!/usr/bin/env bash

set -ex

export DEV_MODE="${DEV_MODE:-false}"
export STORAGE_CLASS="${STORAGE_CLASS:-}"
export DEPLOY_NAMESPACE="${DEPLOY_NAMESPACE:-e2e-tests-$(shuf -i10000-99999 -n1)}"
export NUM_NODES=${NUM_NODES:-2}

# See scripts/common.sh for IMAGE env variable names
./automation/set-crio-permissions-command.sh
./automation/e2e-deploy-resources.sh

kubectl get namespaces -o name | grep -Eq "^namespace/$DEPLOY_NAMESPACE$" || kubectl create namespace "$DEPLOY_NAMESPACE"

if [[ "$DEV_MODE" == "true" ]]; then
  make cluster-sync
else
  make deploy
fi

make cluster-test
