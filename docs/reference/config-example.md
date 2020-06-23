---
weight: 12
title: Kev configuration example
---

# Config example

```yaml
name: hello-world-app                    # Application name
description: hello world app.            # Application description
workload:                                # Defines app default Kubernetes workload parameters.
    # Available workload parameters and their default values.
    image-pull-policy: IfNotPresent      # Default: IfNotPresent. Possible options: IfNotPresent / Always.
    image-pull-secret: nil               # Default: nil - do not use private registry pull secret.
    restart: Always                      # Default: Always. Possible options: Always / OnFailure / Never.
    service-account-name: default        # Default: default.
    security-context-run-as-user: nil    # Default: nil - no default value.
    security-context-run-as-group: nil   # Default: nil - no default value.
    security-context-fs-group: nil       # Default: nil - no default value.
    type: deployment                     # Default: deployment. Possible option: pod | deployment | statefulset | daemonset | job.
    replicas: 1                          # Default: 1. Number of replicas per workload.
    rolling-update-max-surge: 1          # Default: 1. Maximum number of containers to be updated at a time.
    cpu: 0.1                             # Default: 0.1. CPU request per workload.
    memory: 0.1                          # Default: 0.1. Memory request per workload.
    max-cpu: 0.2                         # Default: 0.2. CPU limit per workload.
    max-memory: 0.2                      # Default: 0.2. Memory limit per workload.
    liveness-probe-disable: false        # Default: false. Disable/Enable liveness probe.
    liveness-probe-command: "echo 'n/a'" # Default: "echo 'n/a'".
    liveness-probe-interval: 1m          # Default: 1m. Interval for the probe.
    liveness-probe-retries: 3            # Default: 3. Number of probe retires.
    liveness-probe-initial-delay: 1m     # Default: 1m. How long to wait before initial probe run.
    liveness-probe-timeout: 10s          # Default: 10s. Probe command timeout.
service:                                 # Defines app default component K8s service parameters.
    # Available service parameters and their default values.
    type: none                           # Default: none (no service). Possible options: none | headless | clusterip | nodeport | loadbalancer.
    nodeport: nil                        # Default: nil. Only taken into account when working with service.type: nodeport
    expose: false                        # Default: false (no ingress). Possible options: false | true | domain.com,otherdomain.com (comma
                                         #          separated domain names). When true / domain(s) - it'll set ingress object.
    tls-secret: nil                      # Default: nil (no tls). Secret name where certs will be loaded from.
volumes:                                 # Control volumes defined in compose file by specifing storage class and size.
  vol-1:                                 # Each volume defined in compose.yaml must be parametrised
    class: ssd                           # Defines volume storage class
    size: 10Gi                           # Defiens volume size

service-a:                               # Maps to compose service name
    workload:                            # Service level overrides. See Application level workload settings options above
    service:                             # Service level overrides. See Application level service settings options above
    environment:                         # App component environment settings
        ENV_VAR_A: secret.{secret-name}.{secret-key} # Refer to the a value stored in a secret key
        ENV_VAR_B: config.{config-name}.{config-key} # Refer to the a value stored in a configmap key
        ENV_VAR_C: literal-value                     # Use literal value
```
