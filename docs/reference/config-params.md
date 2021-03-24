---
weight: 50
title: Kev configuration reference
---

# Configuration

Kev leverages Docker Compose specification to configure and prepare an application for deployment in Kubernetes. Environment configuration lives in a dedicated docker compose override file, which automatically gets applied to the project's source sdocker compose files at `render` phase.

### Component level configuration

Configuration is divided into the following groups of parameters:
* [Component](#-component)
* [Workload](#-workload)
* [Service](#-service)
* [Volumes](#-volumes)
* [Environment](#-environment)

# → Component

This configuration group contains application composition related settings. Configuration parameters can be individually defined via set of labels (listed below) for each application stack component.

## kev.component.enabled

Defines whether a component is enabled or disabled. All application components are enabled by default.

### Default: `true`

### Possible options: `true`, `false`.

> kev.workload.image-pull-policy:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.component.enabled: false
...
```

# → Workload

This configuration group contains Kubernetes `workload` specific settings. Configuration parameters can be individually defined via set of labels (listed below) for each application stack component.

## kev.workload.image-pull-policy

Defines docker image pull policy from the container registry. See official K8s [documentation](https://kubernetes.io/docs/concepts/containers/images/#updating-images).

### Default: `IfNotPresent`

### Possible options: `IfNotPresent`, `Always`.

> kev.workload.image-pull-policy:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.image-pull-policy: IfNotPresent
...
```

## kev.workload.image-pull-secret

Defines docker image pull secret which should be used to pull images from the container registry. See official K8s [documentation](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod).

### Default: ""

### Possible options: arbitrary string.

> kev.workload.image-pull-secret:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.image-pull-secret: my-image-pull-secret-name
...
```

## kev.workload.restart-policy

Defines the restart policy for individual application component in the event of a container crash. Kev will attempt to infer that setting for each compose service defined, however in some cases manual override might be necessary. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy).

### Default: `Always`

### Possible options: `Always`, `OnFailure`, `Never`.

> kev.workload.restart-policy:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.restart-policy: Never
...
```

## kev.workload.service-account-name

Defines the kubernetes Service Account name to run a workload with. Useful when specific access level associated with a Service Account is requiered for a given workload type. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/).

### Default: `default`

### Possible options: Arbitrary string.

> kev.workload.service-account-name:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.service-account-name: my-special-service-account-name
...
```

## kev.workload.pod-security-run-as-user

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload. This option is concerned with setting up an appropriate User ID (`runAsUser` field) which specifies that for any Containers in the Pod, all processes will run with user ID as specified by the value.

### Default: nil (not specified)

### Possible options: arbitrary numeric UID, example `1000`.

> kev.workload.pod-security-run-as-user:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.pod-security-run-as-user: 1000
...
```

## kev.workload.pod-security-run-as-group

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload. This option is concerned with setting up an appropriate Group ID (`runAsGroup` field) which specifies the primary group ID for all processes within any containers of the Pod. If this field is omitted (currently a default), the primary group ID of the container will be root(0). Any files created will also be owned by user with specified user ID (`runAsUser` field) and group ID (`runAsGroup` field) when runAsGroup is specified.

### Default: nil (not specified)

### Possible options: Arbitrary numeric GID. Example `1000`.

> kev.workload.pod-security-run-as-group:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.pod-security-run-as-group: 2000
...
```

## kev.workload.pod-security-fs-group

Defines the [Pod Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for the kubernetes workload. This option is concerned with setting up a supplementary group `fsGroup` field. If specified, all processes of the container are also part of this supplementary group ID. The owner for attached volumes and any files created in those volume will be Group ID as specified by the value of this configuration option.

### Default: nil (not specified)

### Possible options: Arbitrary numeric GID. Example `1000`.

> kev.workload.pod-security-fs-group:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.pod-security-fs-group: 3000
...
```

## kev.workload.type

Defines the Kubernetes workload type controller. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/). Kev will attempt to infer workload type from the information specified in the compose file.

Kev uses the following heuristics to derieve the type of workload:

If compose file(s) specifies the `deploy.mode` attribute key in a compose project service config, and it is set to "global" then `DaemonSet` workload type is assumed. Otherwise, workload type will default to `Deployment` unless volumes are in use, in which case workload will default to `StatefulSet`.

### Default: `Deployment`

### Possible options: `Pod`, `Deployment`, `StatefulSet`, `Daemonset`, `Job`.

> type:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.type: StatefulSet
...
```

## kev.workload.replicas

Defines the number of instances (replicas) for each application component. See K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#replicas). Kev will attempt to infer number of replicas type from the information specified in the compose file.

Kev uses the following heuristics to derieve the number of replicas for each service:

If compose file(s) specifies the `deploy.replicas` attribute key in a project service config it will use its value.
Otherwise, number of replicas will default to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> replicas:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.replicas: 3
...
```

## kev.workload.autoscale-max-replicas

Defines the maximum number of instances (replicas) the application component should automatically scale up to. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account when initial number of replicas is lower than this parameter.

### Default: `0`

### Possible options: Arbitrary integer value. Example: `10`.

> autoscale-max-replicas:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.autoscale-max-replicas: 3
...
```

## kev.workload.autoscale-cpu-threshold

Defines the CPU utilization threshold for the horizontal pod autoscaler for the application component. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account maximum number of replicas for the application component is defined.

### Default: `70` (70% cpu utilization)

### Possible options: Arbitrary integer value. Example: `80`.

> autoscale-cpu-threshold:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.autoscale-cpu-threshold: 80
...
```

## kev.workload.autoscale-mem-threshold

Defines the Memory utilization threshold for the horizontal pod autoscaler for the application component. See K8s [documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/). This setting is only taken into account maximum number of replicas for the application component is defined.

### Default: `70` (70% memory utilization)

### Possible options: Arbitrary integer value. Example: `80`.

> autoscale-mem-threshold:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.autoscale-mem-threshold: 80
...
```

## kev.workload.rolling-update-max-surge

Defines the number of pods that can be created above the desired amount of pods during an update. See official K8s [documentation](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#proportional-scaling). Kev will attempt to infer this number from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.update_config.parallelism` attribute key in a service config it will use its value.
Otherwise it will default to `1`.

### Default: `1`

### Possible options: Arbitrary integer value. Example: `10`.

> kev.workload.rolling-update-max-surge:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.rolling-update-max-surge: 2
...
```

## kev.workload.cpu

Defines the CPU share request for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.cpus` attribute key in a project service config it will use its value. Otherwise it'll assume sensible default of `0.1` (equivalent of 100m in Kubernetes).

### Default: `0.1`

### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> kev.workload.cpu:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.cpu: 1
...
```

## kev.workload.max-cpu

Defines the CPU share limit for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer CPU request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.limits.cpus` attribute key in a service config it will use its value.
Otherwise it'll default to a sensible default of `0.2` (equivalent of 200m in Kubernetes).

### Default: `0.2`

### Possible options: Arbitrary [CPU units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu). Examples: `0.2` == `200m`.

> kev.workload.max-cpu:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.max-cpu: 2
...
```

## kev.workload.memory

Defines the Memory request for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer Memory request from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.reservations.memory` attribute key in a service config it will use its value. Otherwise it'll default to a sensible quantity of `10Mi`.

### Default: `10Mi`

### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> kev.workload.memory:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.memory: 200Mi
...
```

## kev.workload.max-memory

Defines the Memory limit for a given workload. See official K8s [documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). Kev will attempt to infer Memory limit from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `deploy.resources.limits.memory` attribute key in a service config it will use its value.
Otherwise it'll default to a sensible quantity of `500Mi`.

### Default: `500Mi`

### Possible options: Arbitrary [Memory units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory). Examples: `64Mi`, `1Gi`...

> kev.workload.max-memory:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.max-memory: 0.3Gi
...
```

## kev.workload.liveness-probe-type

Defines the workload's liveness probe type. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.disable` attribute key in a service config it will set the probe type to `none`.
Otherwise it'll default to `exec` (liveness probe active!)

### Default: `exec`

### Possible options: none, exec, http, tcp.

> kev.workload.liveness-probe-type:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-type: none
...
```

## kev.workload.liveness-probe-command

Defines the liveness probe command to be run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the command from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.test` attribute key in a service config it will use its value.
If probe is not defined it will prompt the user to define one by injecting generic echo command.

### Default: echo "prompt user to define the probe"

### Possible options: shell command

> kev.workload.liveness-probe-command:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-command: ["/is-my-service-alive.sh"]
...
```

## kev.workload.liveness-probe-http-port

Defines the liveness probe port to be used for the workload when the type is `http`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request). 

### Possible options: Integer

> kev.workload.liveness-probe-http-port:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-http-port: 8080
...
```

## kev.workload.liveness-probe-http-path

Defines the liveness probe path to be used for the workload when the type is `http`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request). 

### Possible options: String

> kev.workload.liveness-probe-http-path:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-http-path: /status
...
```

## kev.workload.liveness-probe-tcp-port

Defines the liveness probe path to be used for the workload when the type is `tcp`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request). 

### Possible options: Integer

> kev.workload.liveness-probe-tcp-port:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-tcp-port: 8080
...
```

## kev.workload.liveness-probe-interval

Defines how often liveness proble should run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the interval from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.interval` attribute key in a service config it will use its value.
Otherwise it'll default to `1m` (1 minute).

### Default: `1m`

### Possible options: Time duration

> kev.workload.liveness-probe-interval:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-interval: 30s
...
```

## kev.workload.liveness-probe-retries

Defines how many times liveness proble should retry upon failure for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the number of retries from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.retries` attribute key in a service config it will use its value.
Otherwise it'll default to `3`.

### Default: `3`

### Possible options: Arbitrary integer. Example: `5`

> kev.workload.liveness-probe-retries:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-retries: 10
...
```

## kev.workload.liveness-probe-initial-delay

Defines how many how long to wait before the first liveness probe runs for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the wait time from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.start_period` attribute key in a service config it will use its value.
Otherwise it'll default to `1m` (1 minute).

### Default: `1m`

### Possible options: Arbitrary time duration. Example: `1m30s`

> kev.workload.liveness-probe-initial-delay:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-initial-delay: 10s
...
```

## kev.workload.liveness-probe-timeout

Defines the timeout for the liveness probe for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command). Kev will attempt to infer the timeout value from the information specified in the compose file.

Kev uses the following heuristics to derieve that information for each service:

If compose file(s) specifies the `healthcheck.timeout` attribute key in a service config it will use its value.
Otherwise it'll default to `10s` (10 seconds).

### Default: `10s`

### Possible options: Arbitrary time duration. Example: `30s`

> kev.workload.liveness-probe-timeout:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.liveness-probe-timeout: 10s
...
```

## kev.workload.readiness-probe-type

Defines the workload's probe type. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: `none`

### Possible options: none, exec, http, tcp.

> kev.workload.readiness-probe-type:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-type: none
...
```

## kev.workload.readiness-probe-command

Defines the readiness probe command to be run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: nil

### Possible options: shell command

> kev.workload.liveness-probe-command:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-command: ["/is-my-service-ready.sh"]
...
```

## kev.workload.readiness-probe-http-port

Defines the readiness probe port to be used for the workload when the type is `http`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes). 

### Possible options: Integer

> kev.workload.readiness-probe-http-port:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-http-port: 8080
...
```

## kev.workload.readiness-probe-http-path

Defines the readiness probe path to be used for the workload when the type is `http`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes). 

### Possible options: String

> kev.workload.readiness-probe-http-path:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-http-path: /status
...
```

## kev.workload.readiness-probe-tcp-port

Defines the readiness probe path to be used for the workload when the type is `tcp`. 
See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes). 

### Possible options: Integer

> kev.workload.readiness-probe-tcp-port:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-tcp-port: 8080
...
```


## kev.workload.readiness-probe-interval

Defines how often readiness proble should run for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: `1m`

### Possible options: Time duration

> kev.workload.readiness-probe-interval:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-interval: 30s
...
```

## kev.workload.readiness-probe-retries

Defines how many times readiness proble should retry upon failure for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: `3`

### Possible options: Arbitrary integer. Example: `5`

> kev.workload.readiness-probe-retries:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-retries: 10
...
```

## kev.workload.readiness-probe-initial-delay

Defines how many how long to wait before the first readiness probe runs for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: `1m`

### Possible options: Arbitrary time duration. Example: `1m30s`

> kev.workload.readiness-probe-initial-delay:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-initial-delay: 10s
...
```

## kev.workload.readiness-probe-timeout

Defines how many the timeout for the readiness probe for the workload. See official K8s [documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

### Default: `10s`

### Possible options: Arbitrary time duration. Example: `30s`

> kev.workload.readiness-probe-timeout:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.workload.readiness-probe-timeout: 10s
...
```

# → Service

The `service` group contains configuration detail around Kubernetes services and how they get exposed externally.

**IMPORTANT: At this stage only the first port for each service is processed and used to infer initial configuration!**

## kev.service.type

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

> kev.service.type:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.service.type: LoadBalancer
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


## kev.service.nodeport.port

Defines type Node Port value for Kubernetes service of `NodePort` type. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport).
NOTE: `nodeport` attributes will be ignored for any other service type!
Kev will attempt to extract that information from the compose configuration.

### Default: `nil` - no nodeport defined by default!

### Possible options: Arbitrary integer. Example `10222`.

> kev.service.nodeport.port:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.service.nodeport.port: 5555
...
```

## kev.service.expose

Defines whether to expose the service to external world. This detail can't be easily derived from the compose file and so in order to expose a service to external world user must explicitly instruct Kev to do so. By default all component services aren't exposed i.e. have no ingress attached to them.

### Default: `""` - No ingress will be created!

### Possible options:
* `"true"` - ingress will be created with Kubernetes cluster defaults
* `"domain.com,otherdomain.com..."` - comma separated list of domains for the ingress.

> kev.service.expose:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.service.expose: "my-awesome-service.com"
...
```

## kev.service.expose.tls-secret

Defines whether to use TLS for the exposed service and which secret name contains certificates for the service. See official K8s [documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls).

NOTE: This option is only relevant when service is exposed, see: [kev.service.expose](#kev-service-expose) above.

### Default: `nil` - No TLS secret name specified by default!

### Possible options: Arbitrary string.

> kev.service.expose.tls-secret:
```yaml
version: 3.7
services:
  my-service:
    labels:
      kev.service.expose: "my-domain.com"
      kev.service.expose.tls-secret: "my-service-tls-secret-name"
...
```

# → Volumes

This configuration group contains Kubernetes persistent `volume` claim specific settings. Configuration parameters can be individually defined via a set of labels (see below), for each volume referenced in the project compose file(s).


## kev.volume.storage-class

Defines the class of persitant volume. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: `standard`

### Possible options: Arbitrary string.

> kev.volume.storage-class:
```yaml
version: 3.7
volumes:
  vol1:
    labels:
      kev.volume.storage-class: my-custom-storage-class
...
```

## kev.volume.size

Defines the size of persitant volume. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: `1Gi`

### Possible options: Arbitrary size string. Example: `10Gi`.

> kev.volume.size:
```yaml
version: 3.7
volumes:
  vol1:
    labels:
      kev.volume.size: 10Gi
...
```

## kev.volume.selector

Defines a label selector to further filter the set of volumes. Only the volumes whose labels match the selector can be bound to the PVC claim. See official K8s [documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

### Default: ``

### Possible options: Arbitrary string. Example: `data`.

> kev.volume.selector:
```yaml
version: 3.7
volumes:
  vol1:
    labels:
      kev.volume.selector: my-volume-selector-label
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
    labels:
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
    labels:
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
    labels:
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
    labels:
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
    labels:
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
    labels:
      ...
    environment:
      ENV_VAR_E: container.{container-name}.{resource-field} # Refer to the a value of the K8s workload Container resource field
                                                             # e.g `limits.cpu` to get max CPU allocatable to the container

```

### Supported `container.{name}.{....}` resource fields:
* `limits.cpu`, `limits.memory`, `limits.ephemeral-storage` - return value of selected container `limit` field
* `requests.cpu`, `requests.memory`, `requests.ephemeral-storage` - return value of selected container `requests` field
