---
weight: 12
title: Tako workflow example with a simple Node.js App
---

# Development workflow example with a simple Node.js app

## Get Tako

* Download [Tako](https://github.com/appvia/tako/releases/latest) & add to the PATH

## Initialise project

> Inside the project directory (`./examples/node-app`) instruct Tako to:
> * provide a way to adjust Kubernetes specific configuration for 3 distinct environments,
> * prepare the app for use with [Skaffold](https://skaffold.dev/).

```sh
$ tako init -e staging -e prod --skaffold
```

You will notice that 3 separate environment specific configuration files have been created:

> Added environment specific override files:
```sh
|- docker-compose.env.dev.yaml
|- docker-compose.env.staging.yaml
|- docker-compose.env.prod.yaml
```

(The `dev` configuration is created by default by Tako).

Adjust Kubernetes specific application parameters for each of the components as and when necessary. This is done via Compose [extensions](../../docs/reference/config-params.md).

It'll also bootstrap the Skaffold config file (`skaffold.yaml`). If skaffold.yaml previously existed then it'll add additional profiles to it.

## Iterate on the application

### One-off K8s manifests render

Once changes to base & environment specific Compose file(s) have been made

```sh
# One-off Kubernetes manifests render:
$ tako render
```

This will produce K8s manifests for all environments. See [help](../../docs/cli/tako_render.md) for usage examples.

Inspect produced Kubernetes manifests at default `k8s` directory.

### Watch for Compose changes and auto-rebuild K8s manifests

Run the command below to continuously watch for changes made to any of the source / environment Compose files related to your application and automatically rebuild Kubernetes manifests for changed environments.

See [help](../../docs/cli/tako_dev.md) for usage examples.

```sh
# Watch Compose changes and auto render Kubernetes manifests:
$ tako dev
```

### Watch for Compose and Application source code changes with Build/Push/Deploy loop enabled

Watch for changes to your application's Compose files plus project source code. Then, automatically rebuild the K8s manifests and build/push/deploy the app via Skaffold dev loop to any Kubernetes cluster upon detected changes.

See [help](../../docs/cli/tako_dev.md) for usage examples.

```sh
# Watch Compose and App source code changes, render manifests and
# build/push/deploy with Skaffold:
$ tako dev --skaffold
```

Open the browser at `http://localhost:8080`. You should see `Hello World` displayed on the screen.


*NOTE*: The command above will use current kubectl context, and will attempt to deploy your app to `default` namespace. Those can be adjusted with `--kubecontext=<context>` & `--namespace=<ns>` options added to the command above.

**IMPORTANT**: If `--kubecontext` is pointing at remote Kubernetes cluster you need to make sure you adjust the `image` in docker-compose.yaml file so that it points at registry you control and are able to push to. Should that be a private registry then make sure that the remote Kubernetes cluster is able to [pull images from it](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).

Once the app has been built, pushed and deployed via Tako's Skaffold integration you may inspect that the Node app is running in your cluster:

```sh
# List Kubernetes application pods for the Node.js app:
$ kubectl --context docker-desktop -n default get po

NAME                   READY   STATUS    RESTARTS   AGE
app-69d87ffbc8-pq4bs   1/1     Running   0          9s
```

Now, try to adjust number of replicas for the app by modifying `x-k8s.workload.replicas` value to `2`. You should observe `dev` loop pick up all the changes and do the hard work of generating K8s manifests, building, pushing and deploying your application automatically.

```sh
# List Kubernetes application pods for the Node.js app after change
# to desired replicas number:
$ kubectl --context docker-desktop -n default get po

NAME                   READY   STATUS        RESTARTS   AGE
app-85b85cf987-8ngks   1/1     Running       0          14s
app-85b85cf987-h9l2m   1/1     Running       0          16s
```

Now try and modify the app source code by changing `res.send('Hello World!');` in `server.js` file to something different e.g. `res.send('FOO BAR!');`. Once changes are saved the application will be automatically rebuilt and redeployed. Visit `http://localhost:8080` and see `FOO BAR` appearing on the screen.

That's it. Issue Ctrl+C to stop the `dev` loop and delete objects deployed to Kubernetes namespace.
