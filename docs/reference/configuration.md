# Ambassador Configuration

Ambassador is configured in a declarative fashion, using YAML manifests to describe the state of the world. As with Kubernetes, Ambassador's manifests are identified with `apiVersion`, `kind`, and `name`. The current `apiVersion` is `ambassador/v0`; currently-supported `kind`s are:

- [`Module`](modules) manifests configure things with can apply to Ambassador as a whole. For example, the `ambassador` module can define listener ports, and the `tls` module can configure TLS termination for Ambassador.

- [`AuthService`](modules#authservice) manifests configures the external authentication service[s] that Ambassador will use.

- [`Mapping`](mappings) manifests associate REST _resources_ with Kubernetes _services_. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

## Ambassador Configuration

Ambassador assembles its configuration from YAML blocks that may be stored:

- as `annotations` on Kubernetes `service`s (this is the recommended technique);
- as data in a Kubernetes `ConfigMap`; or
- as files in Ambassador's local filesystem.

The data contained within each YAML block is the same no matter where the blocks are stored, and multiple YAML documents are likewise supported no matter where the blocks are stored.

### Running Ambassador Within Kubernetes

When you run Ambassador within Kubernetes:

1. At startup, Ambassador will look for the `ambassador-config` Kubernetes `ConfigMap`. If it exists, its contents will be used as the baseline Ambassador configuration.
2. Ambassador will then scan Kubernetes `service`s in its namespace, looking for `annotation`s named `getambassador.io/config`. YAML from these `annotation`s will be merged into the baseline Ambassador configuration.
3. Whenever any services change, Ambassador will update its `annotation`-based configuration.
4. The baseline configuration, if present, will **never be updated** after Ambassador starts. To effect a change in the baseline configuration, use Kubernetes to force a redeployment of Ambassador.

**Note:** the baseline configuration is not required. It is completely possible - indeed, recommended - to use _only_ `annotation`-based configuration.

### Running Ambassador Within a Custom Image

You can also run Ambassador by building a custom image that contains baked-in configuration:

1. All the configuration data should be collected within a single directory on the filesystem.
2. At image startup, run `ambassador config $configdir $envoy_json_out` where
   - `$configdir` is the path of the directory containing the configuration data, and
   - `$envoy_json_out` is the path to the `envoy.json` to be written.

In this usage, Ambassador will not look for `annotation`-based configuration, and will not update any configuration after startup.

### Best Practices for Configuration

Ambassador's configuration is assembled from multiple YAML blocks, to help enable self-service routing and make it easier for multiple developers to collaborate on a single larger application. This implies a few things:

- Ambassador's configuration should be under version control.

    While you can always read back Ambassador's configuration from `annotation`s or its diagnostic service, it's far better to have a master copy under git or the like. Ambassador doesn't do any versioning of its configuration.

- Be aware that Ambassador tries to not start with a broken configuration, but it's not perfect.

    Gross errors will result in Ambassador refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to e.g. map a resource to the wrong service, or use the wrong `rewrite` rules. Ambassador can't detect that on its own, although its diagnostic pages can help you figure it out.

- Be careful of mapping collisions.

    If two different developers try to map `/user/` to something, this can lead to unexpected behavior. Ambassador's canary-deployment logic means that it's more likely that traffic will be split between them than that it will throw an error -- again, the diagnostic service can help you here.

## Namespaces

Ambassador supports multiple namespaces within Kubernetes. To make this work correctly, you need to set the `AMBASSADOR_NAMESPACE` environment variable in Ambassador's container. By far the easiest way to do this is using Kubernetes' downward API (this is included in the YAML files from `getambassador.io`):

```yaml
        env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace          
```

Given that `AMBASSADOR_NAMESPACE` is set, Ambassador [mappings](#mapping) can operate within the same namespace, or across namespaces. **Note well** that mappings will have to explictly include the namespace with the service to cross namespaces; see the [mapping](#mappings) documentation for more information.

If you only want Ambassador to only work within a single namespace, set `AMBASSADOR_SINGLE_NAMESPACE` as an environment variable.

### Modifying Ambassador's Underlying Envoy Configuration

Ambassador uses Envoy for the heavy lifting of proxying.

If you wish to use Envoy features that aren't (yet) exposed by Ambassador, you can use your own custom config template. To do this, create a templated `envoy.json` file using the Jinja2 template language. Then, use this template as the value for the key `envoy.j2` in your ConfigMap. This will then replace the [default template](https://github.com/datawire/ambassador/tree/master/ambassador/templates).

Please [contact us on Gitter](https://gitter.im/datawire/ambassador) for more information if this seems necessary for a given use case (or better yet, submit a PR!) so that we can expose this in the future.
