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

TAKO_ROOT=$(cd "$(dirname "$0")/../../.."; pwd)

if [[ $# -gt 1 ]]; then
  echo "usage: ${BASH_SOURCE} [DIRECTORY]"
  exit 1
fi

OUTPUT_DIR="$@"
if [[ -z "${OUTPUT_DIR}" ]]; then
  OUTPUT_DIR=${TAKO_ROOT}/docs/cli
fi

mkdir -p ${OUTPUT_DIR}

go run ${TAKO_ROOT}/hack/doc-gen/cli/tako.go ${OUTPUT_DIR}

# inject header for hugo

echo """---
weight: 100
title: Tako CLI Reference
---
# CLI Reference
""" | cat - ${OUTPUT_DIR}/tako.md > ${OUTPUT_DIR}/temp && mv ${OUTPUT_DIR}/temp ${OUTPUT_DIR}/tako.md


