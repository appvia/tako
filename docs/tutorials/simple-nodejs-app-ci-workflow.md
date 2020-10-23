---
weight: 14
title: Kev CI workflow example with a simple Node.js App
---

# CI workflow example with a simple Node.js app  

This example walks through using Kev in a commit based CI pipeline.

You will setup CircleCI to reconcile, render, build, push and deploy to a remote staging environment. 

We assume that you,
- Use [`git`](https://git-scm.com/) for source control and can push to a remote Git repo.
- Have a [CircleCI](https://circleci.com/) account connected to your Git repo.
- Have a [kube-config file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) with access to a remote cluster.

## Get Kev

* Download [Kev](https://github.com/appvia/kev/releases/latest) & add it your `PATH`.

## Prepare for CI

### kube-config file

We will encode our kube-config file into base64.

```shell script
cat ~/path/to/your/kube/config | base64
```

Then create `KUBE_CONFIG`, a [CircleCI Project Environment Variable](https://circleci.com/docs/2.0/env-vars/#setting-an-environment-variable-in-a-project), and store the base64 value there.

### Docker registry

We will also add `DOCKER_USERNAME` and `DOCKER_TOKEN` as CircleCI Project Environment Variables.

These allow us to push our app's image to a secure docker registry. Should that be a private registry then make sure that the remote Kubernetes cluster is able to [pull images from it](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).

### config.yaml

Here is the [CircleCI config file](../../examples/node-app-ci/.circleci/config.yml). It will create a CI pipeline that is triggered by a commit to your Git repo.

It assumes the necessary **CircleCI Environment Variables have been setup**.

Also, it depends on the staging environment that we will be creating shortly. Here's the relevant step that performs the actual deployment,

> CircleCI Deploy step: reconcile and render staging, use Skaffold to build, push and deploy your app to Kubernetes. 
```yaml
...
...
      - run:
          name: Deploy
          command: |
            kev render -e stage
            skaffold run --kubeconfig ${KUBE_CONFIG_FILE} -p staging-env
...
...
```

Commit the latest and push it to your Git repo. 

## Initialise project

> Inside the project directory (`./examples/node-app-ci`) instruct Kev to:
> * create a `staging` based Kubernetes environment configuration.
> * prepare the app for use with [Skaffold](https://skaffold.dev/).

```sh
kev init -e staging --skaffold
```

You will notice the staging environment configuration file has been created:

> Added environment specific override files:
```
... 
|- docker-compose.kev.staging.yaml      # staging env
```

Adjust your Kubernetes `staging` application parameters for each of the components as needed. This is done via Compose [labels](../../docs/reference/config-params.md).

Also, you'll find that Kev has bootstrapped a Skaffold config file (`skaffold.yaml`). If a `skaffold.yaml` file previously existed, then the additional profiles will be added there.

Our CI pipeline will be using this Skaffold config file to power builds, pushes and deployments.  

## Iterate on the application and commit

Iterate on the application as [described here](simple-nodejs-app-workflow.md#iterate-on-the-application).

When your happy commit the latest and push.

## App deployed to staging

The commit will trigger the CI pipeline.

After the `stage` job successfully finishes, inspect that the Node app is running in your remote cluster:

> List Kubernetes application pods for the Node.js app:
```sh
$ kubectl --context <remote_cluster_context> -n default get po

NAME                   READY   STATUS    RESTARTS   AGE
app-69d87ffbc8-wgs6   1/1     Running   0          9s
```

That's it! If you have an [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) configured then navigate to the correct Url.

Alternatively, you can choose to [port-forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) the service and test the app on your localhost. 