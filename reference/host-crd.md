# The Host CRD

The Ambassador Edge Stack automates the creation of TLS certificates via the [Edge Policy Console](../../about/edge-policy-console), which provides HTTPS for your hosts. Note that **in order to have TLS and automatic HTTPS, your host must be an FQDN as specified in the [product requirements](../../user-guide/product-requirements) page.**

The `Host` CRD defines how Ambassador will be visible to the  outside world. A minimal `Host` defines a hostname by which the Ambassador will be reachable, but a Host can also tell an Ambassador how to manage TLS, and which resources to examine for further configuration.

The `Host` CRD is as follows:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    product: aes
  name: hosts.getambassador.io
spec:
  additionalPrinterColumns:
  - JSONPath: .spec.hostname
    name: Hostname
    type: string
  - JSONPath: .status.state
    name: State
    type: string
  - JSONPath: .status.phaseCompleted
    name: Phase Completed
    type: string
  - JSONPath: .status.phasePending
    name: Phase Pending
    type: string
  - JSONPath: .metadata.creationTimestamp
    name: Age
    type: date
  group: getambassador.io
  names:
    categories:
    - ambassador-crds
    kind: Host
    plural: hosts
    singular: host
  scope: Namespaced
  version: v2
  versions:
  - name: v2
    served: true
    storage: true
```

## HostSpec Resource

The Host resource will usually be a Kubernetes CRD, but it could appear in other forms. The `HostSpec` is part of the Host resource and does not change, no matter what form it's in -- when it's a CRD, this is the part in the "spec" dictionary.

| Attribute | Descriptions | Example |
|-------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------|
| repeated string ambassador_id | Common to all Ambassador objects (and optional) | repeated string ambassador_id = 1 |
| int32 generation | Common to all Ambassador objects (and optional) | int32 generation = 2 |
| string hostname | Hostname by which the Ambassador can be reached. | string hostname = 3 |
| k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector selector | Selector by which we can find further configuration. Defaults to hostname=$hostname | k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector selector = 4 |
| ACMEProviderSpec acmeProvider | Specifies who to talk ACME with to get certs. Defaults to Let's Encrypt; if "none", do not try to do TLS for this Host. | ACMEProviderSpec acmeProvider = 5 |
| k8s.io.api.core.v1.LocalObjectReference tlsSecret | Name of the Kubernetes secret into which to save generated certificates. Defaults to $hostname | k8s.io.api.core.v1.LocalObjectReference tlsSecret = 6 |

The attribute `HostTLSCertificateSource` can have the following:
```
  Unknown = 0;
  None    = 1;
  Other   = 2;
  ACME    = 3;
```

### HostState

The attribute `HostState`has a default value of "zero" value but can have the following: 

```
  Initial = 0;
  Pending = 1;
  Ready   = 2;
  Error   = 3;
```

Note that `phaseCompleted` and `phasePending` are valid when `state==Pending` or `state==Error`.


### HostPhase

The attribute `HostPhase` can have the following:
```
  NA                        = 0;
  DefaultsFilled            = 1;
  ACMEUserPrivateKeyCreated = 2;
  ACMEUserRegistered        = 3;
  ACMECertificateChallenge  = 4;
```

### HostStatus

`HostStatus` provides the value for all of the attributes. For example:
```
  HostTLSCertificateSource tlsCertificateSource = 1;
  HostState state = 2;
  HostPhase phaseCompleted = 3;
  HostPhase phasePending = 4;
  string reason = 5;
  ```

If `state==Error`, then `string reason` is valid.

## ACMEProviderSpec

The attribute `ACMEProviderSpec` specifies where ACME should get its TLS certificates. Defaults to `Let's Encrypt`. **If `none`, do not try to do TLS for this Host.** If you expect TLS to work, check that your host is an FQDN as specified in the [product requirements](../../user-guide/product-requirements) page.

An example:

```
  string authority = 1;
  string email = 2;
  k8s.io.api.core.v1.LocalObjectReference privateKeySecret = 3;
  // This is normally set automatically
  string registration = 4;
```
