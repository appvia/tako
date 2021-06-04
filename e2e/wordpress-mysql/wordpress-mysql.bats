#!/usr/bin/env bats
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

load ../helper

@test "ensure wordpress mysql runs on k8s" {
  run gen-apply-manifests
  echo $output
  [ "$status" -eq 0 ]

  run wait-on-deployment "$E2E_NS" "service=db"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-deployment-type "$E2E_NS" "service=db" "statefulset"
  echo $output
  [ "$status" -eq 0 ]

  run wait-on-deployment "$E2E_NS" "service=wordpress"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-volume "$E2E_NS" "db-data"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-service "$E2E_NS" "service=db"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-service-type "$E2E_NS" "service=db" "clusterip"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-service "$E2E_NS" "service=wordpress"
  echo $output
  [ "$status" -eq 0 ]

  run ensure-service-type "$E2E_NS" "service=wordpress" "clusterip"
  echo $output
  [ "$status" -eq 0 ]

  run check-app-is-running "$E2E_NS" "service=wordpress" 80
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "200 OK" ]]
}
