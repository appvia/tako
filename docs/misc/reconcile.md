---
weight: 99
title: Misc
---

# Misc

## Reconciling project changes

Kev tracks updates made to a project's docker-compose files (files listed in `kev.yaml`).

Kev will specifically monitor the scenarios listed here. And, then apply strategies to manage those scenarios.

### Scenario: source compose file alterations

#### Adding a new service
**Strategy**: extend environments.

#### Removing an existing service.
**Strategy**: override environments.

#### Adding new environment variables.
**Strategy**: extend environments.

#### Removing existing environment variables.
**Strategy**: override environments.

#### Updating values of existing extractable blocks.
**Strategy**: override environments.

#### Explicitly adding new blocks that we extract labels from.
**Strategy**: If values are set to defaults, then override environments.
**Strategy**: If values are NOT set to defaults, then keep environments.

### Scenario: manifest file compose source alterations

#### Adding new compose sources.
**Strategy**: apply all strategies for `Source compose file` alterations.

#### Removing some compose sources.
**Strategy**: apply all strategies for `Source compose file` alterations.

#### Removing all compose sources.
**Strategy**: Do not allow render. (Do nothing to environment files)

### Scenario: manifest file environments alterations

#### Adding new project environments.
**Strategy**: if environment file doesn't exist, create file with init strategy then apply all strategies for `Source compose file` alterations.

#### Removing some project environments.
**Strategy**: Clean up any environment files that are not available in the manifest.

#### Removing all project environments.
**Strategy**: Clean up any environment files that are not available in the manifest.
