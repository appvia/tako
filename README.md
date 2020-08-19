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

## Quick Start

All you need to get started quickly is [kev](https://github.com/appvia/kev/releases) binary added to your PATH, and one or more docker compose files.

### Initiate a project

Run the following command from within your project directory to initialise Kev project:

```sh
kev init
```

The command above will auto-detect default `docker-compose.yaml` and `docker-compoe.override.yaml` files (if present) in the project directory, and tracks them as Kubernetes deployment sources.

If you want to point _Kev_ at set of alternative compose files, simply pass them in with `-f` flag. Multiple source compose files can be specified by providing `-f` flag multiple times.

And, to individually control the configuration of your project for deployments in separate target environments, just specify their respective names with `-e` flag, e.g.:

```sh
kev init -f my-docker-compose.yaml -e stage -e prod
```

If no environment has been specified, a default one will be named `dev`.

Once the project has bootstrapped you should see a few new files added to your project tree, similar to the list below:

```sh
├── docker-compose.kev.stage.yaml # stage environment override
├── docker-compose.kev.prod.yaml  # prod environment override
├── kev.yaml                      # kev project manifest
├── ...
```

Your project is now ready and environment specific configuration files generated. _Kev_ will infer useful configuration details already present in the source compose file(s). If this is not possible it'll assume sensible defaults. Users have an easy way to manually alter values directly in the environment specific override file as and when required.

### Render K8s manifests

In order to deploy your project application to Kubernetes you will need to supply _something_ K8s can understand. Currently, _Kev_ only supports Docker Compose conversion to native Kuberentes manifests. The Community might add additional output formats at later stages. See our Roadmap for planned new features.

To render manifests, simply run the following command from your project root:

```sh
kev render
```

`render` generates the project's Kubernetes manifests based on the tracked docker-compose files, using the desired format, specified via `-f` flag (`kubernetes` by default), and selected environments. Note that ALL environments will be rendered by default if none are specified.

You can control which environments to render the manifests for with `-e` flag(s).

You may also specify the output directory for generated manifests with `-d` flag. Note, that specified directory will contain sub-directories, each for a separate environment name for which manifests were generated.

To render application's manifests to a single file, pass in a `-s` flag and you are good to go.

Note that all generated manifests are fully expanded i.e. they should not be treated as templates. (Quick reminder that a specific environment configuration lives in the docker compose override files.)

From this point onward you're free to use whatever tool or framework you are already familiar with, to deploy your project to Kubernetes e.g `kubectl`, `skaffold` etc. Watch our roadmap for details around planned integrations.

## Commands

- `kev init` - initiate project for kev.
- `kev render` - render application manifests for selected environments, according to desired output format.
- `kev help` - run it if you're a little lost.

## Configuration

As mentioned in the [Quick Start](#quick-start) section above, the environment specific configuration lives in a set of docker-compose override files. Each environment override file holds simplified Kubernetes configuration parameters for each of the application components.

Project components (aka services) are configured via a set of labels attached to them, and optionally environment variables section which allows for localised adjustments - the same exact way you'd control those in a regular docker-compose file.

Volumes come with their own set of labels to control Kubernetes storage specific parameters.

See the [configuration reference](docs/reference/config-params.md) for details.

## Similar tools

_Kev_ is inspired by the simple, easy to use and well adopted Docker Compose specification, as well as several other tools in the Kubernetes manifests generation and templating space such as Kompose, Ksonnet and Kustomize, to name a few.

Each of the solutions above, however, come with their own set challenges and are lacking in various areas. Some have been discontinued, some see very few contributions or updates, others require a great deal of prior Kubernetes expertise to find them useful.

_Kev_ bridges the gaps in the existing tooling, helping developers familiar with Docker & Compose to easily get up and running with Kubernetes.

## Contributing to Kev

We welcome any contributions from the community! Have a look at our [contribution](CONTRIBUTING.md) guide for more information on how to get started. If you use _Kev_, find it useful, or are generally interested in improving Developer Experience on Kubernetes then please let us know by **Staring** and **Watching** this repo. Thanks!

## Roadmap

See our [Roadmap](https://github.com/appvia/kev/issues) for details about our plans for the project.

## License

Copyright (c) 2020 [Appvia Ltd](https://appvia.io)

This project is distributed under the [Apache License, Version 2.0](./LICENSE).
