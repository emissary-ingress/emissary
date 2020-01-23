# Configuring the Ambassador Edge Stack

The Ambassador Edge Stack is configured in a declarative fashion, using YAML manifests to describe the state of the world. As with Kubernetes, the Ambassador Edge Stack's manifests are identified with `apiVersion`, `kind`, and `name`. The current `apiVersion` is `getambassador.io/v2`; some of the currently-supported `kind`s are:

- [`Module`](../modules) manifests configure things that apply to the Ambassador Edge Stack as a whole. For example, the `ambassador Module` can define listener ports, and the `tls` Module can configure TLS termination for the Ambassador Edge Stack.

- [`AuthService`](../services/auth-service) manifests configure the external authentication service[s] that the Ambassador Edge Stack will use.

- [`RateLimitService`](../services/rate-limit-service) manifests configure the external rate limiting service that Ambassador Edge Stack will use.

- [`TracingService`](../services/tracing-service) manifests configure the external tracing service that the Ambassador Edge Stack will use.

- [`Mapping`](../mappings) manifests associate REST _resources_ with Kubernetes _services_. The Ambassador Edge Stack _must_ have one or more mappings defined to provide access to any services at all.

- [`TLSContext`](../core/tls) manifests control the TLS configuration options for a number of different use cases.

- [`Ingress`](../core/ingress-controller) manifests allows you to use Ambassador as a Kubernetes ingress controller. See the provided documention on configuration with Ambassador, and review the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/) for detailed information on the `Ingress` resource.

- [`LogService`](../services/log-service) manifests configure centralized access logging.

- [`TCPMapping`](../tcpmappings) manifests associate TCP mappings with Kubernetes services.

For an exhaustive list, see the [CRDs](../core/crds/#supported-crds) page.

Note that each of these `kind`s are supported as both annotations and as CRDs.

## Configuration sources

The Ambassador Edge Stack assembles its configuration from YAML blocks that may be stored:

- as Custom Resource Definitions on Kubernetes `service`s (this is the recommended technique);
- as data in a Kubernetes `ConfigMap`; or
- as files in the Ambassador Edge Stack's local filesystem.

The data contained within each YAML block is the same no matter where the blocks are stored, and multiple YAML documents are likewise supported no matter where the blocks are stored.

## Best Practices for Configuration

The Ambassador Edge Stack's configuration is assembled from multiple YAML blocks, to help enable self-service routing and make it easier for multiple developers to collaborate on a single larger application. This implies a few things:

- Ambassador Edge Stack's configuration should be under version control.

    While you can always read back the Ambassador Edge Stack's configuration from `annotation`s or its diagnostic service, the Ambassador Edge Stack will not do versioning for you.

- Be aware that the Ambassador Edge Stack tries to not start with a broken configuration, but it's not perfect.

    Gross errors will result in the Ambassador Edge Stack refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to e.g. map a resource to the wrong service, or use the wrong `rewrite` rules. The Ambassador Edge Stack can't detect that on its own, although its diagnostic pages can help you figure it out.

- Be careful of mapping collisions.

    If two different developers try to map `/user/` to something, this can lead to unexpected behavior. The Ambassador Edge Stack's canary-deployment logic means that it's more likely that traffic will be split between them than that it will throw an error -- again, the diagnostic service can help you here.
    
**Note:** Unless specified, mapping attributes cannot be applied to any other resource type.
