# The `Host` CRD, ACME Support, and External Load Balancer Configuration

The custom `Host` resource defines how the Ambassador Edge Stack will be
visible to the outside world. It collects all the following information in a
single configuration resource:

* The hostname by which Ambassador will be reachable
* How Ambassador should handle TLS certificates
* How Ambassador should handle secure and insecure requests
* Which resources to examine for further configuration
* How Ambassador should handle [Service Preview URLs](../../using/edgectl/#ambassador-edge-stack)

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

This Host tells Ambassador to expect to be reached at `host.example.com`,
and to manage TLS certificates using Let’s Encrypt, registering as
`julian@example.com`. Since it doesn’t specify otherwise, requests using
cleartext will be automatically redirected to use HTTPS, and Ambassador will
not search for any specific further configuration resources related to this
Host.

## ACME and TLS Settings

The `Host` is responsible for high-level TLS configuration in Ambassador.

The are two settings in the `Host` that are responsible for TLS configuration:

- `acmeProvider` defines how Ambassador should handle TLS certificates
- `tlsSecret` tells Ambassador which secret to look for the certificate in

In combination, these settings tell Ambassador how it should manage TLS
certificates.

### ACME Support

The Ambassador Edge Stack comes with built in support for automatic certificate
management using the [ACME protocol](https://tools.ietf.org/html/rfc8555).

It does this by using the `hostname` of a `Host` to request a certificate from
the `acmeProvider.authority` using the `HTTP-01` challenge. After requesting a
certificate, Ambassador will then manage the renewal process automatically.

The `acmeProvider` element of the `Host` configures the Certificate Authority
Ambassador will request the certificate from and the email address that the CA
will use to notify about any lifecycle events of the certificate.

```yaml
acmeProvider:
  authority: url-to-provider
  email: email-of-registrant
```

**Notes on ACME Support:**

* If the authority is not supplied, the Let’s Encrypt production environment is assumed.

* In general, `email-of-registrant` is mandatory when using ACME: it should be
a valid email address that will reach someone responsible for certificate 
management.

* ACME stores certificates in Kubernetes secrets. The name of the secret can be
set using the `tlsSecret` element:
   ```yaml
   acmeProvider:
     email: user@example.com
   tlsSecret:
     name: tls-cert
   ```
   if not supplied, a name will be automatically generated from the `hostname` and `email`.

* Ambassador uses the [`HTTP-01` challenge
](https://letsencrypt.org/docs/challenge-types/) for ACME support:
   - Does not require permission to edit DNS records
   - The `hostname` must be reachable from the internet so the CA can check
   `POST` to an endpoint in Ambassador.
   - Wildcard domains are not supported.

### TLS Configuration

Regardless of if you are using the built in ACME support in the Ambassador Edge
Stack, the `Host` is responsible for reading TLS certificates from Kubernetes
`Secret`s and configuring Ambassador to terminate TLS using those certificates.

If you are using the Open-Source Ambassador API Gateway or choosing to not use
the provided ACME support, you need to tell Ambassador to not request a
certificate with ACME by setting `acmeProvider.authority: none` in the `Host`.

After that you simply point Ambassador at the secret you are using to store
your certificates with the `tlsSecret` field.

The following `Host` will configure Ambassador to read a `Secret` named
`tls-cert` for a certificate to use when terminating TLS.

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: tls-cert
```

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

The `additionalPort` element tells Ambassador to listen on the specified `insecure-port` and treat any request arriving on that port as insecure. **By default, `additionalPort` will be set to 8080 for any `Host` using TLS.** To disable this redirection entirely, set `additionalPort` explicitly to `-1`:

```yaml
requestPolicy:
  insecure:
    additionalPort: -1   # This is how to disable the default redirection from 8080.
```

Some special cases to be aware of here:

* **Case matters in the actions:** you must use e.g. `Reject`, not `reject`.
* The `X-Forwarded-Proto` header is honored when determining whether a request is secure or insecure. For more information, see "Load Balancers, the `Host` Resource, and `X-Forwarded-Proto`" below.
* ACME challenges with prefix `/.well-known/acme-challenge/` are always forced to be considered insecure, since they are not supposed to arrive over HTTPS.
* Ambassador Edge Stack provides native handling of ACME challenges. If you are using this support, Ambassador will automatically arrange for insecure ACME challenges to be handled correctly. If you are handling ACME yourself - as you must when running Ambassador Open Source - you will need to supply appropriate Host resources and Mappings to correctly direct ACME challenges to your ACME challenge handler.

## Load Balancers, the `Host` Resource, and `X-Forwarded-Proto`

In a typical installation, Ambassador runs behind a load balancer. The
configuration of the load balancer can affect how Ambassador sees requests
arriving from the outside world, which can in turn can affect whether Ambassador
considers the request secure or insecure. As such:

- **We recommend layer 4 load balancers** unless your workload includes
  long-lived connections with multiple requests arriving over the same
  connection. For example, a workload with many requests carried over a small
  number of long-lived gRPC connections.
- **Ambassador fully supports TLS termination at the load balancer** with a single exception, listed below.
- If you are using a layer 7 load balancer, **it is critical that the system be configured correctly**:
  - The load balancer must correctly handle `X-Forwarded-For` and `X-Forwarded-Proto`.
  - The `xff_num_trusted_hops` element in the `ambassador` module must be set to the number of layer 7 load balancers the request passes through to reach Ambassador (in the typical case, where the client speaks to the load balancer, which then speaks to Ambassador, you would set `xff_num_trusted_hops` to 1). If `xff_num_trusted_hops` remains at its default of 0, the system might route correctly, but upstream services will see the load balancer's IP address instead of the actual client's IP address.

It's important to realize that Envoy manages the `X-Forwarded-Proto` header such that it **always** reflects the most trustworthy information Envoy has about whether the request arrived encrypted or unencrypted. If no `X-Forwarded-Proto` is received from downstream, **or if it is considered untrustworthy**, Envoy will supply an `X-Forwarded-Proto` that reflects the protocol used for the connection to Envoy itself. The `xff_num_trusted_hops` element, although its name reflects `X-Forwarded-For`, is also used when determining trust for `X-Forwarded-For`, and it is therefore important to set it correctly. Its default of 0 should always be correct when Ambassador is behind only layer 4 load balancers; it should need to be changed **only** when layer 7 load balancers are involved.

## Use Cases and Examples

In the definitions below, "L4 LB" refers to a layer 4 load balancer, while "L7
LB" refers to a layer 7 load balancer.

* [HTTPS-only, TLS terminated at Ambassador, not redirecting cleartext](#https-only-tls-terminated-at-ambassador-not-redirecting-cleartext)
* [HTTPS-only, TLS terminated at Ambassador, redirecting cleartext from port 8080](#https-only-tls-terminated-at-ambassador-redirecting-cleartext-from-port-8080)
* [HTTP-only](#http-only)
* [L4 LB, HTTPS-only, TLS terminated at Ambassador, not redirecting cleartext](#l4-lb-https-only-tls-terminated-at-ambassador-not-redirecting-cleartext)
* [L4 LB, HTTPS-only, TLS terminated at Ambassador, redirecting cleartext from port 8080](#l4-lb-https-only-tls-terminated-at-ambassador-redirecting-cleartext-from-port-8080)
* [L4 LB, HTTP-only](#l4-lb-http-only)
* [L4 LB, TLS terminated at LB, LB speaks cleartext to Ambassador](#l4-lb-tls-terminated-at-lb-lb-speaks-cleartext-to-ambassador)
* [L4 LB, TLS terminated at LB, LB speaks TLS to Ambassador](#l4-lb-tls-terminated-at-lb-lb-speaks-tls-to-ambassador)
* [L4 split LB, TLS terminated at Ambassador](#l4-split-lb-tls-terminated-at-ambassador)
* [L4 split LB, TLS terminated at LB](#l4-split-lb-tls-terminated-at-lb)
* [L7 LB](#l7-lb)

### HTTPS-only, TLS terminated at Ambassador, not redirecting cleartext

  This example is the same with a L4 LB, or without a load balancer. It also covers an L4 LB that terminates TLS, then re-originates TLS from the load balancer to Ambassador.

  In this situation, Ambassador does everything on its own, and insecure requests are flatly rejected.

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

  The `acmeProvider` must be set appropriately for your certificate-management needs; by default, it is set to allow the Ambassador Edge Stack to manage certificates for you. Or, you could set `acmeProvider.authority` to `none` if you want to manage certificates by hand.

  An example using the default `acmeProvider`, ACME with Let's Encrypt:

  ```yaml
  apiVersion: getambassador.io/v2
  kind: Host
  metadata:
    name: acme-lets-encrypt-host
  spec:
    hostname: host.example.com
    requestPolicy:
      insecure:
        action: Reject
  ```

  An example managing certificates by hand:

  ```yaml
  ---
  apiVersion: getambassador.io/v2
  kind: Host
  metadata:
    name: manual-tls-only-host
  spec:
    hostname: foo.example.com
    # Specifying acmeProvider.authority none with a manual tlsSecret.name
    # turns off the ACME client, but leaves TLS enabled.
    acmeProvider:
      authority: none
    tlsSecret:
      name: manual-secret-for-foo
    # The default insecure action is Redirect, which is not what we want.
    requestPolicy:
      insecure:
        action: Reject
  ```

  With the configuration above, the system will look for a TLS secret in `manual-secret-for-foo`, but it will not run ACME for it.

### HTTPS-only, TLS terminated at Ambassador, redirecting cleartext from port 8080

This example is the same for an L4 LB, or without a load balancer at all.

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
        additionalPort: 8080
  ```

  The default for `insecure.action` is `Redirect`, so that line could be removed.

  If you do not set `insecure.additionalPort`, Ambassador won't listen on port 8080 at all. However, with the `Redirect` action still in place, Ambassador will still redirect requests that arrive on port 8443 with an `X-Forwarded-Proto` of `http`.

### HTTP-only

  This example is the same with an L4 LB, or without a load balancer at all.

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

  In this case, the Host resource explicitly requests no ACME handling and no TLS, then states that insecure requests must be routed instead of redirected.

### L4 LB, HTTPS-only, TLS terminated at Ambassador, not redirecting cleartext

  Configure this exactly like [case 1](#https-only-tls-terminated-at-ambassador-not-redirecting-cleartext). Leave `xff_num_trusted_hops` in the `ambassador` module at its default of 0.

### L4 LB, HTTPS-only, TLS terminated at Ambassador, redirecting cleartext from port 8080

  This use case is not supported by Ambassador 1.1.1. It will be supported in a forthcoming release.

### L4 LB, HTTP-only

  Configure this exactly like [case 3](#http-only). Leave `xff_num_trusted_hops` in the `ambassador` module at its default of 0.

### L4 LB, TLS terminated at LB, LB speaks cleartext to Ambassador

  Configure this exactly like [case 3](#http-only), since by the time the connection arrives at Ambassador, it will appear to be insecure. Leave `xff_num_trusted_hops` in the `ambassador` module at its default of 0.

### L4 LB, TLS terminated at LB, LB speaks TLS to Ambassador

  Configure this exactly like [case 1](#https-only-tls-terminated-at-ambassador-not-redirecting-cleartext). Leave `xff_num_trusted_hops` in the `ambassador` module at its default of 0.

  Note that since Ambassador _is_ terminating TLS, managing Ambassador's TLS certificate will be important.

### L4 split LB, TLS terminated at Ambassador

  In this scenario, an L4 load balancer terminates TLS on port 443 and relays that traffic as cleartext to Ambassador on port 8443, but the load balancer also relays cleartext traffic on port 80 to Ambassador on port 8080. (This could also be two L4 load balancers working in concert.)

  Configure this exactly like [case 2](#https-only-tls-terminated-at-ambassador-redirecting-cleartext-from-port-8080). Leave `xff_num_trusted_hops` in the `ambassador` module at its default of 0.

### L4 split LB, TLS terminated at LB

  In this scenario, an L4 load balancer terminates TLS on port 443 and relays that traffic as cleartext to Ambassador on port 8443, but the load balancer also relays cleartext traffic on port 80 to Ambassador on port 8080. (This could also be two L4 load balancers working in concert.)

  This case is not supported in Ambassador 1.1.1. It will be supported in a forthcoming release.

### L7 LB

  In general, L7 load balancers will be expected to provide a correct `X-Forwarded-Proto` header, and will require `xff_num_trusted_hops` set to the depth of the L7 LB stack in front of Ambassador. 

  - `client -> L7 LB -> Ambassador` would require `xff_num_trusted_hops: 1`
  - `client -> L7 LB -> L7 LB -> Ambassador` would require `xff_num_trusted_hops: 2`
  - etc.

  If using an L7 LB, **we recommend that the LB handle TLS termination and redirection of cleartext**. For this use case, you can use a `Host` without TLS, but still turn on redirection as a failsafe:

  ```yaml
  ---
  apiVersion: getambassador.io/v2
  kind: Host
  metadata:
    name: l7-redirection-host
  spec:
    hostname: foo.example.com
    # TLS happens at the LB, so disable it here.
    acmeProvider:
      authority: none
    # The default insecure action is Redirect, which is fine.
  ```

  However, as long as the L7 LB is properly supplying `X-Forwarded-Proto` and `xff_num_trusted_hops` is set correctly, it should be possible to configure Ambassador to handle TLS and redirection of cleartext, by configuring Ambassador as if the L7 LB was not present (cases 1 - 3 above).

  **Again, it is critical that the load balancer correctly supplies `X-Forwarded-Proto`, and that `xff_num_trusted_hops` is set correctly.**

## Service Preview URLs

See [Service Preview and Edge Control](../../using/edgectl/#ambassador-edge-stack) for more information.

## `Host` Specification

The Ambassador Edge Stack automates the creation of TLS certificates via the [Edge Policy Console](../../using/edge-policy-console), which provides HTTPS for your hosts. Note that **in order to have TLS and automatic HTTPS, your host must be an FQDN.**

The Host CRD defines how Ambassador will be visible to the outside world. A minimal Host defines a hostname by which the Ambassador will be reachable, but a Host can also tell an Ambassador how to manage TLS, and which resources to examine for further configuration.

### CRD Specification

The `Host` CRD is formally described by its protobuf specification. Developers who need access to the specification can find it [here](https://github.com/datawire/ambassador/blob/master/api/getambassador.io/v2/Host.proto).
