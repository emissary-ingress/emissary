# Timeouts

Ambassador Edge Stack enables you to control timeouts in several different ways.

## Request timeout: `timeout_ms`

`timeout_ms` is the timeout for an entire user-level transaction. By default, this is 3000ms. This spans the point at which the entire downstream request has been processed (i.e., end of stream) to the point where the upstream response has been processed. This timeout includes all retries.

## Idle timeout: `idle_timeout_ms`

`idle_timeout_ms` controls how long a connection should remain open when no traffic is being sent through the connection. If not set, Ambassador Edge Stack will wait 5 minutes (300000 milliseconds).

## Connect timeout: `connect_timeout_ms`

`connect_timeout_ms` controls the connection-level timeout for Ambassador Edge Stack to an upstream service.

### Example

The various timeouts are applied onto a `Mapping` resource and can be combined.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  tour-backend_mapping
prefix: /backend/
service: tour
timeout_ms: 4000
idle_timeout_ms: 500000
connect_timeout_ms: 4000
```

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
