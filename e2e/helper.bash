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

setup() {
  export TMP="$BATS_TEST_DIRNAME/../tmp"
  export E2E_NS="e2e-$(uuidgen | tr "[:upper:]" "[:lower:]")"
  create-namespace
  mkdir -p "$TMP/k8s"
  cd $BATS_TEST_DIRNAME
}

teardown() {
  [ -f "$BATS_TEST_DIRNAME/kev.yaml" ] && rm -f "$BATS_TEST_DIRNAME/kev.yaml"
  [ -f "$BATS_TEST_DIRNAME/docker-compose.kev.$E2E_KEV_ENV.yaml" ] && rm -rf "$BATS_TEST_DIRNAME/docker-compose.kev.$E2E_KEV_ENV.yaml"
  [ -d "$TMP/k8s" ] && rm -rf "$TMP/k8s"
  cd -
}

create-namespace(){
 $KUBECTL create namespace $E2E_NS
}

generate-manifests() {
  kev init -e $E2E_KEV_ENV && kev render -d "$TMP/k8s" -e $E2E_KEV_ENV
}

apply-manifests() {
  $KUBECTL -n $E2E_NS apply -f "$TMP/k8s/$E2E_KEV_ENV"
}

gen-apply-manifests() {
  generate-manifests
  apply-manifests
}

# wait-on-deployment is responsible for waiting for a deployment to deploy
wait-on-deployment() {
  local namespace=$1
  local labels=$2

  for ((i=0; i<=60; i++)); do
    if eval "$KUBECTL -n ${namespace} get po -l ${labels} --field-selector=status.phase=Running --no-headers | grep -i running"; then
      return 0
    else
      echo "failed to run command: retrying (attempt/max = ${i}/60)"
      sleep 3
    fi
  done

  return 1
}

ensure-deployment-type(){
  local namespace=$1
  local labels=$2
  local type=$3

  $KUBECTL -n ${namespace} describe po -l ${labels} | grep -i ${type}
}

ensure-volume(){
  local namespace=$1
  local name=$2

  $KUBECTL -n ${namespace} get persistentvolume --no-headers | grep -i ${name}
}

ensure-service(){
  local namespace=$1
  local label=$2

  $KUBECTL -n ${namespace} get service -o wide --no-headers | grep -i ${label}
}

ensure-service-type(){
  local namespace=$1
  local label=$2
  local type=$3

  $KUBECTL -n ${namespace} get service -l ${label} --no-headers | grep -i ${type}
}

check-app-is-running(){
  local namespace=$1
  local label=$2
  local port=$3

  for ((i=0; i<=60; i++)); do
    if eval "$KUBECTL -n ${namespace} exec $($KUBECTL get pod -n "${namespace}" -l "${label}" -o name) --  curl -sLI localhost:${port}" ; then
      return 0
    else
      sleep 3
    fi
  done

  return 1
}
