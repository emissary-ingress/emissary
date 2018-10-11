# Customizing Envoy

You may run into a situation where Ambassador may not yet support a specific Envoy feature. Ambassador supports two different methods for these situations.

## `envoy_override`

The `envoy_override` attribute can be used to add specific values to the generated configuration file. Any object given as the value of the attribute will be inserted into the Envoy `Route` for a given mapping. For example, you could enable Envoy's `auto_host_rewrite` by supplying:

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

## Modifying Ambassador's Underlying Envoy Configuration

Ambassador ships with a standard configuration template that is used to generate Envoy configuration. If you need to do more extensive modifications, you can create your own custom configuration template to replace the standard template. To do this, create a templated `envoy.json` file using the Jinja2 template language. Then, use this template as the value for the key `envoy.j2` in your ConfigMap. This will then replace the [default template](https://github.com/datawire/ambassador/tree/master/ambassador/templates).

## File an issue

If you do need to use one of these options, we'd appreciate if you filed a [GitHub issue](https://github.com/datawire/ambassador/issues/) and/or contacted us on [Slack](https://d6e.co/slack) so we can understand the use case, and add support in the future. Better yet, we'll send you a T-shirt if you open a PR!