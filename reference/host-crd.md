# The Host CRD

The custom Host resource defines how Ambassador will be visible to the outside world. It collects all the following information in a single configuration resource:

* The hostname by which Ambassador will be reachable
* How Ambassador should handle TLS certificates
* How Ambassador should handle secure and insecure requests
* Which resources to examine for further configuration

The Ambassador Edge Stack automates the creation of TLS certificates via the [Edge Policy Console](../../about/edge-policy-console), which provides HTTPS for your hosts. Note that **in order to have TLS and automatic HTTPS, your host must be an FQDN as specified in the [product requirements](../../user-guide/product-requirements) page.**

A minimal Host resource, using Let’s Encrypt to handle TLS, would be simply:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider:
    email: julian@example.com
```

This Host tells Ambassador to expect to be reached at host.example.com, and to manage TLS certificates using Let’s Encrypt, registering as julian@example.com. Since it doesn’t specify otherwise, requests using cleartext will be automatically redirected to use HTTPS, and Ambassador will not search for any specific further configuration resources related to this Host.

## ACME and TLS Settings

The acmeProvider element in a Host defines how Ambassador should handle TLS certificates:

```yaml
acmeProvider:
  authority: url-to-provider
  email: email-of-registrant
  tlsSecret:
    name: secret-name
```

* In general, the email is mandatory when using ACME: it should be a valid email address that will reach someone responsible for certificate management.
* ACME stores certificates in Kubernetes secrets. The name of the secret can be set using the tlsSecret element; if not supplied, a name will be automatically generated from the hostname and email.
* If the authority is not supplied, the Let’s Encrypt production environment is assumed.
* If the authority is the literal string “none”, TLS certificate management will be disabled. You’ll need to manually create a TLSContext to use for your host in order to use HTTPS.

## Secure and Insecure Requests

A secure request arrives via HTTPS; an insecure request does not. By default, secure requests will be routed and insecure requests will be redirected (using an HTTP 301 response) to HTTPS. The behavior of insecure requests can be overridden using the requestPolicy element of a Host:

```yaml
requestPolicy:
  insecure:
    action: insecure-action
    additionalPort: insecure-port
```

The insecure-action can be one of

  `redirect` (the default): redirect to HTTPS
  `route`: go ahead and route as normal; this will allow handling HTTP requests normally
  `reject`: reject the request with a 400 response

If additionalPort is specified, Ambassador will listen on the specified insecure-port and treat any request arriving on that port as insecure.

## Use Cases and Examples

HTTPS, but redirect HTTP (this is the default case):

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider: <as needed>
requestPolicy:
  insecure:
    action: redirect
```

Since this is the default, the requestPolicy element could also simply be dropped.

HTTPS-only, TLS terminated at Ambassador:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider: <as needed>
requestPolicy:
  insecure:
    action: reject
```

HTTPS-only, TLS terminated at a load balancer:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider: 
    authoriry: none
requestPolicy:
  insecure:
    action: reject
```

This configuration relies on the load balancer to set X-Forwarded-Proto correctly, so that Ambassador can tell insecure requests from secure requests.

HTTP-only:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider: <as needed>
requestPolicy:
  insecure:
    action: route
```

### Split L4 Load Balancer

In this scenario, a L4 load balancer terminates TLS on port 443 and relays that traffic to Ambassador on port 8443, but the load balancer also relays cleartext traffic on port 80 to Ambassador on port 8080. Since the load balancer is at layer 4, it cannot provide X-Forwarded-Proto, so we need to explicitly set port 8080 as insecure:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: minimal-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
requestPolicy:
  insecure:
    action: redirect
    additionalPort: 8080
```


## `Host` CRD

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

## TLS and HTTPS Support

The `requestPolicy` element allows the `Host` CRD to configure actions for both secure and insecure requests based on whether or not `TLSContext` exists and what kind of Load Balancer you have in place.

An example:

```yaml
requestPolicy:
  secure:
    action: route
  insecure:
    action: redirect
    additionalPort: 8080
```

Incoming traffic, regardless of which port it is directed to, will be treated as insecure in order for Ambassador to grab and encrypt it. This sets the action for `requestPolicy.insecure.action` and `requestPolicy.insecure.additionalPort:` as `route`. 
