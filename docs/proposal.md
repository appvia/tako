---
weight: 1
title: Kev proposal
---

# Proposal

To provide solutions to developers to make using Kubernetes easier. To give a developer the least number of steps to take their locally running application and make it consistently deployable into kubernetes environments. Without abstracting them too much from the ecosystem / open source community or reducing operational awareness of their application.

## Current Situation / Assumptions

When a developer works with containers, they usually default to Docker. There are varying reasons developers work with containers:

1. To improve ease of testing by running dependencies / versions in a simple lightweight way
2. To package up their application so it's reusable across many environments and operating systems
3. To make testing / running a local setup easier and quicker, removing the reliance on infrastructure

There are some assumptions we are making about ways of working with containers, these are:

1. A developer looking to adopt containers into their ways of working will use docker and go to the docker documentation as a reference point
2. To test components together, Docker will refer you to use docker compose
3. Because of 1 and 2, a developer will have a docker compose file
4. They will build an application container but best practices around this will vary from team to team
5. They will push their application container to a docker repository (this could be private, public, saas, cloud or on-premise)
6. They will put steps in their CI eventually to repeat the above processes as CI steps

When it comes to then deploying their application inside of Kubernetes, there are additional steps and extra knowledge required. As there is a need to work within the Kubernetes framework, new requirements come into play, the application will now need to:

+ Have an ingress or service endpoint to talk to that routes to their application
+ Have configuration be it environment variables or a configuration file for their application to use
+ Have dependent services, (if required), available to consume for the application to work, within that environment or accessible from the environment
+ Have storage if required for application state
+ Have network policies, (depending on the Kubernetes setup they are deploying to)
+ Have a domain name for CI or others to easily consume to test or access
+ Have a certificate, if encryption is required to consume the application

If developers migrate to Kubernetes, this change can be quite time consuming for a developer to take what is running locally and port it to another framework and repeat this throughout different environments.
