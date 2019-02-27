# Ambassador configuration sequence

When you run Ambassador within Kubernetes:

1. At startup, Ambassador will look for the `ambassador-config` Kubernetes `ConfigMap`. If it exists, its contents will be used as the baseline Ambassador configuration.
2. Ambassador will then scan Kubernetes `service`s in its namespace, looking for `annotation`s named `getambassador.io/config`. YAML from these `annotation`s will be merged into the baseline Ambassador configuration.
3. Whenever any services change, Ambassador will update its `annotation`-based configuration.
4. The baseline configuration, if present, will **never be updated** after Ambassador starts. To effect a change in the baseline configuration, use Kubernetes to force a redeployment of Ambassador.

**Note:** We recommend using _only_ `annotation`-based configuration, so that Ambassador can respond to updates in its environment.

## Configuring Ambassador via a Custom Image

You can run Ambassador by building a custom image that contains baked-in configuration:

1. All the configuration data should be collected within a single directory on the filesystem.
2. At image startup, run `ambassador config $configdir $envoy_json_out` where
   - `$configdir` is the path of the directory containing the configuration data, and
   - `$envoy_json_out` is the path to the `envoy.json` to be written.

In this usage, Ambassador will not look for `annotation`-based configuration, and will not update any configuration after startup.
