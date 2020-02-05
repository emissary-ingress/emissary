# The `Host` CRD

The custom `Host` resource defines how the Ambassador Edge Stack will be visible to the outside world. It collects all the following information in a single configuration resource:

* The hostname by which Ambassador will be reachable
* How Ambassador should handle TLS certificates
* How Ambassador should handle secure and insecure requests
* Which resources to examine for further configuration

A minimal Host resource, using Let’s Encrypt to handle TLS, would be:

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

This Host tells Ambassador to expect to be reached at `host.example.com`, and to manage TLS certificates using Let’s Encrypt, registering as `julian@example.com`. Since it doesn’t specify otherwise, requests using cleartext will be automatically redirected to use HTTPS, and Ambassador will not search for any specific further configuration resources related to this Host.

## ACME and TLS Settings

The acmeProvider element in a Host defines how Ambassador should handle TLS certificates:

```yaml
acmeProvider:
  authority: url-to-provider
  email: email-of-registrant
tlsSecret:
  name: secret-name
```

* In general, `email-of-registrant` is mandatory when using ACME: it should be a valid email address that will reach someone responsible for certificate management.
* ACME stores certificates in Kubernetes secrets. The name of the secret can be set using the `tlsSecret` element; if not supplied, a name will be automatically generated from the `hostname` and `email`.
* If the authority is not supplied, the Let’s Encrypt production environment is assumed.
* **If the authority is the literal string “none”, TLS certificate management will be disabled.** You’ll need to manually create a TLSContext to use for your host in order to use HTTPS.

## Secure and Insecure Requests

A **secure** request arrives via HTTPS; an **insecure** request does not. By default, secure requests will be routed and insecure requests will be redirected (using an HTTP 301 response) to HTTPS. The behavior of insecure requests can be overridden using the `requestPolicy` element of a Host:

```yaml
requestPolicy:
  insecure:
    action: insecure-action
    additionalPort: insecure-port
```

The `insecure-action` can be one of:

* `Redirect` (the default): redirect to HTTPS
* `Route`: go ahead and route as normal; this will allow handling HTTP requests normally
* `Reject`: reject the request with a 400 response

If `additionalPort` is specified, Ambassador will listen on the specified `port` and treat any request arriving on that port as insecure.

Some special cases to be aware of here:

* **Case matters in the actions:** you must use e.g. `Reject`, not `reject`.
* The `X-Forwarded-Proto` header is honored when determining whether a request is secure or insecure. If you are running behind a load balancer, make sure that the `X-Forwarded-Proto` header is correctly set by your load balancer!
* ACME challenges with prefix `/.well-known/acme-challenge/` are always forced to be considered insecure, since they are not supposed to arrive over HTTPS.
* Ambassador Edge Stack provides native handling of ACME challenges. If you are using this support, Ambassador will automatically arrange for insecure ACME challenges to be handled correctly. If you are handling ACME yourself - as you must when running Ambassador Open Source - you will need to supply appropriate Host resources and Mappings to correctly direct ACME challenges to your ACME challenge handler.
* In some cases - like the split L4 scenario below - you will need to have a listener on both port 8443 and port 8080, but you will need to not have a TLSContext. The way to do this is to set the service port for Ambassador to 8443 using the Ambassador module, then explicitly set an insecure additionalPort of 8080 in your Host resource.

## Use Cases and Examples

1. The most common use case involves primarily using HTTPS, but redirecting HTTP to HTTPS:

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
        action: Redirect
  ```

  Since this is the default, the `requestPolicy` element could also simply be dropped.

2. HTTPS-only, TLS terminated at Ambassador:

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
        action: Reject
  ```

  We need to make sure to set the `acmeProvider` appropriately for Ambassador to manage certificates for both of the previous cases.

3. HTTPS-only, TLS terminated at a load balancer:

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
        action: Reject
  ```

  This configuration relies on the load balancer to set `X-Forwarded-Proto` correctly, so that Ambassador can tell insecure requests from secure requests. We also need to explicitly set the `acmeProvider` to none, so that Ambassador doesn’t try to do certificate management when it shouldn’t.

4. HTTP-only:

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
        action: Route
  ```

  In this case, the Host resource explicitly requests no ACME handling, then states that insecure requests must be routed instead of redirected.

5. Split L4 Load Balancer: In this scenario, an L4 load balancer terminates TLS on port 443 and relays that traffic to Ambassador on port 8443, but the load balancer also relays cleartext traffic on port 80 to Ambassador on port 8080.

  Since the load balancer is at layer 4, it cannot provide X-Forwarded-Proto, so we need to explicitly set port 8080 as insecure:

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
        action: Redirect
        additionalPort: 8080
  ```  

## `Host` Specification

The Ambassador Edge Stack automates the creation of TLS certificates via the [Edge Policy Console](../../about/edge-policy-console), which provides HTTPS for your hosts. Note that **in order to have TLS and automatic HTTPS, your host must be an FQDN as specified in the [product requirements](../../user-guide/product-requirements) page.**

The Host CRD defines how Ambassador will be visible to the outside world. A minimal Host defines a hostname by which the Ambassador will be reachable, but a Host can also tell an Ambassador how to manage TLS, and which resources to examine for further configuration.

### CRD Specification

The `Host` CRD is formally described by its protobuf specification. Developers who need access to the specification can find it [here](https://github.com/datawire/ambassador/blob/master/api/getambassador.io/v2/Host.proto).
