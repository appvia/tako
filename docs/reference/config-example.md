---
weight: 12
title: Kev configuration example
---

# Config example

```yaml
version: "3.7"

services:                                                # compose services section
  wordpress:                                             # compose project service name
    labels:                                              # configuration labels
      kev.service.type: None                             # Default: none (no service). Possible options: none | headless | clusterip | nodeport | loadbalancer.
      kev.service.nodeport.port: ""                      # Default: "". Only taken into account when working with service.type: nodeport
      kev.service.expose: ""                             # Default: "" (no ingress). Possible options: "" | true | domain.com,otherdomain.com (comma separated domain names). When true / domain(s) - it'll set ingress object.
      kev.service.expose.tls-secret: ""                  # Default: "" (no tls). Secret name where certs will be loaded from.
      kev.workload.type: Deployment                      # Default: deployment. Possible options: pod | deployment | statefulset | daemonset | job.
      kev.workload.image-pull-policy: IfNotPresent       # Default: IfNotPresent. Possible options: IfNotPresent / Always.
      kev.workload.restart-policy: Always                # Default: Always. Possible options: Always / OnFailure / Never.
      kev.workload.replicas: "1"                         # Default: 1. Number of replicas per workload.
      kev.workload.rolling-update-max-surge: "1"         # Default: 1. Maximum number of containers to be updated at a time.
      kev.workload.service-account-name: default         # Default: default. Service account to be used.
      kev.workload.cpu: "0.1"                            # Default: 0.1. CPU request per workload.
      kev.workload.max-cpu: "0.2"                        # Default: 0.2. CPU limit per workload.
      kev.workload.memory: 10Mi                          # Default: 10Mi. Memory request per workload.
      kev.workload.max-memory: 500Mi                     # Default: 500Mi. Memory limit per workload.
      kev.workload.liveness-probe-disabled: "false"      # Default: false. Disable/Enable liveness probe.
      kev.workload.liveness-probe-command: "echo 'n/a'"  # Liveness probe command to run.
      kev.workload.liveness-probe-initial-delay: 1m0s    # Default: 1m. How long to wait before initial probe run.
      kev.workload.liveness-probe-interval: 1m0s         # Default: 1m. Interval for the probe.
      kev.workload.liveness-probe-retries: "3"           # Default: 3. Number of probe retires.
      kev.workload.liveness-probe-timeout: 10s           # Default: 10s. Probe command timeout.
    environment:                                         # App component environment variable overrides
      ENV_VAR_A: secret.{secret-name}.{secret-key}       # Refer to the a value stored in a secret key
      ENV_VAR_B: config.{config-name}.{config-key}       # Refer to the a value stored in a configmap key
      ENV_VAR_C: literal-value                           # Use literal value

volumes:                                                 # compose volumes section
  db_data:                                               # volume name
    labels:                                              # configuration labels
      kev.volume.size: 100Mi                             # Defines volume size
      kev.volume.selector: my-selector                   # Defines volume selector
      kev.volume.storage-class: standard                 # Defines volume storage class
```
