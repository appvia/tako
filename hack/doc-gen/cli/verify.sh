#!/bin/bash -e
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

TAKO_ROOT=$(realpath $(dirname ${BASH_SOURCE})/../../..)
DOC_GEN_DIR=$(dirname "${BASH_SOURCE}")
DOCS_DIR=${TAKO_ROOT}/docs/cli
TMP_DIR="$(mktemp -d)"

trap cleanup INT TERM HUP EXIT

cleanup() {
  rm -rf ${TMP_DIR}
}

${DOC_GEN_DIR}/generate.sh ${TMP_DIR}

exclude_file="README.md"
output=$(echo "`diff -r ${DOCS_DIR} ${TMP_DIR}`" | sed "/${exclude_file}/d")

if [[ -n "${output}" ]] ; then
    echo "FAILURE: verification of docs failed:"
    echo "${output}"
    exit 1
fi
