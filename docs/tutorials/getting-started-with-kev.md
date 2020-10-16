---
weight: 10
title: Getting started with Kev
---

# Getting started with Kev

This tutorial will walk you through how to **connect your Docker Compose Workflow to Kubernetes** - using _Kev_.

This is NOT a migration. On the contrary, we're going to create a continuous development workflow.

Meaning, your hard-earned Docker Compose skills will make it faster to develop and iterate on Kubernetes.

We'll set up _Kev_, iterate and deploy a [WordPress application](https://docs.docker.com/compose/wordpress/) onto Kubernetes.

The tutorial assumes that you have,
- Prior [docker-compose](https://docs.docker.com/compose/) experience.
- [docker](https://docs.docker.com/engine/install/) & [docker-compose](https://docs.docker.com/compose/install/) installed.
- [_Kev_ installed](../../README.md#installation).
- A local Kubernetes installation. This tutorial uses Docker Desktop ([Mac](https://docs.docker.com/docker-for-mac/#kubernetes) / [Windows](https://docs.docker.com/docker-for-windows/#kubernetes)). As an alternative, use [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [Kind](https://kind.sigs.k8s.io/).

As we walk through the tutorial we'll cover some Kubernetes concepts and how they relate to Docker Compose and _Kev_.

These will be explained under a **Kube Notes** heading.

Finally, we'll use the term "Compose" to mean "Docker Compose".

## Create your docker-compose config

Let's start by creating an empty project directory and `cd` into it.
```shell script
$ mkdir kev-wordpress
$ cd kev-wordpress
```

Then, add a bare bones `docker-compose` file with a basic description of our `wordpress` service that looks like this,
```shell script
$ cat <<EOT >> docker-compose.yaml
version: '3.7'
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8000:80"
    restart: always
EOT
```

This Compose config,
- Sets a `wordpress` service.
- Exposes the service on port `8000`.
- Adds a `restart` always restart policy.

To confirm this is a valid configuration, let's start `wordpress` locally,
```shell script
$ docker-compose up -d    # Run in the background. Force recreate the containers.
```

Navigating to [http://0.0.0.0:8000](http://0.0.0.0:8000) in a browser, should display `wordpress`'s setup page.

Ace, we're all good, let's stop the service,
```shell script
$ docker-compose down -v    # Stop all containers. Remove named volumes.
```

## Preparing for Kubernetes

Compose and Kubernetes address different problems.

_Compose_, helps you wire, develop and run your application locally as a set of services. It's super for development.

_Kubernetes_, however, is designed to help you run your application in a highly available mode using clusters and service replications. It is production grade.

Describing Compose services as a Kubernetes application requires an extra layer that translates concepts from one to the other.

Furthermore, on Kubernetes, you might also want to deploy or promote your app to different "stages", commonly known as environments. Application configuration may vary depending on the environment it is deployed to due to various infrastructure or operational constraints.

So, a good approach to managing your app configuration in different environments is a must.

_Kev_ will help you with all the above! So let's get cracking.

### Compose + Kev

Let's instruct Kev to track our _source Compose_ file, `docker-compose.yaml`, [that we've just created](#create-your-docker-compose-config).

_Kev_ will introspect the Compose config and _infer the key attributes_ to enable Compose services to run on Kubernetes.

Also, as we're moving beyond development, we'll instruct _Kev_ to create two _environment overrides_ to target two different sets of parameters (annotated as service `labels`).

No time to lose, let's get started...

```shell script
$ kev init -e local -e stage
# ✓ Init
#  → Creating kev.yaml ... Done
#  → Creating docker-compose.kev.local.yaml ... Done
#  → Creating docker-compose.kev.stage.yaml ... Done
```

_Kev_ has now been initialised and configured. It has,
- Started tracking the `docker-compose.yaml` file as the _source application definition_.
- Inferred configuration details from the `docker-compose.yaml` file.
- Assigned sensible defaults for any config it couldn't infer.
- Created `local` (useful for testing on our own machine) and `staging` (useful for testing on a remote machine) _Compose environment overrides_.

It has also generated three files:
- `kev.yaml`, a manifest that describes our _source application definition_ and _Compose environment overrides_.
- `docker-compose.kev.*.yaml`, two files to represent our _Compose environment overrides_.

#### Manifest: kev.yaml

The `kev.yaml` manifest file confirm a successful `init`,
```yaml
compose:
  - docker-compose.yaml
environments:
  local: docker-compose.kev.local.yaml
  stage: docker-compose.kev.stage.yaml
...
```

#### Compose environment overrides: docker-compose.kev.*.yaml

The created `local` and `stage` _Compose environment overrides_ are currently identical.

The `labels` section for each service enables you to control how the app runs on Kubernetes. See the [configuration reference](../reference/config-params.md) to find all the available options and understand how they affect deployments.

We'll be adjusting these values soon per target environment. For now, they look as below,

```yaml
version: "3.7"
services:
  wordpress:
    labels:
      kev.workload.liveness-probe-command: '["CMD", "echo", "Define healthcheck command for service wordpress"]'
      kev.workload.replicas: "1"
```

## Moving to Kubernetes

Admittedly, our `wordpress` app is very basic, it only starts a `wordpress` container.

However, all the translation wiring is now in place, so let's run it on Kubernetes!

### Generate Kubernetes manifests

First, we instruct _Kev_ to generate manifests for the required Kubernetes resources.

**Kube Notes**
> Our single `wordpress` Compose service requires [Deployment](https://kubernetes.io/docs/tutorials/kubernetes-basics/deploy-app/deploy-intro/), [Service](https://kubernetes.io/docs/tutorials/kubernetes-basics/expose/expose-intro/) and (an optional) [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/) Kubernetes resources.

Simply run,
```shell script
$ kev render
# ✓ Reconciling environment [local]
# → nothing to update
# ✓ Reconciling environment [stage]
# → nothing to update
# ...............................

# INFO 💡: ⚙️  Output format: kubernetes
# INFO 💡: 🖨️  Rendering local environment
# INFO 💡: Target Dir: k8s/local
# INFO 💡: ⎈  kubernetes file "k8s/local/wordpress-service.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/local/wordpress-deployment.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/local/default-networkpolicy.yaml" created
# INFO 💡: 🖨️  Rendering stage environment
# INFO 💡: Target Dir: k8s/stage
# INFO 💡: ⎈  kubernetes file "k8s/stage/wordpress-service.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/stage/wordpress-deployment.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/stage/default-networkpolicy.yaml" created
# INFO 💡: 🧰 App render complete!
```

In this case, _Kev_,
- Has re-introspected our _source application definition_.
- Has NOT detected any config changes that need to be applied to our _Compose environment overrides_.
- Has generated manifests to enable our app to run in both a `local` and `stage` mode.

We're now ready to run our app on Kubernetes!

### Running on Kubernetes

This means we need to deploy our newly minted manifests to a Kubernetes cluster.

Run the following commands on your local Kubernetes (we use [Docker Desktop](https://docs.docker.com/docker-for-mac/#kubernetes)).

We'll be deploying our app in `local` environment mode.

**Kube Notes**
> We're using `kubectl` the Kubernetes CLI to apply our manifests onto Kubernetes.

> We utilise the [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) `kev-local` to isolate our project resources from other resources in the cluster.

> Our `wordpress` container runs as a single [Pod](https://kubernetes.io/docs/concepts/workloads/pods/) as we're only running 1 replica.

> The `service/wordpress` is a [Service](https://kubernetes.io/docs/tutorials/kubernetes-basics/expose/expose-intro/) that proxies the `Pod` running the container.

> To access the `wordpress` container from our localhost we [port forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) traffic from `service/wordpress` port `8000` to our localhost on port `8080`.

```shell script
$ kubectl create namespace kev-local    # create a namespace to host our app
# namespace/kev-local created

$ kubectl apply -f k8s/local -n kev-local   # apply the generated k8s/local to our namespace
# networkpolicy.networking.k8s.io/default created
# deployment.apps/wordpress created
# service/wordpress created

$ kubectl port-forward service/wordpress 8080:8000 -n kev-local    # make the wordpress service accessible on port 8080
# Forwarding from 127.0.0.1:8080 -> 8000
# Forwarding from [::1]:8080 -> 8000
# Handling connection for 8080
```

Navigate to [http://localhost:8080](http://localhost:8080]) in your browser. This should display `wordpress`'s setup page. The same `wordpress` web page you saw when we ran `docker-compose up -d` earlier.

Hurray!! We're up and running on K8s using **JUST our Compose config (with sensible _Kev_ defaults)**.

For now, `ctrl+c` to stop the `wordpress` service. We need to move beyond a basic container.

## Add a DB service

Let's wire in a database to make our basic `wordpress` app more useful.

In this case this means adding a `db` service backed by a `mysql` container to our Compose config.

### Update Compose config

Update the source `docker-compose.yaml` to,

```yaml
version: '3.7'
services:
  db:
    image: mysql:5.7
    volumes:
        - db_data:/var/lib/mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: somewordpress
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: wordpress
  wordpress:
    image: wordpress:latest
    ports:
      - "8000:80"
    restart: always
    depends_on:
      - db
    environment:
      WORDPRESS_DB_HOST: db:3306
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_NAME: wordpress
volumes:
  db_data:
```

This adds,
- A new `mysql` service.
- A volume `db_data` to store the `mysql` data.
- Environment variables to configure the `mysql` service.
- Environment variables to configure the `wordpress` service to use the `mysql` db.

Running,
```shell script
$ docker-compose up -d
# ...
# Creating network "wordpress-mysql_default" with the default driver
# Creating wordpress-mysql_wordpress_1 ... done
# Creating wordpress-mysql_db_1        ... done
```

Navigate to [http://0.0.0.0:8000](http://0.0.0.0:8000) in a browser.

You should now see a `Welcome` screen for _the famous five-minute WordPress installation process_.

This confirms that all is well.

Stop the service by running,
```shell script
$ docker-compose down -v    # Stop all containers. Remove named volumes.
```

### Re-sync Kubernetes

Now, that we have a new `db` service and `db_data` volume we need to let _Kev_ _infer the key attributes_ to enable the new Compose service and volume to run on Kubernetes.

Also, we've made some minor adjustments to the `wordpress` service. _Kev_ will reconcile those changes.

This will be applied to all _Compose environment overrides_.

Simply, re-run,
```shell script
$ kev render
# ✓ Reconciling environment [local]
# ...
# ...
# ✓ Reconciling environment [stage]
# ...
# ...
# ...............................
# INFO 💡: ⚙️  Output format: kubernetes
# INFO 💡: 🖨️  Rendering stage environment
# INFO 💡: Target Dir: k8s/stage
# ...
# INFO 💡: ⎈  kubernetes file "k8s/stage/db-statefulset.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/stage/db-data-persistentvolumeclaim.yaml" created
# ...
# INFO 💡: 🖨️  Rendering local environment
# INFO 💡: Target Dir: k8s/local
# ...
# INFO 💡: ⎈  kubernetes file "k8s/local/db-statefulset.yaml" created
# INFO 💡: ⎈  kubernetes file "k8s/local/db-data-persistentvolumeclaim.yaml" created
# ...
# INFO 💡: 🧰 App render complete!
```

This time round, _Kev_
- Has detected and inferred config for the new `mysql` service and `db_data` volume.
- It assigned sensible defaults for any config it couldn't infer.
- It re-generated the kubernetes manifests for the `local` and `stage` deployment environments.

**Kube Notes**
> To accommodate the `db` service, _Kev_ uses the [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) Kubernetes resource as the `db` service requires persistent storage.

> _Kev_ uses the [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) resource to provide the `db` service with the required `db_data` volume it needs to store data.

We'll be re-deploying our app in `local` environment mode.

Run the following command on your local Kubernetes instance (we use [Docker Desktop](https://docs.docker.com/docker-for-mac/#kubernetes)).

```shell script
$ kubectl apply -f k8s/local -n kev-local   # re-apply the re-generated k8s/local manifests to our namespace
# persistentvolumeclaim/db-data created
# service/db created
# statefulset.apps/db created
# networkpolicy.networking.k8s.io/default configured
# deployment.apps/wordpress configured
# service/wordpress configured
```

**This BREAKS our running app** - to fix it we need to understand how service discovery differs between Compose and Kubernetes.

### Fix db service discovery

In our Compose config, the `db` service does not have a `ports` attribute. Meaning it is not exposed externally as there are no published ports.

This is not an issue for dependent Compose services as containers connected to the same user-defined bridge network effectively expose all ports to each other and communicate using service names or aliases.

**Kubernetes is different**. To help our `wordpress` containers connect to the `db`, Kubernetes requires an explicit `Service` resource.

The fix is simple, we need to instruct _Kev_ to recognise `db` as service that will be accessed from other services.

Simply add the `ports` attribute like below,

```yaml
version: '3.7'
services:
  db:
    ...
    ...
    ports:
      - "3306"

  wordpress:
    ...
volumes:
  ...
```

Then, re-render and re-deploy,
```shell script
$ kev render
# ...
# INFO 💡: Target Dir: k8s/local
# ...
# INFO 💡: ⎈  kubernetes file "k8s/stage/db-service.yaml" created
# ...
# INFO 💡: Target Dir: k8s/stage
#...
# INFO 💡: ⎈  kubernetes file "k8s/stage/db-service.yaml" created
# ...

$ kubectl apply -f k8s/local -n kev-local   # re-apply the re-generated k8s/local manifests to our namespace
# service/db created
# ...

$ kubectl port-forward service/wordpress 8080:8000 -n kev-local    # make the wordpress service accessible on port 8080
# Forwarding from 127.0.0.1:8080 -> 8000
# Forwarding from [::1]:8080 -> 8000
# Handling connection for 8080
```

Navigate to [http://0.0.0.0:8000](http://0.0.0.0:8000) in a browser.

... and Yay!! Live on Kubernetes, you should now see the `Welcome` screen for _the famous five-minute WordPress installation process_.

`ctrl+c` to stop the `wordpress` service.

## Run more replicas

As it happens, we have a requirement that our `stage` environment should mirror `production` as much as possible.

In this case, we need to run 5 instances of the `wordpress` service to simulate how the app works in a heavy user traffic setting.

Let's make this happen. We need to edit our `docker-compose.kev.stage.yaml` Compose environment override file.

We'll change the `label`: `kev.workload.replicas`, from "1" to "5".

```yaml
version: "3.7"
services:
  wordpress:
    labels:
      ...
      kev.workload.replicas: "5"
```

### Re-sync Kubernetes

When we re-sync _Kev_, the `stage` environment's generated manifests will reflect the new number of `replicas`.

```shell script
$ kev render
# ✓ Reconciling environment [local]
#  → nothing to update
# ✓ Reconciling environment [stage]
#  → nothing to update
# ..............................

# INFO 💡: ⚙️  Output format: kubernetes
...
# INFO 💡: 🧰 App render complete!
```

Re-deploying the manifests to Kubernetes on a `stage` environment will run 5 `wordpress` [Pods](https://kubernetes.io/docs/concepts/workloads/pods/) on Kubernetes - meaning 5 `wordpress` instances.

We now have 2 different target environments,
- `local` will only run a single `wordpress` instance.
- `stage` will only run a 5 `wordpress` instances.

These are easily tracked in easy to understand Compose files.

Check the [configuration reference](../reference/config-params.md) if you want to configure other params.

## Conclusion

We have successfully moved a `wordpress` app from a local Docker Compose development flow to a connected multi-environment Kubernetes setup.

_Kev_ facilitated all the heavy lifting. It enabled us to easily iterate on and manage our target environments.

We also have an understanding of the **gotchas** we can face when moving from Compose to Kubernetes.

All the generated manifests can be tracked in source control and shared in a team context.

Finally, you can find the artefacts for this tutorial here: [wordpress-mysql example](../../examples/wordpress-mysql/README.md).

