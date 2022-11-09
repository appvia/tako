---
weight: 51
title: Configuration example
---

# Config example

```yaml
version: "3.7"

services:                                                # compose services section
  wordpress:                                             # compose project service name
    x-k8s:                                               # compose K8s configuration extension
      disabled: false                                    # When disabled the compose service won't be included in generated K8s manifests. Defaults to 'false'.
      service:                                           # K8s service configuration (only required if values are overridden)
        type: None                                       # Default: none (no service). Possible options: none | headless | clusterip | nodeport | loadbalancer.
        nodeport:                                        # Default: nil. Set with numeric value e.g. 5555. Only taken into account when working with service.type: nodeport
        expose:                                          # K8s configuration to expose a service externally (by default services are never exposed)
          domainPrefix:                                  # Default: "" (no prefix). If defined it'll prefix exposed domain name.
          domain:                                        # Default: "" (no ingress). Possible options: "" | "default" | domain.com,otherdomain.com (comma separated domain names). When with "default" or domain name(s) - it'll generate an ingress object and expose service externally.
          tlsSecret:                                     # Default: "" (no tls). Kubernetes secret name where certs will be loaded from.
      workload:                                          # K8s workload configuration (only required if values are overridden)
        command:                                         # Default: nil. When defined it'll override start command defined in the container.
          - /bin/sh
        commandArgs:                                     # Default: nil. When defined it'll override command arguments defined in the container.
          - -c
          - /do/something
        annotations:                                     # Default: nil. A key/value map to attach metadata to a K8s Pod Spec in a deployable object, e.g. Deployment, StatefulSet, etc...
          key-one: value one
          key-two: value two
        autoscale:                                       # Configures an application for auto-scaling.
          maxReplicas: 0                                 # Default: 0. Number of replicas to autoscale to.
          cpuThreshold: 70                               # Default: 70. The CPU utilisation threshold.
          memThreshold: 70                               # Default: 70. The Memory utilization threshold.
        imagePull:                                       # The docker image pull policy.
          policy: IfNotPresent                           # Default: IfNotPresent. Possible options: IfNotPresent / Always.
          secret: ""                                     # Default: "" (no secret). Docker image pull secret to pull images from the container registry.
        livenessProbe:                                   # Workload's liveness probe
          ### EXEC
          type: exec                                     # Default: exec. Possible options: none | exec | http | tcp. See examples for other probe types in commented sections below.
          exec:                                          # The exec command matching the liveness probe type.
            command:                                     # Liveness probe command to run.
              - echo
              - Define healthcheck command for service
          ### HTTP
          type: http                                     # HTTP Liveness probe type.
          http:
            port: 8080                                   # HTTP Liveness probe port. Only used when using an http probe type.
            path: /status                                # HTTP Liveness probe path. Only used when using an http probe type.
          ### TCP
          type: tcp                                      # TCP Liveness probe type.
          tcp:
            port: 8080                                   # TCP Liveness probe port. Only used when using an tcp probe type.
          failureThreshold: 3                            # Default: 3. The failure threshold (number of retries) for the workload before giving up. Shared by all liveness probe types.
          successThreshold: 1                            # Default: 1. Minimum consecutive successes for the probe to be considered successful. Shared by all liveness probe types.
          initialDelay: 1m0s                             # Default: 1m. How long to wait before initial probe run. Shared by all liveness probe types.
          period: 1m0s                                   # Default: 1m. How often liveness probe should run for the workload. Shared by all liveness probe types.
          timeout: 10s                                   # Default: 10s. Probe timeout. Shared by all liveness probe types.
        readinessProbe:                                  # Workload's readiness probe
          # EXEC
          type: exec                                     # Default: exec. Possible options: none | exec | http | tcp. See examples for other probe types in commented sections below.
          exec:                                          # The exec command matching the readiness probe type.
            command:                                     # Readiness probe command to run.
              - echo
              - not applicable
          ### HTTP
          type: http                                     # HTTP readiness probe type.
          http:
            port: 8080                                   # HTTP readiness probe port. Only used when using an http probe type.
            path: /status                                # HTTP readiness probe path. Only used when using an http probe type.
          ### TCP
          type: tcp                                      # TCP readiness probe type.
          tcp:
            port: 8080                                   # TCP readiness probe port. Only used when using an tcp probe type.
          failureThreshold: 3                            # Default: 3. The failure threshold (number of retries) for the workload before giving up. Shared by all readiness probe types.
          successThreshold: 1                            # Default: 1. Minimum consecutive successes for the probe to be considered successful. Shared by all readiness probe types.
          initialDelay: 1m0s                             # Default: 1m. How long to wait before initial probe run. Shared by all readiness probe types.
          period: 1m0s                                   # Default: 1m. How often readiness probe should run for the workload. Shared by all readiness probe types.
          timeout: 10s                                   # Default: 10s. Probe timeout. Shared by all readiness probe types.
        replicas: 25                                     # Default: 1. Number of replicas per workload.
        resource:                                        # Defines resource requests and limits
          cpu: "0.1"                                     # Default: 0.1. CPU request per workload.
          maxCpu: "0.5"                                  # Default: 0.5. CPU limit per workload.
          memory: 10Mi                                   # Default: 10Mi. Memory request per workload.
          maxMemory: 500Mi                               # Default: 500Mi. Memory limit per workload.
          storage: 100Mi                                 # Default: not set. Ephemeral storage size request per workload.
          maxStorage: 500Mi                              # Default: not set. Ephemeral storage size limit per workload.
        restartPolicy: Always                            # Default: Always. Possible options: Always / OnFailure / Never.
        rollingUpdateMaxSurge: 1                         # Default: 1. Maximum number of containers to be updated at a time.
        serviceAccountName: default                      # Default: default. Service account name to be used.
        type: Deployment                                 # Default: Deployment. Possible options: Pod | Deployment | StatefulSet | Daemonset | Job.
    environment:                                         # App component environment variable overrides
      ENV_VAR_A: secret.{secret-name}.{secret-key}       # Refer to the a value stored in a secret key
      ENV_VAR_B: config.{config-name}.{config-key}       # Refer to the a value stored in a configmap key
      ENV_VAR_C: literal-value                           # Use literal value

volumes:                                                 # compose volumes section
  db_data:                                               # volume name
    x-k8s:                                               # configuration labels
      size: 100Mi                                        # Defines volume size
      selector: my-selector                              # Defines volume selector
      storage-class: standard                            # Defines volume storage class
```
