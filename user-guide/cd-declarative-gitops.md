# Continuous Delivery with Declarative Config and DVCS as the Source of Truth

No code provides value to end-users until it is running in production. Many of the architectural patterns that are associated with today's cloud-native applications, such as microservices and serverless, promote looser coupling between components, which enables a faster iteration cycle. However, traditional software continuous integration and delivery mechanisms need to adapt to be able to support this increase in speed.

## Components of Cloud Native Continuous Delivery

[Continuous Delivery](https://continuousdelivery.com/) is the ability to get changes of all types -- including new features, configuration changes, bug fixes, and experiments -- into production and in front of customers safely and quickly in a sustainable way. In a complex and ever-changing cloud-native environment, combining continuous delivery with declarative configuration -- focusing on the what, rather than the how -- is an obvious choice to reduce the burden of describing all of the low-level implementation detail, and also allow orchestration and scheduling frameworks to make optimal second-by-second modifications that wouldn't be practical for a human operator.

By using a distributed version control system (DVCS), like git, to store all code and configuration organizations get many benefits: there can be a single source of truth; engineers can apply contextual comments to code, and create discussion close to the issue being examined; and repetitive (but vital) review and delivery operations can be automated and changes written back to a repository in the form of an audit log.

High-performing organisations combine all three of these approaches to drive success within IT and business. This is supported by conclusions from the "State of DevOps 2018" report, and also the emergence of methodologies like "[GitOps](https://www.weave.works/blog/gitops-operations-by-pull-request)" and the "[Software Defined Delivery Manifesto](https://sdd-manifesto.org/)"

## How Ambassador Edge Stack Supports Continuous Delivery of Cloud Services

As engineering teams take advantage of the ability to iterate faster by using loosely coupled architectures, such as microservices, they frequently find that they require a new workflow in order to expose their services to end-users. With a monolithic application, this was easy: there was typically a single ingress point (via an edge gateway or Application Delivery Controller), and routing additions or updates were simply added into the monolith's codebase. With microservices, there may be multiple points of ingress into a system, and the routing is typically decoupled from any single (monolithic) application -- which is often pushed downstream into the edge gateway. This, in turn, means that the edge gateways become integral to your continuous delivery process.

What is required is a mechanism to decentralize edge configuration and also support the continuous delivery of new changes. Ambassador Edge Stack supports this by allowing routing configuration to be specified in code next to a team's Kubernetes' service definitions.

A microservice development team's workflow, therefore, moves from this deployment pattern when using traditional edge technologies:

1. App developer defines configuration.
2. App developer opens a ticket for operations.
3. Operations team reviews ticket.
4. Operations team initiates infrastructure change management process.
5. Operations team executes change using UI or REST API.
6. Operations team notifies app developer of the change.
7. App developer tests change, and opens a ticket to give feedback to operations if necessary.

To do this deployment pattern with Ambassador Edge Stack:

1. App developer opens Pull Request in Git with proposed edge configuration changes.
2. Operations reviews PR, offers comments and ultimately merges PR.
3. Automated GitOps delivery pipeline applies changes to Ambassador Edge Stack.
4. App developer tests the changes.
