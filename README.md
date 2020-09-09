# Kev

> Develop Kubernetes apps iteratively with Docker-Compose

![Stability:Beta](https://img.shields.io/badge/stability-beta-orange)
![CircleCI](https://img.shields.io/circleci/build/github/appvia/kev/master)
![GitHub tag (latest SemVer)](https://img.shields.io/github/v/release/appvia/kev)
![License: Apache-2.0](https://img.shields.io/github/license/appvia/kev)


_Kev_ helps developers port and iterate Docker Compose apps onto Kubernetes. It understands the Docker Compose application topology and prepares it for deployment in (multiple) target environments, with minimal user input.

We leverage the [Docker Compose](https://docs.docker.com/compose/compose-file/) specification and allow for target-specific configurations to be applied to each component of the application stack, simply.

_Kev_ is opinionated in its choice of Kubernetes elements you should be able to control. It automatically infers key config parameters by analysing and reconciling changes in the project source compose file(s). The configuration parameters can be manually overridden to allow for better control of a cloud application deployment on Kubernetes.

_Kev_ reduces the need for Kubernetes expertise in the team. The generated Kubernetes deployment configuration follows best industry practices, with a thin layer of config options to enable further control. See [kev reference documentation](docs/reference/config-params.md) for a list of available options.


## Features

* **Simplicity** - Based on familiar Docker Compose specification. There is no new framework to learn, no new specification to embrace, and vastness of Kubernetes reduced to a limited set of easy to follow configuration parameters. You focus on the app development. Kev will prepare it for deployment in Kubernetes.

* **Multi-environment support** - Parameterisation enabled with the same configuration primitives you're already familiar with. Each defined environment gets its own docker-compose override file, which is there to control the behaviour of your application in Kubernetes in a simple and consistent way.

* **Best practice out of the box** - Best practice is codified and embedded in the translation layer, so you don't have to think about what's required to run your project application on Kubernetes.

* **Secure** - _Kev_ is opinionated about the secret management. At this stage of its relatively short life it delegates that responsibility to the user, to remove the risk of potential uncontrolled secrets leak. No secrets == No leaks!

* **No vendor lock-in** - Because you already use docker-compose, you can keep using it, even if _Kev_ turns out to be not your cup of tea.

* **Easy integrations** - You may use generated Kubernetes manifests with any tool / framework of your choice. We aim at adding some useful integrations further improving developer experience.

## Installation

All you need to get started quickly is the [kev](https://github.com/appvia/kev/releases) binary added to your PATH, and one or more docker compose files.

## Quickstart

- `kev init` - identifies a project's Compose Kubernetes source files and creates Compose environment overrides.
- `kev render` - detects, applies any config changes and generates deployment manifests.
- `kev help` - run it if you're a little lost.

### init

Run the following command within your project directory:

```sh
kev init
```

This identifies the default `docker-compose.yaml` and (if present) `docker-compose.override.yaml` files in your project directory. They will be used as the source of truth for your application deployment in Kubernetes.

Also, it creates a default `dev` environment and its Compose override file.  

Here's another example. It uses an alternate `docker-compose` file with `stage` & `prod` environments:

```sh
kev init -f my-docker-compose.yaml -e stage -e prod
```

It makes use of,
- `-f` flag, to specify an alternate filename.
- `-e` flags, to specify different deployment environments.

Creating the files below in your project directory:

```sh
├── docker-compose.kev.stage.yaml       # stage Compose environment override file
├── docker-compose.kev.prod.yaml        # prod Compose environment override file
├── kev.yaml                            # kev project manifest
├── ...
```

Here's what happened, _Kev_ has,
- Inferred the configuration details already present in your compose Kubernetes deployment sources.
- Assigned sensible defaults for any config it couldn't infer.
- Created Compose overrides files for the `stage` and `prod` environments.

That's it, your _Kev_ project is now ready!

From now on it can,
- Detect edits in your source compose file.
- Apply any related config changes to your compose environment overrides.
- Generate deployment manifests.   

You can now customise your deployment targets by altering values in the relevant Compose environment override file. 

### render

We now need to generate manifests based on your Docker Compose config and environments. You'll use these manifests to deploy your app to Kubernetes. 

Run the following command from your project root:

```sh
kev render
```

The command above,
- Detects edits you made to the project's source compose file(s).
- Applies any found config changes to your compose environment overrides.
- Generates kubernetes manifests based on all compose files including environment overrides.
- Generates kubernetes manifests for all environments.

The directory below should now appear in your project directory:

```sh
├── k8s         # stores the Kubernetes manifests for all target deployment environments. 
├──── prod      # prod deploymeny environment
├────── ...     # prod manifests
├──── stage     # stage deploymeny environment
├────── ...     # stage manifests
```

Other flag options include,
- `-f` flag, to specify the deployment files format (defaults to `kubernetes`).
- `-s` flag, to render application's manifests to a single file.
- `-d` flag, to specify the output directory for generated manifests (it will contain sub-directories, each for a separate environment name).
- `-e` flag(s), to control which environments to generate the manifests for.

**Note:** Generated manifests should **NOT** be treated as templates as they are fully expanded.

#### How can I deploy the app to Kubernetes?

To deploy your app onto Kubernetes,
- Ensure you can access a running Kubernetes installation, either locally (e.g. [Docker Desktop](https://docs.docker.com/desktop/), [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), etc...) or remotely.
- Use [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) to apply the manifests.

In this example, we deploy the `stage` environment:

```sh
kubectl apply -f k8s/stage     # deploys your app with stage settings onto the default namespace
```

#### Other deployment tooling    

With _Kev_, you can use any Kubernetes deployment tool or framework you're familiar with, e.g `skaffold`, `tilt`, etc...

Check our [Roadmap][roadmap] for upcoming planned integrations.

## Tutorial

This tutorial will walk you through how to **connect your Docker Compose Workflow to Kubernetes** - using _Kev_.

This is NOT a migration. On the contrary, we're going to create a continuous development workflow.

Meaning, your hard-earned Docker Compose skills will make it faster to develop and iterate on Kubernetes.   

We'll set up _Kev_, iterate and deploy a [WordPress application](https://docs.docker.com/compose/wordpress/) onto Kubernetes.

The tutorial assumes that you have,
- Prior [docker-compose](https://docs.docker.com/compose/) experience. 
- [docker](https://docs.docker.com/engine/install/) & [docker-compose](https://docs.docker.com/compose/install/) installed.
- [_Kev_ installed](#installation).
- A local Kubernetes installation. This tutorial uses Docker Desktop ([Mac](https://docs.docker.com/docker-for-mac/#kubernetes) / [Windows](https://docs.docker.com/docker-for-windows/#kubernetes)). As an alternative, use [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [Kind](https://kind.sigs.k8s.io/).  

As we walk through the tutorial we'll cover some Kubernetes concepts and how they relate to Docker Compose and _Kev_.

These will be explained under a **Kube Notes** heading.

Finally, we'll use the term "Compose" to mean "Docker Compose".  

### Create your docker-compose config 

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

### Preparing for Kubernetes

Compose and Kubernetes address different problems.

_Compose_, helps you wire, develop and run your application locally as a set of services. It's super for development.

_Kubernetes_, however, is designed to help you run your application in a highly available mode using clusters and service replications. It is production grade.

Describing Compose services as a Kubernetes application requires an extra layer that translates concepts from one to the other.

Furthermore, on Kubernetes, you will share and promote your app through different stages. These stages require varying settings that should ideally be captured from the start. 
     
_Kev_ will help you with all the above! So let's get cracking.

#### Compose + Kev 

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
- Started tracking the `docker-compose.yaml` file as the _Source application manifest_.
- Inferred configuration details from the `docker-compose.yaml` file.
- Assigned sensible defaults for any config it couldn't infer.
- Created `local` (useful for testing on our own machine) and `staging` (useful for testing on a remote machine) _Compose environment overrides_.

It has also generated three files:
- `kev.yaml`, a manifest that describes our _Source application manifest_ and _Compose environment overrides_.
- `docker-compose.kev.*.yaml`, two files to represent our _Compose environment overrides_.

##### Manifest: kev.yaml

The `kev.yaml` manifest file confirm a successful `init`,
```yaml
compose:
  - docker-compose.yaml
environments:
  local: docker-compose.kev.local.yaml
  stage: docker-compose.kev.stage.yaml
...
```

##### Compose environment overrides: docker-compose.kev.*.yaml

The created `local` and `stage` _Compose environment overrides_ are currently identical.

The `labels` section for each service enable you to control how the app runs on Kubernetes. See the [configuration reference](docs/reference/config-params.md) to understand how these affect deployments.

We'll be adjusting these values soon per target environment. For now, they look as below,

```yaml
version: "3.7"
services:
  wordpress:
    labels:
      kev.service.type: LoadBalancer
      kev.workload.image-pull-policy: IfNotPresent
      kev.workload.liveness-probe-command: '["CMD", "echo", "Define healthcheck command for service wordpress"]'
      kev.workload.liveness-probe-disabled: "false"
      kev.workload.liveness-probe-initial-delay: 1m0s
      kev.workload.liveness-probe-interval: 1m0s
      kev.workload.liveness-probe-retries: "3"
      kev.workload.liveness-probe-timeout: 10s
      kev.workload.max-cpu: "0.5"
      kev.workload.cpu: "0.1"
      kev.workload.max-memory: 500Mi
      kev.workload.memory: 10Mi
      kev.workload.replicas: "1"
      kev.workload.rolling-update-max-surge: "1"
      kev.workload.service-account-name: default
      kev.workload.type: Deployment
```

### Moving to Kubernetes

Admittedly, our `wordpress` app is very basic, it only starts a `wordpress` container.

However, all the translation wiring is now in place, so let's run it on Kubernetes!

#### Generate Kubernetes manifests

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
- Has re-introspected our _Source application manifest_.
- Has NOT detected any config changes that need to be applied to our _Compose environment overrides_.
- Has generated manifests to enable our app to run in both a `local` and `stage` mode.

We're now ready to run our app on Kubernetes!

#### Running on Kubernetes
 
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

### Add a DB service 

Let's wire in a database to make our basic `wordpress` app more useful.  

In this case this means adding a `db` service backed by a `mysql` container to our Compose config.     

#### Update Compose config  

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

#### Re-sync Kubernetes

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

#### Fix db service discovery

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

### Run more replicas

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
      ...
      kev.workload.replicas: "5"
      ...
      ...
``` 

#### Re-sync Kubernetes

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

Check the [configuration reference](docs/reference/config-params.md) if you want to configure other params.

### Conclusion

We have successfully moved a `wordpress` app from a local Docker Compose development flow to a connected multi-environment Kubernetes setup.

_Kev_ facilitated all the heavy lifting. It enabled us to easily iterate on and manage our target environments.

We also have an understanding of the **gotchas** we can face when moving from Compose to Kubernetes.

All the generated manifests can be tracked in source control and shared in a team context.

Finally, you can find the artefacts for this tutorial here: [wordpress-mysql example](examples/wordpress-mysql/README.md). 

## Configuration

As mentioned in the [Quickstart](#quickstart) section above, the environment specific configuration lives in a set of docker-compose override files. Each environment override file holds simplified Kubernetes configuration parameters for each of the application components.

Project components (aka services) are configured via a set of labels attached to them, and optionally environment variables section which allows for localised adjustments - the same exact way you'd control those in a regular docker-compose file.

Volumes come with their own set of labels to control Kubernetes storage specific parameters.

See the [configuration reference](docs/reference/config-params.md) for details.

## Similar tools

_Kev_ is inspired by the simple, easy to use and well adopted Docker Compose specification, as well as several other tools in the Kubernetes manifests generation and templating space such as Kompose, Ksonnet and Kustomize, to name a few.

Each of the solutions above, however, come with their own set of challenges and are lacking in various areas. Some have been discontinued, some see very few contributions or updates, others require a great deal of prior Kubernetes expertise to find them useful.

_Kev_ bridges the gaps in the existing tooling, helping developers familiar with Docker & Compose to easily get up and running with Kubernetes.

## Contributing to Kev

We welcome any contributions from the community! Have a look at our [contribution](CONTRIBUTING.md) guide for more information on how to get started. If you use _Kev_, find it useful, or are generally interested in improving Developer Experience on Kubernetes then please let us know by **Starring** and **Watching** this repo. Thanks!

## Roadmap

See our [Roadmap][roadmap] for details about our plans for the project.

## License

Copyright (c) 2020 [Appvia Ltd](https://appvia.io)

This project is distributed under the [Apache License, Version 2.0](./LICENSE).

[roadmap]: https://github.com/appvia/kev/issues