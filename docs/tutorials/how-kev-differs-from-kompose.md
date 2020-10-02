---
weight: 11
title: How does Kev differ from Kompose?
---

# How does Kev differ from Kompose?

Currently, _Kev_ provides a feature set which is akin to that of the Kompose, however, it differs in its design and the general purpose.

While Kompose focuses on the docker compose conversion to Kubernetes manifests, _Kev_ extends that functionality with easy to follow config convention and additional control parameters. It also introduces a notion of an "environment" and aims at providing other alternative output formats such as Kustomize, OAM and more in the future.

_Kev’s_ scope will grow by integrating with external tooling in order to further improve a local development life cycle on Kubernetes.

Some of the key differences:

**The purpose**

- _Kev's_ goal is to offer multiple conversion formats and integrate with external tooling in order to improve and automate a local app development lifecycle on Kubernetes.

- Kompose, on the other hand, is focused exclusively on compose conversion to K8s manifests.


**Inferred configuration with sensible defaults**

- _Kev_ extracts all useful application config from the source compose files. If the key attributes are not present at source it'll preset the config parameters with sensible defaults automatically. The entire configuration is stored in a dedicated file as a set of labels. This way you know precisely what you can control and the results are guaranteed to be consistent - even if you forget about key compose config parameters. This means you can start iterating on your application stack in docker-compose quickly, with just a few lines of basic config and still get the fully qualified Kubernetes manifests in return.

- In contrast, Kompose relies on a manual addition of docker-compose configuration attributes and/or labels directly in the source input files. This requires pre-existing expertise or additional learning and understanding of docker compose spec. The Kompose conversion outcome is only as good as the input compose files. If you omit attributes or labels that Kompose business logic depends on you won't get the expected or optimal results.

**Clear environment definition**

- _Kev_ auto generates configuration files for each named environment and applies them automatically when rendering the manifests. This way the Kubernetes centric config is cleanly separated from the source compose files and (by convention) you know where to look should further environment adjustments be necessary.

- Kompose, however, accepts any arbitrary combination of compose files as its input. With attributes and config labels scattered across multiple files it's difficult to control the outcome in a simple way. This is OK for a one-off conversion but as there is no standard config convention to follow it gets quite cumbersome when iterating on multi environment configurations.

**Additional output formats**

- _Kev_ aims at providing multiple output formats. Currently, like Kompose, it only supports native Kubernetes manifests, however, formats such as Kustomize, OAM and more are on our [Roadmap][roadmap].

- Kompose only produces native Kubernetes manifests.

**Improved control & Best practice**

- _Kev_ aims at keeping track of all the key control elements which result in the best practice Kubernetes manifests, in line with Kubernetes API changes.

- Kompose doesn't support some of quite useful Kubernetes configuration parameters at the moment. Examples: PodSecurityContext, ServiceAccount, StatefulSet etc.

**Less repetition**

- _Kev_ tracks and analyses source compose files and their changes by default. This helps you focus on your app while Kev keeps your config in sync. There is also no need to repeatadely specify input files either.

- Kompose generates native Kubernetes manifests based on a set of docker compose files which the user selects as an input. This is OK for a one-off conversion, but not ideal when iterating on multi environment configuration as input files must be specified over and over again.

[roadmap]: https://github.com/appvia/kev/issues
