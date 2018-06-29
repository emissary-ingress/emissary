## Envoy Override

It's possible that your situation may strain the limits of what Ambassador can do. The `envoy_override` attribute is provided for cases we haven't predicted: any object given as the value of `envoy_override` will be inserted into the Envoy `Route` synthesized for the given mapping. For example, you could enable Envoy's `auto_host_rewrite` by supplying:

```yaml
envoy_override:
  auto_host_rewrite: True
```

Here is another example of using `envoy_override` to set Envoy's [connection retries](https://www.envoyproxy.io/docs/envoy/latest/api-v1/route_config/route.html#retry-policy):

```
envoy_override:
   retry_policy:
     retry_on: connect-failure
     num_retries: 4
```

Note that `envoy_override` has the following limitations:

* It is restricted to Envoy v1 configuration only
* It only supports adding information to Envoy routes, and not clusters
* It cannot change any element already synthesized in the mapping

These limitations will be addressed in future releases of Ambassador.