---
weight: 11
title: Tako dev with skaffold
---

# Tako Dev with Skaffold

This tutorial will walk you through how to iterate on a Kubernetes application using the Tako `dev` command with an optional [Skaffold](https://skaffold.dev/) hook.

We encourage everyone to get familiar with the [Getting started with Tako](getting-started-with-tako.md) tutorial before proceeding with this guide.

## Iterate Docker Compose

As you've learnt in the [Getting Started](getting-started-with-tako.md) guide, you can turn any Docker Compose project into a set of Kubernetes manifests with a simple `tako render` command.

However, when frequent changes are made to any of the source or environment specific compose files, this becomes cumbersome.

To automate the process of rendering K8s manifests, run Tako in development mode (see command [reference](cli/tako_dev.md) for details):

```sh
# Run Tako in dev mode: starts the watch loop and automatically
# re-renders K8s manifests for changed environments.
$ tako dev
```

It will start the watch loop over source compose & environment override files. When a modification is detected it automatically re-renders Kubernetes manifests for the changed environments.

## Automatic Develop / Build / Push / Deploy

This section will describe how to take advantage of existing Development Lifecycle tools enhancing developer experience when iterating on the Kubernetes application locally. We'll focus on Tako's [Skaffold](https://skaffold.dev/) integration.

#### Initialise Tako project with Skaffold support

In order to take advantage of Skaffold, you must prepare your project accordingly. Provided you haven't initialised a Tako application yet it's as simple as running the following command:

```sh
# Initialise Tako project with Skaffold support
tako init --skaffold -e stage
```

This command prepares your application and bootstraps a new Skaffold config (_skaffold.yaml_) if it doesn't already exist. Alternatively, it'll add environment & helper profiles to already existing Skaffold config automatically. The profiles added by Tako can be used to control which application Kubernetes manifests should be deployed and to which K8s cluster, be it local or remote. They should also come handy when defining steps in CI/CD pipelines.

#### Retrofit Skaffold support in existing Tako project

If a Tako project has been previously initialised without Skaffold support, the easiest way forward to adopt Skaffold is to remove _tako.yaml_ file and initialize the project again.

**Note:** Be mindful that names of all the environments you want to track must be specified - Tako `init` won't automatically discover existing environment override files!

Alternatively, use `skaffold init` to bootstrap _skaffold.yaml_ and tell Tako about the fact by adding the following line in _tako.yaml_ file:

```sh
# Initialize Skaffold in your project
# It'll interactively build skaffold.yaml based on user choices.
# The configuration will be saved in skaffold.yaml file.
skaffold init
```

And then add the following line to the `tako.yaml` file.

```yaml
compose:
  - ...
environments:
  dev: ...
skaffold: skaffold.yaml # <= tell Tako that skaffold is now initialised
```

#### Tako + Skaffold

At this point all you need to do to take advantage of Skaffold integration is to start Tako in [development](cli/tako_dev.md) mode with Skaffold hook enabled:

```sh
# Start Tako in development with Skaffold integration activated
# 1) Starts watch loop and automatically re-renders K8s manifests
#    for specified environments.
# 2) Observes application source code and K8s manifests changes
#    and triggers build/push/deploy.
tako dev --skaffold
```

The command will start two watch loops,
1) One responsible for reconciling changes in your Docker Compose project source and environment override files to produce up-to-date Kubernetes manifests for changed environments,
2) A second loop responsible for watching changes in your project source code and deployment manifests.

Every change made to the Docker Compose project will produce an updated set of K8s manifests for your app, which in turn will inform Skaffold to trigger another Build/Push/Deploy iteration. This will deploy a fresh set of manifests to the target Kuberentes cluster.

There are a few extra bits of information that Skaffold requires to perform its intended task, all of which are itemised below:

* `--namespace | -n` - Informs Skaffold which namespace the application should be deployed to. Default: `default`.
* `--kubecontext | -k` - Specified kubectl context to be used by Skaffold. This determines the cluster to which your application will be deployed to. If not specified it will default to current [kebectl context](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-context-and-configuration).
* `--tako-env` - Tako tracked environment name of which Kubernetes manifests will be deployed to a target cluster/namespace. Defaults to the sandbox `dev` environment, if no environments have been specified.

Additional OPTIONAL flags for Skaffold enabled workflow:

* `--manual-trigger | -m` - Triggers Skaffold's build/push/deploy only after manual user action (hit ENTER to release)
* `--tail | -t` - Will stream application logs once it's deployed to Kubernetes cluster.

When the dev loop is interrupted with Ctrl+C it will automatically cleanup all deployed K8s objects from a target namespace and attempt to prune locally built docker images.

![][diagram]

_NOTE: Image pruning might take some time and in some cases won't remove stale docker images. it's therefore advised that local images are periodically pruned manually._

[diagram]: ../images/tako-with-skaffold.svg
