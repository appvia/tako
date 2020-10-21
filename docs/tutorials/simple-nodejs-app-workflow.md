---
weight: 12
title: Kev workflow example with a simple Node.js App
---

# Workflow example with a simple Node.js App

## Get Kev

* Download [Kev](https://github.com/appvia/kev/releases/latest) & add to the PATH

## Initialise project

> Inside the project directory (`./examples/node-app`) instruct Kev to:
> * provide a way to adjust Kubernetes specific configuration for 3 distinct environments,
> * prepare the app for use with [Skaffold](https://skaffold.dev/).

```sh
kev init -e dev -e staging -e prod --skaffold
```

You will notice that 3 separate environment specific configuration files have been created:

```
|- docker-compose.kev.dev.yaml
|- docker-compose.kev.staging.yaml
|- docker-compose.kev.prod.yaml
```

Adjust Kubernetes specific application parameters for each of the components as and when necessary. This is done via Compose [labels](../../docs/reference/config-params.md).

It'll also bootstrap the Skaffold config file (`skaffold.yaml`). If skaffold.yaml previously existed then it'll add additional profiles to it.

## Iterate on the application

### One-off K8s manifests render

Once changes to base & environment specific Compose file(s) have been made

```sh
kev render
```

This will produce K8s manifests for all environments. See [help](../../docs/cli/kev_render.md) for usage examples.

Inspect produced Kubernetes manifests at default `k8s` directory.

### Watch for Compose changes and auto-rebuild K8s manifests

Run the command below to continously watch for changes made to any of the source / environment Compose files related to your application and automatically rebuild Kubernetes manifests for all environments. See [help](../../docs/cli/kev_dev.md) for usage examples.

```sh
kev dev
```

### Watch for Compose and Application source code changes with Build/Push/Deploy loop enabled

Watch for the changes to your application Compose files, as well as application source code and automatically rebuild the K8s manifests and Build/Push/Deploy the app (via Skaffold dev loop) to any Kubernetes cluster upon detected changes. See [help](../../docs/cli/kev_dev.md) for usage examples.

```sh
kev dev --skaffold
```

Open the browser at `http://localhost:8080`. You should see `Hello World` displayed on the screen.


*NOTE*: The command above will use current kubectl context, and will attempt to deploy your app to `default` namespace. Those can be adjusted with `--kubecontext=<context>` & `--namespace=<ns>` options added to the command above.

*IMPORTANT*: If `--kubecontext` is pointing at remote Kubernetes cluster you need to make sure that you adjust `image` in docker-compose.yaml file so that it points at registry you control and are able to push to. Should that be a private registry then make sure that the remote Kubernetes cluster is able to [pull images from it](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).

Once the app has been build, pushed and deployed via _Kev's_ Skaffold integration you may inspect that the Node app is running in your cluster:

```sh
$ kubectl --context docker-desktop -n default get po

NAME                   READY   STATUS    RESTARTS   AGE
app-69d87ffbc8-pq4bs   1/1     Running   0          9s
```

Now, try to adjust number of replicas for the app by modifying `kev.workload.replicas` label value to "2". You should observe `dev` loop pick up all the changes and do the hard work of generating K8s manifests, building, pushing and deploying your application automatically.

```sh
$ k -n default get po
NAME                   READY   STATUS        RESTARTS   AGE
app-85b85cf987-8ngks   1/1     Running       0          14s
app-85b85cf987-h9l2m   1/1     Running       0          16s
```

Now try and modify the app source code by changing `res.send('Hello World!');` in `server.js` file to something different e.g. `res.send('FOO BAR!');`. Once changes are saved the application will be automatically rebuilt and redeployed. Visit `http://localhost:8080` and see `FOO BAR` appearing on the screen.

That's it. Issue Ctrl+C to stop the `dev` loop and delete objects deployed to Kubernetes namespace.
