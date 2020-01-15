# The Ambassador Edge Stack Configuration Format

The Ambassador Edge Stack exposes three methods for configuration.

- [Ambassador Custom Resource Definitions (CRDs)](../../reference/core/crds)
- [Kubernetes Service Annotations](../../reference/core/annotations)
- [Ambassador as an Ingress Controller](../../reference/core/ingress-controller)

For most use cases, CRDs are the recommended configuration format since they directly expose all configuration options in a format that allows them to be managed as a separate object by Kubernetes. Because of this, all example configuration in the documentation is presented in CRD format. See [translating CRDs to annotations](../../reference/core/annotations#crd-translation) to translate CRD examples to Service annotations.

Since CRDs are a cluster-wide resource you need admin access to your cluster to install them. If you do not have the necessary cluster permissions to do this using Service annotations or Ingress resources are still available to you.

See the individual documentation for more information.
