# Traffic Shadowing

Traffic shadowing is a deployment pattern where production traffic is asynchronously copied to a non-production service for testing. Shadowing is a close cousin to two other commonly known deployment patterns, [canary releases](../canary) and blue/green deployments. Shadowing traffic has several important benefits over blue/green and canary testing:

* Zero production impact. Since traffic is duplicated, any bugs in services that are processing shadow data have no impact on production.

* Test persistent services. Since there is no production impact, shadowing provides a powerful technique to test persistent services. You can configure your test service to store data in a test database, and shadow traffic to your test service for testing. Both blue/green deployments and canary deployments require more machinery for testing.

* Test the actual behavior of a service. When used in conjunction with tools such as [Twitter's Diffy](https://github.com/twitter/diffy), shadowing lets you measure the behavior of your service and compare it with an expected output. A typical canary rollout catches exceptions (e.g., HTTP 500 errors), but what happens when your service has a logic error and is not returning an exception?

## Shadowing and Ambassador Edge Stack

Ambassador Edge Stack lets you easily shadow traffic to a given endpoint. In Ambassador Edge Stack, only requests are shadowed; responses from a service are dropped. All normal metrics are collected for the shadow services. This makes it easy to compare the performance of the shadow service versus the production service on the same data set. Ambassador Edge Stack also prioritizes the production path, i.e., it will return responses from the production service without waiting for any responses from the shadow service.

![Shadowing](../../doc-images/shadowing.png)

## The `shadow` Mapping

In Ambassador Edge Stack, you can enable shadowing for a given mapping by setting `shadow: true` in your `Mapping`.  One copy proceeds as if the shadowing `Mapping` was not present: the request is handed onward per the `service`(s) defined by the non-shadow `Mapping`s, and the reply from whichever `service` is picked is handed back to the client.

The second copy is handed to the `service` defined by the `Mapping` with `shadow` set. Any reply from this `service` is ignored, rather than being handed back to the client. Only a single `shadow` per resource can be specified (i.e., you can't shadow the same resource to more than 1 additional destination). In this situation, Ambassador  Edge Stack will indicate an error in the diagnostic service, and only one `shadow` will be used. If you need to implement this type of use case, you should shadow traffic to a multicast proxy (or equivalent).

You can shadow multiple different services.

During shadowing, the host header is modified such that `-shadow` is appended.

## Example

The following example may help illustrate how shadowing can be used. This first attribute sets up a basic mapping between the `myservice` Kubernetes service and the `/myservice/` prefix, as expected.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  myservice
spec:
  prefix: /myservice/
  service: myservice.default
```

What if we want to shadow the traffic to `myservice`, and send that exact same traffic to `myservice-shadow`? We can create a new mapping that does this:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  myservice-shadow
spec:
  prefix: /myservice/
  service: myservice-shadow.default
  shadow: true
```

The `prefix` is set to be the same as the first mapping, which tells Ambassador Edge Stack which production traffic to shadow. The destination service, where the shadow traffic is routed, is a *different* Kubernetes service, `myservice-shadow`. Finally, the `shadow: true` attribute actually enables shadowing.

### Shadow traffic weighting

It is possible to shadow a portion of the traffic by specifying the `weight` in the mapping. Eg.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  myservice-shaddow
spec:
  prefix: /myservice/
  service: myservice-shadow.default
  shadow: true
  weight: 10
```

In the example above, only the 10% of the traffic will be forwarded to the shadowing service.
