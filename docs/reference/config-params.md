---
weight: 11
title: Kev configuration reference
---

# Configuration

Kev configuration consists of the following components:

### Application level configuration

Settings defined in the application level configuration will be applied to all application components unless they are overridden on individual component level. See examples below.

### Component level configuration

Settings defined on component level will always take presedence over the "global" application configuration. See example below on how to override specific settings.

Configuration is divided into the following sections:
* [Workload](#workload)
* [Service](#service)
* [Volumes](#volumes)
* [Environment](#environment)

# Workload

This configuration section contains Kubernetes workload specific settings. `workload` configuration block may be defined globally for the entire application as top level configuration attribute, or within individual app component.

## image-pull-policy

Defines docker image pull policy from the registry. This setting can be set on application level in top level `workload` block, or on individual application component level, or both. Application level setting takes precedence over the global setting.

### Default: `IfNotPresent`

### Possible options: `IfNotPresent`, `Always`.

> image-pull-policy:
```yaml
name: hello-world-app
description: hello world app.
workload:
  image-pull-policy: IfNotPresent # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    image-pull-policy: Always     # App component (service) specific. Takes presedence over app level workload setting!
...
```

## image-pull-secret

Defines docker image pull policy from the registry. This setting can be set on application level in top level `workload` block, or on individual application component level, or both. Application level setting takes precedence over the global setting.

### Default: `IfNotPresent`

### Possible options: `IfNotPresent`, `Always`.

> image-pull-secret:
```yaml
name: hello-world-app
workload:
  image-pull-secret: app-wide-secret-name # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    image-pull-secret: my-secret-name     # App component (service) specific. Takes presedence over app level workload setting!
...
```

## restart

Defines the restart policy for individual application components in the event of a container crash. Kev will do its best to try and infer that setting for each compose service defined, however in some cases manual override might be necessary.

### Default: `Always`

### Possible options: `Always`, `OnFailure`, `Never`.

> image-pull-secret:
```yaml
name: hello-world-app
workload:
  restart: OnFailure           # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    image-pull-secret: Never   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## service-account-name

Defines the kubernetes Service Account name to run a workload with. Sometimes it may be necessary to run a particular Kubernetes workload with a specialised access level and this setting allow to run workloads with only level of access they require.

### Default: `default`

### Possible options: Arbitrary string.

> service-account-name:
```yaml
name: hello-world-app
workload:
  restart: default                            # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    image-pull-secret: special-svc-acc-name   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## security-context-run-as-user

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload.
This option is concerned with setting up an appropriate User ID (`runAsUser` field) which specifies that for any Containers in the Pod, all processes will run with user ID as specified by the value.

### Default: nil (not specified)

### Possible options: arbitrary numeric UID, example `1000`.

> security-context-run-as-user:
```yaml
name: hello-world-app
workload:
  security-context-run-as-user: 1000     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    security-context-run-as-user: 2000   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## security-context-run-as-group

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload.
This option is concerned with setting up an appropriate Group ID (`runAsGroup` field) which specifies the primary group ID for all processes within any containers of the Pod. If this field is omitted (currently a default), the primary group ID of the containers will be root(0). Any files created will also be owned by user with specified user ID (`runAsUser` field) and group ID (`runAsGroup` field) when runAsGroup is specified.

### Default: nil (not specified)

### Possible options: Arbitrary numeric GID. Example `1000`.

> security-context-run-as-group:
```yaml
name: hello-world-app
workload:
  security-context-run-as-group: 1000     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    security-context-run-as-group: 2000   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## security-context-fs-group

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload.
This option is concerned with setting up a supplementary group `fsGroup` field. If specified, all processes of the container are also part of this supplementary group ID. The owner for attached volumes and any files created in those volume will be Group ID as specified by the value of this configuration option.

### Default: nil (not specified)

### Possible options: Arbitrary numeric GID. Example `1000`.

> security-context-fs-group:
```yaml
name: hello-world-app
workload:
  security-context-fs-group: 1000     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    security-context-fs-group: 2000   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## type

Defines the Kubernetes workload type. Kev will attempt to infer workload type from the information specified in the compose file.

Kev uses the following heuristics to derieve the type of workload:

If compose file(s) specifies the `deploy.mode` attribute key in a service config, and it is set to "global" then Kev will assume `DaemonSet` workload type. Otherwise, workload type will default to `Deployment` unless volumes are in use in which case workload will be set as `StatefulSet`.

### Default: `Deployment`

### Possible options: `Pod`, `Deployment`, `StatefulSet`, `Daemonset`, `Job`.

> type:
```yaml
name: hello-world-app
workload:
  type: Deployment      # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    type: StatefulSet   # Kev will attempt to guess the value of this setting!
                        # App component (service) specific. Takes presedence over app level workload setting!
...
```

## replicas

Defines the number of instances for each application component. Kev will attempt to infer number of replicas type from the information specified in the compose file.

Kev uses the following heuristics to derieve the number of replicas for each service:

If compose file(s) specifies the `deploy.replicas` attribute key in a service config it will use its value.
Otherwise, number of replicas will default to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> replicas:
```yaml
name: hello-world-app
workload:
  replicas: 1     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    replicas: 10  # Kev will attempt to guess the value of this setting!
                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

## rolling-update-max-surge

Defines the number of pods that can be created above the desired amount of pods during an update.
Kev will attempt to infer this number from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.update_config.parallelism` attribute key in a service config it will use its value.
Otherwise defaults to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> rolling-update-max-surge:
```yaml
name: hello-world-app
workload:
  rolling-update-max-surge: 1     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    rolling-update-max-surge: 10  # Kev will attempt to guess the value of this setting!
                                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

## cpu

Defines the CPU share request for a given workload.
Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.cpus` attribute key in a service config it will use its value.
Defaults to `100m`.

### Default: `100m`

### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> cpu:
```yaml
name: hello-world-app
workload:
  cpu: 100m   # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    cpu: 1    # Kev will attempt to guess the value of this setting!
              # App component (service) specific. Takes presedence over app level workload setting!
...
```

## max-cpu

Defines the CPU share limit for a given workload.
Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.limits.cpus` attribute key in a service config it will use its value.
Defaults to `200m`.

### Default: `200m`

### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> max-cpu:
```yaml
name: hello-world-app
workload:
  max-cpu: 100m   # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    max-cpu: 2    # Kev will attempt to guess the value of this setting!
                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

## memory

Defines the Memory request for a given workload.
Kev will attempt to infer Memory request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.memory` attribute key in a service config it will use its value.
Defaults to `10Mi`.

### Default: `10Mi`

### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> memory:
```yaml
name: hello-world-app
workload:
  memory: 10Mi     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    memory: 200Mi  # Kev will attempt to guess the value of this setting!
                   # App component (service) specific. Takes presedence over app level workload setting!
...
```

## max-memory

Defines the Memory limit for a given workload.
Kev will attempt to infer Memory limit from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.limits.memory` attribute key in a service config it will use its value.
Defaults to `200m`.

### Default: `200m`

### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> max-memory:
```yaml
name: hello-world-app
workload:
  max-memory: 200m    # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    max-memory: 0.5Gi # Kev will attempt to guess the value of this setting!
                      # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-disable

Defines whether workload should have a liveness probe enabled.
Kev will attempt to infer from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.disable` attribute key in a service config it will use its value.
Defaults to `false`.

### Default: `false`

### Possible options: Bool

> liveness-probe-disable:
```yaml
name: hello-world-app
workload:
  liveness-probe-disable: false   # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-disable: true  # Kev will attempt to guess the value of this setting!
                                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-command

Defines the liveness probe command to be run for the workload.
Kev will attempt to infer the command from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.test` attribute key in a service config it will use its value.
If probe is not defined it will prompt the user to define one.

### Default: echo "prompt user to define the probe"

### Possible options: shell command

> liveness-probe-command:
```yaml
name: hello-world-app
workload:
  liveness-probe-command: ["echo", "n/a"]  # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-command: ["/is-my-service-alive.sh"] # Kev will attempt to guess the value of this setting!
                                                        # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-interval

Defines how often liveness proble should run for the workload.
Kev will attempt to infer the interval from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.interval` attribute key in a service config it will use its value.
Defaults to `1m` (1 minute).

### Default: `1m`

### Possible options: Time duration

> liveness-probe-interval:
```yaml
name: hello-world-app
workload:
  liveness-probe-interval: 1m     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-interval: 30s  # Kev will attempt to guess the value of this setting!
                                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-retries

Defines how many times liveness proble should retry upon failure for the workload.
Kev will attempt to infer the number of retries from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.retries` attribute key in a service config it will use its value.
Defaults to `3`.

### Default: `3`

### Possible options: Arbitrary integer. Example: `5`

> liveness-probe-retries:
```yaml
name: hello-world-app
workload:
  liveness-probe-retries: 1m     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-retries: 30s  # Kev will attempt to guess the value of this setting!
                                 # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-initial-delay

Defines how many how long to wait before the first liveness probe runs for the workload.
Kev will attempt to infer the wait time from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.start_period` attribute key in a service config it will use its value.
Defaults to `1m` (1 minute).

### Default: `1m`

### Possible options: Arbitrary time duration. Example: `1m30s`

> liveness-probe-initial-delay:
```yaml
name: hello-world-app
workload:
  liveness-probe-initial-delay: 1m     # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-initial-delay: 30s  # Kev will attempt to guess the value of this setting!
                                       # App component (service) specific. Takes presedence over app level workload setting!
...
```

## liveness-probe-timeout

Defines how many the timeout for the liveness probe command for the workload.
Kev will attempt to infer the timeout value from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.timeout` attribute key in a service config it will use its value.
Defaults to `10s` (10 seconds).

### Default: `10s`

### Possible options: Arbitrary time duration. Example: `30s`

> liveness-probe-timeout:
```yaml
name: hello-world-app
workload:
  liveness-probe-timeout: 1m      # Application level setting. Applies to all components by default.
...
service-a:
  workload:
    liveness-probe-timeout: 30s   # Kev will attempt to guess the value of this setting!
                                  # App component (service) specific. Takes presedence over app level workload setting!
...
```

# Service

This section contains configuration around services in Kubernetes and how they are exposed (or not) to the internet. `service` configuration block may be defined globally for the entire application as top level configuration attribute, or within an individual app component.

**IMPORTANT: At this stage only the first port for each service is processed and used to infer initial configuration!**

## type

Defines type of Kubernetes service for specific workload. Kev will attempt to extract that information from the compose configuration.

The following heuristic is used to determine service type for application component:

* If compose service publishes a port (i.e. defines a port mapping between external and container ports):
    * and specifies a mode as `host`
        * It will assume `NodePort` service type
    * and specifies a mode as `ingress`
        * It will assume `LoadBalancer` service type
    * and doesn't specify port mode
        * It will assume `ClusterIP` service type
* If compose service doesn't publish a port but defines container port.
    * It will assume `Headless` service type

### Default: `None` - no service will be created for the workload by default!

### Possible options: `None`, `Headless`, `ClusterIP`, `Nodeport`, `LoadBalancer`.

> type:
```yaml
name: hello-world-app
service:
  type: ClusterIP         # Application level setting. Applies to all components by default.
...
service-a:
  service:
    type: LoadBalancer    # Kev will attempt to guess the value of this setting!
                          # App component (service) specific. Takes presedence over app level workload setting!
...
```

## nodeport

Defines type Node Port value for Kubernetes service of `NodePort` type.
NOTE: `nodeport` attributes will be ignored for any other service type!
Kev will attempt to extract that information from the compose configuration.

### Default: `nil` - no nodeport defined by default!

### Possible options: Arbitrary integer. Example `10222`.

> nodeport:
```yaml
name: hello-world-app
service:
  type: ClusterIP     # Application level setting. Applies to all components by default.
...
service-a:
  service:
    type: NodePort    # Kev will attempt to guess the value of this setting!
    nodeport: 10222   # Kev will attempt to guess the value of this setting!
                      # App component (service) specific. Takes presedence over app level workload setting!
...
```

## expose

Defines whether to expose the service to external world. This detail can't be easily extracted from the compose file and so need to be specified by the user. By default all component services aren't exposed i.e. have no ingress attached to them.

### Default: `""` - No ingress will be created!

### Possible options:
* `"true"` - ingress will be created with Kubernetes cluster defaults
* `"domain.com,otherdomain.com..."` - comma separated list of domains for the ingress.

> expose:
```yaml
name: hello-world-app
service-a:
  service:
    expose: "my-service.com"
...
```

## tls-secret

Defines whether to use TLS for the exposed service and which secret name contains certificates for the service.

NOTE: This option is only relevant when service is exposed, see: [expose](#expose) above.

### Default: `nil` - No TLS secret name specified by default!

### Possible options: Arbitrary string.

> tls-secret:
```yaml
name: hello-world-app
service-a:
  service:
    expose: "my-service.com"
    tls-secret: "my-service-tls"
...
```

# Volumes

Contains information on all the volumes referenced in the compose file(s) and adds extra configuration regarding storage class and size of each volume.

## class

Defines the class of persitant volume.

### Default: `standard`

### Possible options: Arbitrary string.

> class:
```yaml
name: hello-world-app
volumes:
  vol-1:
    class: my-custom-storage-class
...
```

## size

Defines the size of persitant volume.

### Default: `1Gi`

### Possible options: Arbitrary size string. Example: `10Gi`.

> size:
```yaml
name: hello-world-app
volumes:
  vol-1:
    size: 10Gi
...
```

# Environment

This config section describes application component environment variables and references their values.
Environment can be configured on the "global" application level, or more commonly at each app component level.
Variables defined on the component level will take presedence over the global settings.

> Environment variables can be configured in the following formats:

```
ENV_A: literal-value                      # Literal value
ENV_B: secret.{secret-name}.{secret-key}  # Refer to the a value stored in a secret key
ENV_C: config.{config-name}.{config-key}  # Refer to the a value stored in a configmap key
```

> environment:
```yaml
name: hello-world-app
environment:
  ENV_VAR_A: literal-value                      # Application level setting. Applies to all components by default.
...
service-a:
  environment:
    ENV_VAR_A: another-literal-value              # Local service override. Will take presedence over app level environment.
    ENV_VAR_B: secret.{secret-name}.{secret-key}  # Refer to the a value stored in a secret key
    ENV_VAR_C: config.{config-name}.{config-key}  # Refer to the a value stored in a configmap key
...
```
