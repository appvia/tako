#!/bin/bash -eu
#
# Copyright 2020 Appvia Ltd <info@appvia.io>
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
#

## Set the defaults
BUILD_CLI=true
CLUSTER_NAME="cluster-$(uuidgen | tr "[:upper:]" "[:lower:]")"

# Make this pretty
export NC='\e[0m'
export GREEN='\e[0;32m'
export YELLOW='\e[0;33m'
export RED='\e[0;31m'
export PATH=${PATH}:${PWD}/bin
export KUBECTL="kubectl"
export E2E_KEV_ENV='e2e'
export E2E_KUBECONFIG="${PWD}/hack/e2e/kubeconfig"

log() { (printf 2>/dev/null "$@${NC}\n"); }
announce() { log "${GREEN}[$(date +"%T")] [INFO] $@"; }
failed() { log "${YELLOW}[$(date +"%T")] [FAIL] $@"; }

usage() {
  cat <<EOF
  Usage: $(basename $0)
  --build-cli    <bool>    : indicates should build the kev cli (defaults: ${BUILD_CLI})
  -h|--help                : display this usage menu
EOF
  if [[ -n $@ ]]; then
    echo "[error] $*"
    exit 1
  fi
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  --build-cli)
    BUILD_CLI=${2}
    shift 2
    ;;
  -h | --help) usage ;;
  *) shift 1 ;;
  esac
done

can-run-e2e(){
  if ! command -v kind &> /dev/null
  then
    failed "Kind could not be found"
    exit 1
  fi

  if ! command -v bats &> /dev/null
  then
    failed "bats (Bash Automated Testing System) could not be found"
    exit 1
  fi
}

build-cli() {
  if [[ ${BUILD_CLI} == true ]]; then
    announce "Building Kev"
    make build
  fi
}

create-cluster() {
  announce "Provisioning kind cluster"
  kind create cluster --name "${CLUSTER_NAME}" --kubeconfig hack/e2e/kubeconfig
}

configure-kubeconfig(){
  export KUBECONFIG_SAVED="${KUBECONFIG:-''}"
  export KUBECONFIG=$E2E_KUBECONFIG
}

run-tests() {
  announce "Running e2e tests"
  bats e2e/**/*.bats
}

finally() {
  announce "Removing kind cluster"
  kind delete cluster --name "${CLUSTER_NAME}" --kubeconfig hack/e2e/kubeconfig

  announce "Reset KUBECONFIG"
  if [[ -z "${KUBECONFIG_SAVED}" ]]; then
    unset KUBECONFIG
  else
    export KUBECONFIG=$KUBECONFIG_SAVED
  fi
}

can-run-e2e
build-cli
create-cluster
configure-kubeconfig
run-tests || {
  finally
  exit 1
}
finally
