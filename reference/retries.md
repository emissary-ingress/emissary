# Automatic Retries

Sometimes requests fail. When these requests fail for transient issues, Ambassador Edge Stack can automatically retry the request.

Retry policy can be set for all Ambassador Edge Stack mappings in the [AmbassadorEdge Stack](/reference/core/ambassador) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings). Generally speaking, you should set retry policy on a per mapping basis. Global retries can easily result in unexpected cascade failures.

## Configuring retries

The `retry_policy` attribute configures automatic retries. The following fields are supported:
```yaml
retry_policy:
  retry_on: <string>
  num_retries: <integer>
  per_try_timeout: <string>
```

### `retry_on`
(Required) Specifies the condition under which Ambassador Edge Stack retries a failed request. The list of supported values is one of: `5xx`, `gateway-error`, `connect-failure`, `retriable-4xx`, `refused-stream`, `retriable-status-codes`. For more details on each of these values, see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.9.0/configuration/http_filters/router_filter#x-envoy-retry-on).

### `num_retries`
(Default: 1) Specifies the number of retries to execute for a failed request.

### `per_try_timeout`
(Default: global request timeout) Specify the timeout for each retry, e.g., `1s`, `1500ms`.

## Examples

A per mapping retry policy:

```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  tour-backend_mapping
prefix: /backend/
service: tour
retry_policy:
  retry_on: "5xx"
  num_retries: 10
```

A global retry policy (not recommended):

```yaml
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  retry_policy:
    retry_on: "retriable-4xx"
    num_retries: 4
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tour-backend_mapping
prefix: /backend/
service: tour
```

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
