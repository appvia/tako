---
weight: 11
title: Kev dev with skaffold
---

# Kev Dev w/Skaffold

This tutorial will walk you through how to iterate on a Kubernetes application using the _Kev_ `dev` command with an optional [Skaffold](https://skaffold.dev/) hook.

We encourage everyone to get familiar with the [Getting started with Kev](getting-started-with-kev.md) tutorial before proceeding with this guide.

## Iterate Docker Compose

As you've learnt in the [Getting Started](getting-started-with-kev.md) guide, you can turn any Docker Compose project into a set of Kubernetes manifests
with a simple `kev render` command.

However, when frequent changes are made to any of the source or environment specific compose files, this becomes cumbersome.

To automate the process of rendering K8s manifests, run _Kev_ in development mode (see command [reference](cli/kev_dev.md) for details)

> Run Kev in dev mode. Starts the watch loop and automatically re-renders K8s manifests.
```sh
# for all environments
kev dev

# for specified environment(s)
kev dev -e myenv
````

It will start the watch loop over source compose & environment override files. When a modification is detected it automatically re-renders Kubernetes manifests for environments specified via `--environment | -e` flag(s). If no environments have been specified it defaults to `dev` environment.

## Automatic Develop / Build / Push / Deploy

This section will describe how to take advantage of existing Development Lifecycle tools enhancing developer experience when iterating on the Kubernetes application locally. We'll focus on _Kev_'s [Skaffold](https://skaffold.dev/) integration.

## ⚬ Initialise Kev project with Skaffold support

In order to take advantage of Skaffold, you must prepare your project accordingly. Provided you haven't initialised a Kev application yet it's as simple as running the following command:

> Initialise Kev project with Skaffold support
```sh
kev init --skaffold -e dev -e ...
```

This command prepares your application and bootstraps a new Skaffold config (_skaffold.yaml_) if it doesn't already exist. Alternatively, it'll add environment & helper profiles to already existing Skaffold config automatically. The profiles added by Kev can be used to control which application Kubernetes manifests should be deployed and to which K8s cluster, be it local or remote. They should also come handy when defining steps in CI/CD pipelines.

## ⚬ Retrofit Skaffold support in existing Kev project

If a Kev project has been previously initialised without Skaffold support, the easiest way forward to adopt Skaffold is to remove _kev.yaml_ file and initialize the project again.

**Note:** Be mindful that names of all the environments you want to track must be specified - _Kev_ `init` won't automatically discover existing environment override files!

Alternatively, use `skaffold init` to bootstrap _skaffold.yaml_ and tell Kev about the fact by adding the following line in _kev.yaml_ file:

> Initialize Skaffold in your project
```sh
# It'll interactively build skaffold.yaml based on user choices.
# The configuration will be saved in skaffold.yaml file.

skaffold init
```

> Add the following line in kev.yaml file.
```yaml
compose:
  - ...
environments:
  dev: ...
skaffold: skaffold.yaml # <= tell Kev that skaffold is now initialised
```

## ⚬ Kev + Skaffold

At this point all you need to do to take advantage of Skaffold integration is to start _Kev_ in [development](cli/kev_dev.md) mode with Skaffold hook enabled:

> Start Kev in development with Skaffold integration activated
```sh
# 1) Starts watch loop and automatically re-renders K8s manifests
#    for specified environments.
# 2) Observes application source code and K8s manifests changes
#    and triggers build/push/deploy.

kev dev --skaffold
```

The command will start two watch loops,
1) one responsible for reconciling changes in your Docker Compose project source and environment override files to produce up-to-date Kubernetes manifests for each of specified environments (or default `dev` environment if none provided),
2) a second loop responsible for watching changes in your project source code and deployment manifests.

Every change made to the Docker Compose project will produce an updated set of K8s manifests for your app, which in turn will inform Skaffold to trigger another Build/Push/Deploy iteration. This will deploy a fresh set of manifests to the target Kuberentes cluster.

There are a few extra bits of information that Skaffold requires to perform its intended task, all of which are itemised below:

* `--namespace | -n` - Informs Skaffold which namespace the application should be deployed to. Default: `default`.
* `--kubecontext | -k` - Specified kubectl context to be used by Skaffold. This determines the cluster to which your application will be deployed to. If not specified it will default to current [kebectl context](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-context-and-configuration).
* `--kev-env` - Kev tracked environment name of which Kubernetes manifests will be deployed to a target cluster/namespace. Defaults to the first specified environment passed via `--environment (-e)` flag. If no environments have been specified it will default to `dev` environment.

When the dev loop is interrupted with Ctrl+C it will automatically cleanup all deployed K8s objects from a target namespace and attempt to prune locally built docker images.

_NOTE: Image pruning might take some time and in some cases won't remove stale docker images. it's therefore advised that local images are periodically (manually) pruned._
