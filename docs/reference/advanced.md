# Ambassador configuration sequence

When you run Ambassador within Kubernetes:

1. At startup, Ambassador will look for the `ambassador-config` Kubernetes `ConfigMap`. If it exists, its contents will be used as the baseline Ambassador configuration.
2. Ambassador will then scan Kubernetes `service`s in its namespace, looking for `annotation`s named `getambassador.io/config`. YAML from these `annotation`s will be merged into the baseline Ambassador configuration.
3. Whenever any services change, Ambassador will update its `annotation`-based configuration.
4. The baseline configuration, if present, will **never be updated** after Ambassador starts. To effect a change in the baseline configuration, use Kubernetes to force a redeployment of Ambassador.

**Note:** We recommend using _only_ `annotation`-based configuration, so that Ambassador can respond to updates in its environment.

## Modifying Ambassador's Underlying Envoy Configuration

Ambassador uses Envoy for the heavy lifting of proxying.

If you wish to use Envoy features that aren't (yet) exposed by Ambassador, you can use the [`envoy_override` annotation](mappings#using-envoy-override). This annotation lets you add additional configuration for [Envoy routes](https://www.envoyproxy.io/docs/envoy/latest/api-v1/route_config/route.html).

If you need to add additional configuration for Envoy clusters, you will need to use your own custom configuration template. To do this, create a templated `envoy.json` file using the Jinja2 template language, and use this to to  replace the [default template](https://github.com/datawire/ambassador/tree/master/ambassador/templates/envoy.j2). This method is not officially supported -- if you need to do this, please open a GitHub issue and [contact us on Slack](https://d6e.co/slack) for more information if this seems necessary so that we can explore direct Ambassador support for your use case (or, better yet, submit a PR!).

## Configuring Ambassador via a Custom Image

You can also run Ambassador by building a custom image that contains baked-in configuration:

1. All the configuration data should be collected within a single directory on the filesystem.
2. At image startup, run `ambassador config $configdir $envoy_json_out` where
   - `$configdir` is the path of the directory containing the configuration data, and
   - `$envoy_json_out` is the path to the `envoy.json` to be written.

In this usage, Ambassador will not look for `annotation`-based configuration, and will not update any configuration after startup.
