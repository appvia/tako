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

KEV_ROOT=$(cd "$(dirname "$0")/../../.."; pwd)

if [[ $# -gt 1 ]]; then
  echo "usage: ${BASH_SOURCE} [DIRECTORY]"
  exit 1
fi

STATIC_WEB_GENERATOR_DIR=${KEV_ROOT}/hack/doc-gen/static-web

OUTPUT_DIR="$@"
if [[ -z "${OUTPUT_DIR}" ]]; then
  OUTPUT_DIR=${STATIC_WEB_GENERATOR_DIR}/public
fi

mkdir -p ${OUTPUT_DIR}

# NOTE: Hugo fails to generate static assets (css) in production environment
#       but generate it properly when run in server mode locally (development).
#       Starting local hugo server will also generate public site content
#       and store in ./public directory which is gitignored!

hugo server -c ${KEV_ROOT}/docs -d ${OUTPUT_DIR} --cleanDestinationDir

# -s ${STATIC_WEB_GENERATOR_DIR}
