---
weight: 50
title: Kev configuration reference
---

# Configuration

Kev leverages the Docker Compose specification to configure and prepare an application for deployment in Kubernetes.

## Project wide configuration

Project wide configuration is OPTIONAL and can be defined in one (or more) of the source compose files used to initialise the project.

It will be applied against all environments unless a specific environment overrides a setting with its own value.

## Environment configuration

Environment configuration lives in a dedicated docker compose override file. This automatically gets applied to the project's source docker compose files at the `render` phase.

Any project wide configuration found will be overridden by environment specific values.

### Component level configuration

Configuration is divided into the following groups of parameters:

* [Component](#-component)
* [Workload](#-workload)
* [Service](#-service)
* [Volumes](#-volumes)
* [Environment](#-environment)

# → Component

This configuration group contains application composition related settings. Configuration parameters can be individually defined for each application stack component.

## disabled

Defines whether a component is disabled. All application components are enabled by default.

### Default: `false`

### Possible options: `true`, `false`.

> disabled
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      disabled: true
...
```

# → Workload

This configuration group contains Kubernetes `workload` specific settings. Configuration parameters can be individually defined for each application stack component.

## workload.imagePull

Defines the docker image pull policy, and if applicable, the secret required to access the container registry.

### workload.imagePull.policy

Defines docker image pull policy from the container registry. See official K8s [documentation](https://kubernetes.io/docs/concepts/containers/images/#updating-images).

#### Default: `IfNotPresent`

#### Possible options: `IfNotPresent`, `Always`, `Never`.

> workload.image-pull-policy:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        imagePull:
          policy: IfNotPresent
...
```

### workload.imagePull.secret

Defines docker image pull secret which should be used to pull images from the container registry. See official K8s [documentation](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod).

#### Default: ""

#### Possible options: arbitrary string.

> workload.imagePull.secret:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        imagePull:
          secret: my-image-pull-secret-name
...
```

## workload.restartPolicy

Defines the restart policy for individual application component in the event of a container crash. Kev will attempt to infer that setting for each compose service defined, however in some cases manual override might be necessary. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy).

### Default: `Always`

### Possible options: `Always`, `OnFailure`, `Never`.

> workload.restartPolicy:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        restartPolicy: Always
...
```

## workload.serviceAccountName

Defines the kubernetes Service Account name to run a workload with. Useful when specific access level associated with a Service Account is required for a given workload type. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/).

### Default: `default`

### Possible options: Arbitrary string.

> workload.serviceAccountName:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        serviceAccountName: my-special-service-account-name
...
```

## workload.podSecurity

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload

### workload.podSecurity.runAsUser

This option sets up an appropriate User ID (`runAsUser` field) which specifies that for any Containers in the Pod, all processes will run with user ID as specified by the value.

#### Default: nil (not specified)

#### Possible options: arbitrary numeric UID, example `1000`.

> workload.podSecurity.runAsUser:
```yaml
version: 3.7
services:
  x-k8s:
    workload:
      podSecurity:
        runAsUser: 1000
...
```

### workload.podSecurity.runAsGroup

This option sets up an appropriate Group ID (`runAsGroup` field) which specifies the primary group ID for all processes within any containers of the Pod. If this field is omitted (currently a default), the primary group ID of the container will be root(0). Any files created will also be owned by user with specified user ID (`runAsUser` field) and group ID (`runAsGroup` field) when runAsGroup is specified.

#### Default: nil (not specified)

#### Possible options: Arbitrary numeric GID. Example `2000`.

> workload.podSecurity.runAsGroup:
```yaml
version: 3.7
services:
  my-service:
    workload:
      podSecurity:
        runAsGroup: 2000
...
```

### workload.podSecurity.fsGroup

This option is concerned with setting up a supplementary group `fsGroup` field. If specified, all processes of the container are also part of this supplementary group ID. The owner for attached volumes and any files created in those volume will be Group ID as specified by the value of this configuration option.

#### Default: nil (not specified)

#### Possible options: Arbitrary numeric GID. Example `1000`.

> workload.podSecurity.fsGroup:
```yaml
version: 3.7
services:
  my-service:
    workload:
      podSecurity:
        fsGroup: 3000
...
```

## workload.type

Defines the Kubernetes workload type controller. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/). Kev will attempt to infer workload type from the information specified in the compose file.

Kev uses the following heuristics to derive the type of workload:

If compose file(s) specifies the `deploy.mode` attribute key in a compose project service config, and it is set to "global" then `DaemonSet` workload type is assumed. Otherwise, workload type will default to `Deployment` unless volumes are in use, in which case workload will default to `StatefulSet`.

### Default: `Deployment`

### Possible options: `Pod`, `Deployment`, `StatefulSet`, `Daemonset`, `Job`.

> workload.type:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        type: StatefulSet
...
```

## workload.replicas

Defines the number of instances (replicas) for each application component. See K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#replicas). Kev will attempt to infer number of replicas type from the information specified in the compose file.

Kev uses the following heuristics to derive the number of replicas for each service:

If compose file(s) specifies the `deploy.replicas` attribute key in a project service config it will use its value.
Otherwise, number of replicas will default to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> workload.replicas:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        replicas: 1
...
```

## workload.autoscale

Enables application horizontal pod autoscaling. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)

Autoscaling assumes that a workload's number of replicas is smaller than the maximum desired number of replicas - otherwise autoscaling won't be enabled.

Then, it periodically adjusts the number of replicas to match the observed metrics such as average CPU utilisation, average memory utilisation or any other custom metric to the target max replicas specified by the user.

### workload.autoscale.maxReplicas

Defines the maximum number of instances (replicas) the application component should automatically scale up to. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account when initial number of replicas is lower than this parameter.

#### Default: `0`

#### Possible options: Arbitrary integer value. Example: `10`.

> workload.autoscale.maxReplicas:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        autoscale:
          maxReplicas: 10
      replicas: 2
...
```

### workload.autoscale.cpuThreshold

Defines the CPU utilisation threshold for the horizontal pod autoscaler for the application component. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account maximum number of replicas for the application component is defined.

#### Default: `70` (70% cpu utilization)

#### Possible options: Arbitrary integer value. Example: `80`.

> workload.autoscale.cpuThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        autoscale:
          maxReplicas: 10
          cpuThreshold: 70
        replicas: 2
...
```

### workload.autoscale.memThreshold

Defines the Memory utilization threshold for the horizontal pod autoscaler for the application component. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account maximum number of replicas for the application component is defined.

#### Default: `70` (70% memory utilization)

#### Possible options: Arbitrary integer value. Example: `80`.

> workload.autoscale.memThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        autoscale:
          maxReplicas: 10
          memThreshold: 70
        replicas: 2
...
```

## workload.rollingUpdateMaxSurge

Defines the number of pods that can be created above the desired amount of pods during an update. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#proportional-scaling). Kev will attempt to infer this number from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `deploy.update_config.parallelism` attribute key in a service config it will use its value.
Otherwise it will default to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> workload.rollingUpdateMaxSurge:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        rollingUpdateMaxSurge: 2
...
```

## workload.resource

Defines the resource share request and limits for a given workload using different parameters.

### workload.resource.cpu

Defines the CPU share request for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.cpus` attribute key in a project service config it will use its value. Otherwise it'll assume sensible default of `0.1` (equivalent of 100m in Kubernetes).

#### Default: `0.1`

#### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> workload.resource.cpu:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          cpu: 0.1
...
```

### workload.resource.maxCpu

Defines the CPU share limit for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `deploy.resources.limits.cpus` attribute key in a service config it will use its value.
Otherwise, it'll default to a sensible default of `0.5` (equivalent of 500m in Kubernetes).

#### Default: `0.5`

#### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> kev.workload.max-cpu:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          maxCpu: 2
...
```

### workload.resource.memory

Defines the Memory request for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer Memory request from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.memory` attribute key in a service config it will use its value. Otherwise it'll default to a sensible quantity of `10Mi`.

#### Default: `10Mi`

#### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> workload.resource.memory:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          memory: 200Mi
...
```

### workload.resource.maxMemory

Defines the Memory limit for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer Memory limit from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `deploy.resources.limits.memory` attribute key in a service config it will use its value.
Otherwise it'll default to a sensible quantity of `500Mi`.

#### Default: `500Mi`

#### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> workload.resource.maxMemory:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          maxMemory: 0.3Gi
...
```
### workload.resource.storage

Defines the ephemeral storage size request for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#setting-requests-and-limits-for-local-ephemeral-storage).

#### Default: `nil` (not set)

#### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> workload.resource.storage:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          storage: 200M
...
```

### workload.resource.maxStorage

Defines the ephemeral storage size limit for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#setting-requests-and-limits-for-local-ephemeral-storage).

#### Default: `nil` (not set)

#### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> workload.resource.maxStorage:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        resource:
          maxStorage: 10Gi
...
```

## workload.livenessProbe

Defines the workload's liveness probe.

### workload.livenessProbe.type

This setting defines the workload's liveness probe type. Kev will attempt to infer from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `healthcheck.disable` attribute key in a service config it will set the probe type to `none`.
Otherwise it'll default to `exec` (liveness probe active!)

#### Default: `exec`

#### Possible options: none, exec, http, tcp.

> workload.livenessProbe.type:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          type: none
...
```

### workload.livenessProbe.exec.command

Defines the liveness probe command to be run for the workload when the type is `exec`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the command from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `healthcheck.test` attribute key in a service config it will use its value.
If probe is not defined, it will prompt the user to define one by injecting a generic echo command.

#### Default: echo "<generic prompt text>"

#### Possible options: shell command

> workload.livenessProbe.exec.command
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
    workload:
      livenessProbe:
        type: exec
        exec:
          command:
            - /is-my-service-alive.sh
...
```

### workload.livenessProbe.http.port

Defines the liveness probe port to be used for the workload when the type is `http`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request).

#### Possible options: Integer

> workload.livenessProbe.http.port:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          type: http
          http:
            port: 8080
...
```

### workload.livenessProbe.http.path

Defines the liveness probe path to be used for the workload when the type is `http`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request).

#### Possible options: String

> workload.livenessProbe.http.path:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          type: http
          http:
            port: 8080
            path: /status
...
```

### workload.livenessProbe.tcp.port

Defines the liveness probe port to be used for the workload when the type is `tcp`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-tcp-liveness-probe).

#### Possible options: Integer

> workload.livenessProbe.tcp.port:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          type: tcp
          tcp:
            port: 8080
...
```

### workload.livenessProbe.failureThreshold

Defines the failure threshold (number of retries) for the workload before giving up. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command).

#### Default: `3`

#### Possible options: Arbitrary integer. Example: `5`

> workload.livenessProbe.failureThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          ...
          failureThreshold: 3
...
```

### workload.livenessProbe.successThreshold

Defines the minimum consecutive successes for the probe to be considered successful. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command).

#### Default: `1`

#### Possible options: Arbitrary integer. Example: `5`

> workload.livenessProbe.successThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          ...
          successThreshold: 1
...
```

### workload.livenessProbe.initialDelay

Defines how long to wait before the first liveness probe runs for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the wait time from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `healthcheck.start_period` attribute key in a service config it will use its value.
Otherwise, it'll default to `1m` (1 minute).

#### Default: `1m`

#### Possible options: Arbitrary time duration. Example: `1m30s`

> workload.livenessProbe.initialDelay:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          ...
          initialDelay: 2m
...
```

### workload.livenessProbe.period

Defines how often liveness probe should run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the interval from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `healthcheck.interval` attribute key in a service config it will use its value.
Otherwise, it'll default to `1m` (1 minute).

#### Default: `1m`

#### Possible options: Arbitrary time duration. Example: `1m30s`

> workload.livenessProbe.period:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          ...
          period: 1m0s
...
```

### workload.livenessProbe.timeout

Defines the timeout for the liveness probe for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the timeout value from the information specified in the compose file.

Kev uses the following heuristics to derive that information for each service:

If compose file(s) specifies the `healthcheck.timeout` attribute key in a service config it will use its value.
Otherwise, it'll default to `10s` (10 seconds).

#### Default: `10s`

#### Possible options: Arbitrary time duration. Example: `30s`

> workload.livenessProbe.timeout:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        livenessProbe:
          ...
          timeout: 30s
...
```

## workload.readinessProbe

Defines the workload's readiness probe.

### workload.readinessProbe.type

Defines the workload's readiness probe type. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `none`

#### Possible options: none, exec, http, tcp.

> workload.readinessProbe.type:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          type: none
...
```

### workload.readinessProbe.exec.command

Defines the readiness probe command to be run for the workload when the type is `exec`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: nil

#### Possible options: shell command

> workload.readinessProbe.exec.command:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          type: exec
          exec:
            command:
            - /is-my-service-ready.sh
...
```

### workload.readinessProbe.http.port

Defines the readiness probe port to be used for the workload when the type is `http`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Possible options: Integer

> workload.readinessProbe.http.port:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          type: http
          http:
            port: 8080
...
```

### workload.readinessProbe.http.path

Defines the readiness probe path to be used for the workload when the type is `http`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Possible options: String

> workload.readinessProbe.http.path:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          type: http
          http:
            port: 8080
            path: /status
...
```

### workload.readinessProbe.tcp.port

Defines the readiness probe path to be used for the workload when the type is `tcp`.
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Possible options: Integer

> workload.readinessProbe.tcp.port:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          type: tcp
          tcp:
            port: 8080
...
```

### workload.readinessProbe.period

Defines how often a readiness probe should run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `1m`

#### Possible options: Time duration

> workload.readinessProbe.period:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          ...
          period: 30s
...
```

### workload.readinessProbe.initialDelay

Defines how long to wait before the first readiness probe runs for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `1m`

#### Possible options: Arbitrary time duration. Example: `1m30s`

> workload.readinessProbe.initialDelay:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          ...
          initialDelay: 10s
...
```

### workload.readinessProbe.timeout

Defines the timeout for the readiness probe for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `10s`

#### Possible options: Arbitrary time duration. Example: `30s`

> workload.readinessProbe.timeout:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          ...
          timeout: 10s
...
```

### workload.readinessProbe.failureThreshold

Defines the failure threshold (number of retries) for the workload before giving up. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `3`

#### Possible options: Arbitrary integer. Example: `5`

> workload.readinessProbe.failureThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          ...
          failureThreshold: 3
...
```

### workload.readinessProbe.successThreshold

Defines the minimum consecutive successes for the probe to be considered successful. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

#### Default: `1`

#### Possible options: Arbitrary integer. Example: `1`

> workload.readinessProbe.successThreshold:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      workload:
        readinessProbe:
          ...
          successThreshold: 1
...
```

# → Service

The `service` group contains configuration details around Kubernetes services and how they get exposed externally.

**IMPORTANT: Only the first port for each service is processed and used to infer initial configuration!**

## service.type

Defines the type of Kubernetes service for a specific workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types).

Although Kev provides a variety of types you can use, it only tries to extract two types of services from the compose configuration, namely `None` or `ClusterIP`.

If you need a different type, please configure it manually. The different types are listed and explained below. Related official K8s

Here is the heuristic used to extract a service type:

* If compose project service publishes a port (i.e. defines a port mapping between host and container ports):
    * It will assume a `ClusterIP` service type
* If compose project service does not publish a port:
    * It will assume a `None` service type

### Default: `None` - no service will be created for the workload by default!

### Possible options: `None`, `ClusterIP`, `Nodeport`, `Headless`,  `LoadBalancer`.

These options are useful for exposing a Service either internally or externally onto an external IP address, that's outside of your cluster.

> service.type:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      service:
        type: LoadBalancer
...
```

#### None

Simply, no service will be created.

#### ClusterIP

Choosing this type makes the Service only reachable internally from within the cluster by other services. There is no external access.

In development, you can access this service on your localhost using [Port Forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/).

It is ideal for an internal service or locally testing an app before exposing it externally.

#### Nodeport

This service type is the most basic way to get external traffic directly to your service.

Its opens a specific port on each of the K8s cluster Nodes, and any traffic that is sent to this port is forwarded to the ClusterIP service which is automatically created.

You'll be able to contact the NodePort Service, from outside the cluster, by requesting `<NodeIP>:<NodePort>`.

It is ideal for running a service with limited availability, e.g. a demo app.

#### Headless

This is the same as a `ClusterIP` service type, but lacks load balancing or proxying. Allowing you to connect to a Pod directly.

Specifically, it does have a service IP, but instead of load-balancing it will return the IPs of the associated Pods.

It is ideal for scenarios such as Pod to Pod communication or clustered applications node discovery.

#### LoadBalancer

This service type is the standard way to expose a service to the internet.

All traffic on the port you specify will be forwarded to the service allowing any kind of traffic to it, e.g. HTTP, TCP, UDP, Websockets, gRPC, etc...

Again, it is ideal for exposing a service or app to the internet under a single IP address.

Practically, in non development environments, a LoadBalancer will be used to route traffic to an Ingress to expose multiple services under the same IP address and keep your costs down.

## service.nodeport

Defines the Node Port value for a Kubernetes service of type `NodePort`. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport).
NOTE: `nodeport` attributes will be ignored for any other service type!
Kev will attempt to extract that information from the compose configuration.

### Default: `nil` - no nodeport defined by default!

### Possible options: Arbitrary integer. Example `10222`.

> service.nodeport:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      service:
        type: nodeport
        nodeport: 5555
...
```

## service.expose

Defines how to expose the service externally. This detail can't be easily derived from the compose file and so in order to expose a service the user must explicitly instruct Kev to do so. By default, all component services aren't exposed i.e. have no ingress attached to them.

### service.expose.domain

#### Possible options:

- "default" - ingress will be created with Kubernetes cluster defaults.
- "domain.com,otherdomain.com..." - comma separated list of domains for the ingress.

#### Default: `""` - No ingress will be created!

> service.expose.domain:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      service:
        type: LoadBalancer
        expose:
          domain: my-awesome-service.com
...
```

### service.expose.tlsSecret

Defines whether to use TLS for the exposed service and which secret name contains certificates for the service. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls).

NOTE: This option is only relevant when service is exposed, see: [service.expose.domain](#service.expose.domain) above.

#### Default: `nil` - No TLS secret name specified by default!

#### Possible options: Arbitrary string.

> service.expose.tlsSecret:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      service:
        type: LoadBalancer
        expose:
          domain: "my-domain.com"
          tlsSecret: "my-service-tls-secret-name"
...
```

### service.expose.ingressAnnotations

Ingress annotations are used to configure some options depending on the Ingress controller. Different Ingress controller support different annotations. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource)

NOTE: This option is only relevant when service is exposed, see: [service.expose.domain](#service.expose.domain) above.

#### Possible options: map with a string and string value.

> service.expose.tlsSecret:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      service:
        type: LoadBalancer
        expose:
          domain: "my-domain.com"
          tlsSecret: "my-service-tls-secret-name"
          ingressAnnotations:
            kubernetes.io/ingress.class: external
            cert-manager.io/cluster-issuer: prod-le-dns01
...
```

# → Volumes

This configuration group contains Kubernetes persistent `volume` claim specific settings. Configuration parameters can be individually defined for each volume referenced in the project compose file(s).

## volume.storageClass

Defines the class of persistent volume. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: `""`

### Possible options: Arbitrary string.

> volume.storageClass:
```yaml
version: 3.7
volumes:
  vol1:
    x-k8s:
      storageClass: my-custom-storage-class
...
```

## volume.size

Defines the size of persistent volume. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: `1Gi`

### Possible options: Arbitrary size string. Example: `10Gi`.

> volume.size:
```yaml
version: 3.7
volumes:
  vol1:
    x-k8s:
      size: 10Gi
...
```

## volume.selector

Defines a label selector to further filter the set of volumes. Only the volumes whose labels match the selector can be bound to the PVC claim. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: `""`

### Possible options: Arbitrary string. Example: `data`.

> volume.selector:
```yaml
version: 3.7
volumes:
  vol1:
    x-k8s:
      selector: my-volume-selector
...
```

# → Environment

This group allows for application component `environment` variables configuration.

## Literal string

To set an environment variable with explicit string value

> Environment variable with as literal string:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_A: some-literal-value  # Literal value
```

When there is a need to reference any dependent environment variables it can be achieved by using double curly braces

> Environment variable with as literal string referencing dependent environment variables:
```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_A: foo
      ENV_VAR_B: bar
      ENV_VAR_C: {{ENV_VAR_A}}/{{ENV_VAR_B}}  # referencing other dependent environment variables
```

## Reference K8s secret key value

To set an environment variable with a value taken from Kubernetes secret, use the following shortcut: `secret.{secret-name}.{secret-key}`.

```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_B: secret.{secret-name}.{secret-key}  # Refer to a value stored in a secret key
```

## Reference K8s config map key value

To set an environment variable with a value taken from Kubernetes config map, use the following shortcut: `config.{config-name}.{config-key}`.

```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_C: config.{config-name}.{config-key}  # Refer to a value stored in a configmap key
```

## Reference Pod field path

To set an environment variable with a value referencing K8s Pod field value, use the following shortcut: `pod.{field-path}`.

```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_D: pod.{field-path} # Refer to the a value of the K8s workload Pod field path
                                  # e.g. `pod.metadata.namespace` to get the k8s namespace
                                  # name in which pod operates

```

### Supported `pod.{...}` field paths:
* `metadata.name` - returns current app component K8s Pod name
* `metadata.namespace` - returns current app component K8s namespace name in which Pod operates
* `metadata.labels` - return current app component labels
* `metadata.annotations` - returns current app component annotations
* `spec.nodeName` - returns current app component K8s cluster node name
* `spec.serviceAccountName` - returns current app component K8s service account name with which Pod runs
* `status.hostIP` - returns current app component K8s cluster Node IP address
* `status.podIP` - returns current app component K8s Pod IP address

## Reference Container resource field

To set an environment variable with a value referencing K8s Container resource field value, use the following shortcut: `container.{name}.{....}`.

```yaml
version: 3.7
services:
  my-service:
    x-k8s:
      ...
    environment:
      ENV_VAR_E: container.{container-name}.{resource-field} # Refer to the a value of the K8s workload Container resource field
                                                             # e.g `limits.cpu` to get max CPU allocatable to the container

```

### Supported `container.{name}.{....}` resource fields:
* `limits.cpu`, `limits.memory`, `limits.ephemeral-storage` - return value of selected container `limit` field
* `requests.cpu`, `requests.memory`, `requests.ephemeral-storage` - return value of selected container `requests` field
