---
weight: 9
title: Quick start guide
---

# Quickstart

- `kev init` - identifies a project's Compose Kubernetes source files and creates Compose environment overrides.
- `kev render` - detects, applies any config changes and generates deployment manifests.
- `kev help` - run it if you're a little lost.

## Initialise project

Run the following command within your project directory:

```sh
$ kev init
```

This identifies the default `docker-compose.yaml` and (if present) `docker-compose.override.yaml` files in your project directory. They will be used as the source of truth for your application deployment in Kubernetes.

Also, it creates an implicit sandbox `dev` environment and its Compose override file.

Here's another example. It uses an alternate `docker-compose` file with `stage` & `prod` environments:

```sh
$ kev init -f my-docker-compose.yaml -e stage -e prod
```

It makes use of,
- `-f` flag, to specify an alternate filename.
- `-e` flags, to specify different deployment environments.

Creating the files below in your project directory:

```sh
├── docker-compose.kev.dev.yaml         # dev sandbox Compose environment override file
├── docker-compose.kev.prod.yaml        # prod Compose environment override file
├── docker-compose.kev.stage.yaml       # stage Compose environment override file
├── kev.yaml                            # kev project manifest
├── ...
```

Here's what happened, Kev has,
- Inferred the configuration details already present in your compose Kubernetes deployment sources.
- Assigned sensible defaults for any config it couldn't infer.
- Created Compose overrides files for the `dev`, `prod` and `stage` environments.

That's it, your Kev project is now ready!

From now on it can,
- Detect edits in your source compose file.
- Apply any related config changes to your compose environment overrides.
- Generate deployment manifests.

You can now customise your deployment targets by altering values in the relevant Compose environment override file.

## Generate Kubernetes manifests

We now need to generate manifests based on your Docker Compose config and environments. You'll use these manifests to deploy your app to Kubernetes.

Run the following command from your project root:

```sh
$ kev render
```

The command above,
- Detects edits you made to the project's source compose file(s).
- Applies any found config changes to your compose environment overrides.
- Generates kubernetes manifests based on all compose files including environment overrides.
- Generates kubernetes manifests for all environments.

The directory below should now appear in your project directory:

```sh
├── k8s         # stores the Kubernetes manifests for all target deployment environments.
├──── dev       # dev deployment environment
├────── ...     # dev manifests
├──── prod      # prod deployment environment
├────── ...     # prod manifests
├──── stage     # stage deployment environment
├────── ...     # stage manifests
```

Other flag options include,
- `-f` flag, to specify the deployment files format (defaults to `kubernetes`).
- `-s` flag, to render application's manifests to a single file.
- `-d` flag, to specify the output directory for generated manifests (it will contain sub-directories, each for a separate environment name).
- `-e` flag(s), to control which environments to generate the manifests for.

**Note:** Generated manifests should **NOT** be treated as templates as they are fully expanded.

### How can I deploy the app to Kubernetes?

To deploy your app onto Kubernetes,
- Ensure you can access a running Kubernetes installation, either locally (e.g. [Docker Desktop](https://docs.docker.com/desktop/), [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), etc...) or remotely.
- Use [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) to apply the manifests.

In this example, we deploy the `stage` environment:

```sh
# deploys your app with stage settings onto the default namespace
kubectl apply -f k8s/stage
```

### Other deployment tooling

With Kev, you can use any Kubernetes deployment tool or framework you're familiar with, e.g `skaffold`, `tilt`, etc...

Check our [Roadmap][roadmap] for upcoming planned integrations.
