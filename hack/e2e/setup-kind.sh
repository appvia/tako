#!/bin/bash
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

# standard bash error handling
set -o errexit
set -o pipefail
set -o nounset
# debug commands
set -x

# bin dir to install binaries
BIN_DIR=${PWD}/bin
# kind binary will be here
KIND="${BIN_DIR}/kind"

# util to install a released kind version into ${BIN_DIR}
install_kind_release() {
  VERSION="v0.26.0"
  KIND_BINARY_URL="https://github.com/kubernetes-sigs/kind/releases/download/${VERSION}/kind-linux-amd64"
  mkdir -p "${BIN_DIR}"
  curl --request GET -sL \
    --url "${KIND_BINARY_URL}" \
    --output "${KIND}"
  chmod +x "${KIND}"
}

main() {
  # get kind
  install_kind_release
}
main
