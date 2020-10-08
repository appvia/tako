#!/usr/bin/env bats

load ../helper

@test "ensure wordpress mysql runs on k8s" {
  run gen-apply-manifests
  echo $output
  [ "$status" -eq 0 ]

  wait-on-deployment "$E2E_NS" "io.kev.service=db"
  echo $output
  [ "$status" -eq 0 ]

  ensure-deployment-type "$E2E_NS" "io.kev.service=db" "statefulset"
  echo $output
  [ "$status" -eq 0 ]

  wait-on-deployment "$E2E_NS" "io.kev.service=wordpress"
  echo $output
  [ "$status" -eq 0 ]

  ensure-volume "$E2E_NS" "db-data"
  echo $output
  [ "$status" -eq 0 ]

  ensure-service "$E2E_NS" "io.kev.service=db"
  echo $output
  [ "$status" -eq 0 ]

  ensure-service-type "$E2E_NS" "io.kev.service=db" "clusterip"
  echo $output
  [ "$status" -eq 0 ]

  ensure-service "$E2E_NS" "io.kev.service=wordpress"
  echo $output
  [ "$status" -eq 0 ]

  ensure-service-type "$E2E_NS" "io.kev.service=wordpress" "loadbalancer"
  echo $output
  [ "$status" -eq 0 ]
}
