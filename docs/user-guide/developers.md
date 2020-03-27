# Developer Guide

Ambassador Edge Stack has been designed to be managed by developers and frontline application engineers that are working within an independent product (service) focused teams. This section of the documentation focuses on the core functionality of Ambassador Edge Stack for application developers.

## Why Should Developers Use Ambassador Edge Stack?

The decentralized control plane and ability to locate configuration close to each team's Kubernetes service code enables rapid rollout of new APIs and features, and the ability for developers to manage the deployment, testing, and monitoring in production.

In more detail, Ambassador Edge Stack supports developers in the following ways:

* [Enables publishing a service](../../concepts/developers) publicly without a hand-off to operations
* [Fine-grained control of routing](../../concepts/developers), with support for regex-based routing, host routing, and more
* Support for [gRPC and HTTP/2](../../user-guide/grpc)
* [Testing in production](../../docs/dev-guide/test-in-prod)
* Support for [canarying and shadow traffic](../../docs/dev-guide/canary-release-concepts)
* [Transparent monitoring](../../reference/statistics) of L7 traffic to given services
