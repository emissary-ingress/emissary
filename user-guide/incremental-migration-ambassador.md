# Incremental migration

Many organizations are migrating from monolithic applications to a microservices-based system, and an API gateway like Ambassador Edge Stack can help with this transition.

## Traffic Shadowing and Canarying

The primary benefit of using an API gateway within the development stage of a project is the ability to deploy your application or service to production and “hide” it — i.e. not expose the endpoints to end-users. A gateway can block traffic to a new endpoint, or simply not expose the endpoints publicly. Some gateways can also be configured to route only permitted traffic to a new endpoint, either via security policies or request header metadata. This allows you to test your walking skeleton application deployed into the real environment. This is more likely to give you results that are highly correlated within an actual live release — you can’t get a more production-like environment than production itself!

## Test and QA: Shadowing and Shifting

A modern API gateway can help with testing on many levels. As mentioned previously, we can deploy a service — or a new version of a service — into production, hide this deployment via the gateway, and run acceptance and nonfunctional tests here (e.g. load tests and security analysis). This is invaluable in and of itself, but we can also use a gateway to “shadow” (duplicate) real production traffic to the new version of the service and hide the responses from the user. This allows you to learn how this service will perform under realistic use cases and load.

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
