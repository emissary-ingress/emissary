---
description: "Edge Stack uses the Mapping resource to map a resource, like a URL prefix, to a Kubernetes service or web service."
---

import Alert from '@material-ui/lab/Alert';

# The Mapping Resource

Edge Stack is designed around a [declarative, self-service management model](../../concepts/gitops-continuous-delivery). The core resource used to support application development teams who need to manage the edge with Edge Stack is the Mapping resource. At its core a Mapping resource maps a URL path (or prefix) to a service (either a Kubernetes service or a web service.

##  Example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  myservice-mapping
spec:
  prefix: /myservice/
  service: myservice
```

| Name | Type | Description |
| :--- | :--- | :--- |
| `metadata.name` | String | Indentifies the Mapping. |
| `spec.prefix` | String | The URL prefix identifying your resource. [See below](#resources) on how Edge Stack handles resources. |
| `spec.service` | String | The service handling the resource.  If a Kuberentes service, it must include the namespace (in the format `service.namespace`) if the service is in a different namespace than Edge Stack. [See below](#services) on service name formatting.|

Here's another example using a web service that maps requests to `/httpbin/` to `http://httpbin.org`:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin-mapping
spec:
  prefix: /httpbin/
  service: http://httpbin.org
```

## Applying a Mapping Resource

A Mapping resource can be managed using the same workflow as any other Kubernetes resources (like a Service or Deployment). For example, if the above Mapping is saved into a file called `httpbin-mapping.yaml`, the following command will apply the configuration directly to Edge Stack:

```
kubectl apply -f httpbin-mapping.yaml
```

<Alert severity="info" variant="outlined">For production use, the general recommended best practice is to store the file in a version control system and apply the changes with a continuous deployment pipeline. <a href="../../concepts/gitops-continuous-delivery">The Ambassador Operating Model</a> provides more detail.</Alert>

## Resources

To Edge Stack, a resource is a group of one or more URLs that all share a common prefix in the URL path. For example, these URLs all share the `/resource1/` path prefix, so `/resource1/` can be considered a single resource:

* `https://ambassador.example.com/resource1/foo`
* `https://ambassador.example.com/resource1/bar`
* `https://ambassador.example.com/resource1/baz/zing`

On the other hand, these URLs share only the prefix `/` -- you _could_ tell Edge Stack to treat them as a single resource, but it's probably not terribly useful.

* `https://ambassador.example.com/resource1/foo`
* `https://ambassador.example.com/resource2/bar`
* `https://ambassador.example.com/resource3/baz/zing`

Note that the length of the prefix doesn't matter; a prefix like `/v1/this/is/my/very/long/resource/name/` is valid.

Also note that Edge Stack does not actually require the prefix to start and end with `/` -- however, in practice, it's a good idea. Specifying a prefix of `/man` would match all of the following, which probably is not what was intended:

* `https://ambassador.example.com/man/foo`
* `https://ambassador.example.com/mankind`
* `https://ambassador.example.com/man-it-is/really-hot-today`

## Services

Edge Stack routes traffic to a service. A service is defined as `[scheme://]service[.namespace][:port]`.  Everything except for the service is optional.

- `scheme` can be either `http` or `https`; if not present, the default is `http`.
- `service` is the name of a service (typically the service name in Kubernetes or Consul); it is not allowed to contain the `.` character.
- `namespace` is the namespace in which the service is running. Starting with Edge Stack 1.0.0, if not supplied, it defaults to the namespace in which the Mapping resource is defined. The default behavior can be configured using the [Module resource](../../running/ambassador). When using a Consul resolver, `namespace` is not allowed.
- `port` is the port to which a request should be sent. If not specified, it defaults to `80` when the scheme is `http` or `443` when the scheme is `https`. Note that the [resolver](../../running/resolvers) may return a port in which case the `port` setting is ignored.

<Alert severity="info" variant="outlined">While using <code>service.namespace.svc.cluster.local</code> may work for Kubernetes resolvers, the preferred syntax is <code>service.namespace</code>.</Alert>


## Extending Mappings

Mapping resources support a rich set of annotations to customize the specific routing behavior.  Here's an example service for implementing the [CQRS pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/cqrs) (using HTTP):

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  cqrs-get
spec:
  prefix: /cqrs/
  method: GET
  service: getcqrs
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  cqrs-put
spec:
  prefix: /cqrs/
  method: PUT
  service: putcqrs
```

## Best Practices for Configuration

Edge Stack's configuration is assembled from multiple YAML blocks which are managed by independent application teams. This implies that certain best practices should be followed.

#### Edge Stack's configuration should be under version control.

While you can always read back the Edge Stack's configuration from Kubernetes or its diagnostic service, the Edge Stack will not do versioning for you.

#### Edge Stack tries to not start with a broken configuration, but it's not perfect.

Gross errors will result in the Edge Stack refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to map a resource to the wrong service, or use the wrong `rewrite` rules. Edge Stack can't detect that on its own, although its [diagnostic service](../../running/diagnostics/) can help you figure it out.

#### Be careful of mapping collisions.

If two different developers try to map `/myservice/` to something, this can lead to unexpected behavior. Edge Stack's [canary deployment](../../using/canary/) logic means that it's more likely that traffic will be split between them than that it will throw an error -- again, the diagnostic service can help you here.

#### Unless specified, mapping attributes cannot be applied to any other resource type.

## <img class="os-logo" src="../../../images/logo.png"/> What's Next?

Mappings are a powerful resource and essential to managing edge traffic. Continue reading on [advanced mapping configurations](../mappings), or other specific features like [circuit breakers](../circuit-breakers/), [rate limiting](../rate-limits), and [redirects](../redirects).