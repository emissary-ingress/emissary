# Upgrading Ambassador

Since Ambassador's configuration is entirely stored in annotations or a `ConfigMap`, no special process is necessary to upgrade Ambassador. If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating the following `kubectl apply` commands.

First determine if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled.

If RBAC is enabled:
```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

If RBAC is not enabled:
```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

This will trigger a rolling upgrade of Ambassador.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum, you'll need to change the pulled `image` for the Ambassador container and redeploy.

## Upgrading to 0.50.0
Ambassador 0.50.0 adds the v1 API. While all v0 API objects will still work in Ambassador 0.50.0, it is recommended you upgrade them to v1 so you can take advantage of the new features being added to Ambassador.

### Mapping
The v1 `Mapping` object gives you the ability to configure Ambassador to [add response headers](/reference/add_response_headers) and [bypass the external authorization service](/reference/mappings#mapping-configuration).

Upgrading from v0 to v1 is as simple as changing the `apiVersion` in your `Mapping` definition. 

**Note:** The `rate_limits` attribute is replaced by `labels` in the v1 API. This change is required by Ambassador 0.50.0 See the [rate limiting tutorial](/user-guide/rate-limiting-tutorial#v1-api) and [rate limits](/reference/rate-limits/) documentation for more information. 

### AuthService
The v1 AuthService API adds the ability to forward the request body to your external authorization service. It also allows you to define which headers are allowed to and from the authorization service. More information on these changes can be found in the [authentication plugin](/reference/services/auth-service) documentation.

### TracingService
The v1 `TracingService` adds more configuration options of the [Zipkin driver](/reference/services/tracing-service#zipkin-driver-configurations). 

Upgrading from v0 to v1 is as simple as changing the `apiVersion` in your `TracingService` definition. 

### RateLimitService
There is no difference between the v0 and v1 `RateLimitService`  API at this time. Upgrading is as simple as changing the `apiVersion` in the `Mapping` definition.

**Note:** Ambassador 0.50.0 requires the `rate_limits` `Mapping` attribute be replaced by `labels`. See the [rate limiting tutorial](/user-guide/rate-limiting-tutorial#v1-api) and [rate limits](/reference/rate-limits/) documentation for more information. 

### Module
There is no difference between the v0 and v1 `Module` API at this time. Upgrading is as simple as changing the `apiVersion` in the `Mapping` definition.
