---
weight: 9
title: Quick start guide
---

# Quickstart

- `tako init` - identifies a project's Compose Kubernetes source files and creates Compose environment overrides.
- `tako render` - detects, applies any config changes and generates deployment manifests.
- `tako patch` - patches deployment manifests by replacing images for specified services.
- `tako help` - run it if you're a little lost.

## Initialise project

Run the following command within your project directory:

```sh
$ tako init
```

This identifies the default `docker-compose.yaml` and (if present) `docker-compose.override.yaml` files in your project directory. They will be used as the source of truth for your application deployment in Kubernetes.

Also, it creates an implicit sandbox `dev` environment and its Compose override file.

Here's another example. It uses an alternate `docker-compose` file with `stage` & `prod` environments:

```sh
$ tako init -f my-docker-compose.yaml -e stage -e prod
```

It makes use of,
- `-f` flag, to specify an alternate filename.
- `-e` flags, to specify different deployment environments.

Creating the files below in your project directory:

```sh
├── docker-compose.env.dev.yaml         # dev sandbox Compose environment override file
├── docker-compose.env.prod.yaml        # prod Compose environment override file
├── docker-compose.env.stage.yaml       # stage Compose environment override file
├── appmeta.yaml                        # Tako project manifest
├── ...
```

Here's what happened, Tako has,
- Inferred the configuration details already present in your compose Kubernetes deployment sources.
- Assigned sensible defaults for any config it couldn't infer.
- Created Compose overrides files for the `dev`, `prod` and `stage` environments.

That's it, your Tako project is now ready!

From now on it can,
- Detect edits in your source compose file.
- Apply any related config changes to your compose environment overrides.
- Generate deployment manifests.

You can now customise your deployment targets by altering values in the relevant Compose environment override file.

## Generate Kubernetes manifests

We now need to generate manifests based on your Docker Compose config and environments. You'll use these manifests to deploy your app to Kubernetes.

Run the following command from your project root:

```sh
$ tako render
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

## Patch Kubernetes manifests (optional / CI/CD)

There may be a need to patch the generated manifests, e.g to replace images for specific services.

This step is usually required when you're using a CI/CD pipeline to build and push images to a container registry.

To ensure that your deployments use up-to-date images, you may use the `tako patch` command to replace images in your manifests.

Run the following command:

```sh
$ tako patch -d /path/to/my/k8s/manifest -i web=myweb:tag1 -i db=mydb:tag2
```

Note that,
- `-d` flag, specifies the directory containing the Kubernetes manifests to patch.
- `-i` flag(s), specifies the service name and image to replace it with.
- `-o` [optional] flag, specifies the output directory for the patched manifests (if not specified, the patched manifests will be overwritten in the input directory).

The command above replaces the image for the `web` service with `myweb:tag1` and the image for the `db` service with `mydb:tag2`.

Ensure that the service names match the ones in your source K8s file(s).

**Note:** The only resource kinds that `patch` command might affect are: Deployment, StatefulSet and DaemonSet.

### How can I deploy the app to Kubernetes?

To deploy your app onto Kubernetes,
- Ensure you can access a running Kubernetes installation, either locally (e.g. [Docker Desktop](https://docs.docker.com/desktop/), [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/), etc...) or remotely.
- Use [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) to apply the manifests.

In this example, we deploy the `stage` environment:

```sh
# deploys your app with stage settings onto the default namespace
$ kubectl apply -f k8s/stage
```

### Other deployment tooling

With Tako, you can use any Kubernetes deployment tool or framework you're familiar with, e.g `skaffold`, `tilt`, etc...

Check our [Roadmap][roadmap] for upcoming planned integrations.
