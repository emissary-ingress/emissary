# Ambassador Configuration

Ambassador is configured in a declarative fashion, using YAML manifests to describe the state of the world. As with Kubernetes, Ambassador's manifests are identified with `apiVersion`, `kind`, and `name`. The current `apiVersion` is `ambassador/v0`; currently-supported `kind`s are:

- [`Module`](/reference/modules) manifests configure things with can apply to Ambassador as a whole. For example, the `ambassador` module can define listener ports, and the `tls` module can configure TLS termination for Ambassador.

- [`AuthService`](/reference/services/auth-service) manifests configures the external authentication service[s] that Ambassador will use.

- [`RateLimitService`](/reference/services/rate-limit-service) manifests configures the external rate limiting service that Ambassador will use.

- [`TracingService`](/reference/services/tracing-service) manifests configures the external tracing service that Ambassador will use.

- [`Mapping`](/reference/mappings) manifests associate REST _resources_ with Kubernetes _services_. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

## Configuration sources

Ambassador assembles its configuration from YAML blocks that may be stored:

- as `annotations` on Kubernetes `service`s (this is the recommended technique);
- as data in a Kubernetes `ConfigMap`; or
- as files in Ambassador's local filesystem.

The data contained within each YAML block is the same no matter where the blocks are stored, and multiple YAML documents are likewise supported no matter where the blocks are stored.

## Best Practices for Configuration

Ambassador's configuration is assembled from multiple YAML blocks, to help enable self-service routing and make it easier for multiple developers to collaborate on a single larger application. This implies a few things:

- Ambassador's configuration should be under version control.

    While you can always read back Ambassador's configuration from `annotation`s or its diagnostic service, Ambassador will not do versioning for you. Tools like [Forge](https://forge.sh) can help you maintain proper version control for your services' routing configurations.

- Be aware that Ambassador tries to not start with a broken configuration, but it's not perfect.

    Gross errors will result in Ambassador refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to e.g. map a resource to the wrong service, or use the wrong `rewrite` rules. Ambassador can't detect that on its own, although its diagnostic pages can help you figure it out.

- Be careful of mapping collisions.

    If two different developers try to map `/user/` to something, this can lead to unexpected behavior. Ambassador's canary-deployment logic means that it's more likely that traffic will be split between them than that it will throw an error -- again, the diagnostic service can help you here.
    
**Note:** Unless specified, mapping attributes cannot be applied to any other resource type.
