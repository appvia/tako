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

* **Simplicity** - Based on the familiar Docker Compose specification. There is no new framework to learn, no new specification to embrace, and vastness of Kubernetes reduced to a limited set of easy to follow configuration parameters. You focus on the app development. Kev will prepare it for deployment in Kubernetes.

* **Multi-environment support** - Parameterisation enabled with the same configuration primitives you're already familiar with. Each defined environment gets its own docker-compose override file, which is there to control the behaviour of your application in Kubernetes in a simple and consistent way.

* **Best practice out of the box** - Best practice is codified and embedded in the translation layer, so you don't have to think about what's required to run your project application on Kubernetes.

* **Secure** - _Kev_ is opinionated about the secret management. At this stage of its relatively short life it delegates that responsibility to the user, to remove the risk of potential uncontrolled secrets leak. No secrets == No leaks!

* **No vendor lock-in** - Because you already use docker-compose, you can keep using it, even if _Kev_ turns out to be not your cup of tea.

* **Easy integrations** - You may use generated Kubernetes manifests with any tool / framework of your choice. Check out our [Skaffold Integration](docs/tutorials/kev-dev-with-skaffold.md).

## Contents

- **[Installation](#installation)**
- **[Quickstart](#quickstart)**
    * **[Initialise project](#initialise-project)**
    * **[Generate Kubernetes manifests](#generate-kubernetes-manifests)**
- **[Tutorials & guides](#tutorials-and-guides)**
- **[Configuration](#configuration)**
- **[Similar tools](#similar-tools)**
- **[Contributing to Kev](#contributing-to-kev)**
- **[Roadmap](#roadmap)**
- **[License](#license)**

## Installation

All you need to get started quickly is the [kev](https://github.com/appvia/kev/releases) binary added to your PATH, and one or more docker compose files.

## Quickstart

- `kev init` - identifies a project's Compose Kubernetes source files and creates Compose environment overrides.
- `kev render` - detects, applies any config changes and generates deployment manifests.
- `kev help` - run it if you're a little lost.

### Initialise project

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

### Generate Kubernetes manifests

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

## Tutorials and guides

- [How does Kev differ from Kompose?](docs/tutorials/how-kev-differs-from-kompose.md)
- [Getting started with Kev](docs/tutorials/getting-started-with-kev.md)
- [Iterate on the app with Kev and Skaffold](docs/tutorials/kev-dev-with-skaffold.md)
- [Simple Node.js app workflow example](docs/tutorials/simple-nodejs-app-workflow.md)

  This is an example of how to use _Kev_ to iterate and deploy a [WordPress Docker Compose application](https://docs.docker.com/compose/wordpress/) onto Kubernetes.

## Configuration

As mentioned in the [Quickstart](#quickstart) section above, the environment specific configuration lives in a set of docker-compose override files. Each environment override file holds simplified Kubernetes configuration parameters for each of the application components.

Project components (aka services) are configured via a set of labels attached to them, and optionally environment variables section which allows for localised adjustments - the same exact way you'd control those in a regular docker-compose file.

Volumes come with their own set of labels to control Kubernetes storage specific parameters.

See the [configuration reference](docs/reference/config-params.md) for details.

## Similar tools

_Kev_ is inspired by the simple, easy to use and well adopted Docker Compose specification, as well as several other tools in the Kubernetes manifests generation and templating space such as Kompose, Ksonnet and Kustomize, to name a few.

See how [Kev differs from Kompose.](docs/tutorials/how-kev-differs-from-kompose.md)

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
