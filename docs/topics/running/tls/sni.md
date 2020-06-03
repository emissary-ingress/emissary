# Server Name Indication (SNI)

Ambassador supports serving multiple `Host`s behind a single IP address, each
with their own certificate. 

This is as easy to do as creating a `Host` for each domain or subdomain you 
want Ambassador to serve, getting a certificate for each, and telling 
Ambassador which `Host` the route should be created for.

The example below configures two `Host`s and assigns routes to them.

## Configuring a `Host`

The `Host` resources lets you separate configuration for each distinct domain 
and subdomain you plan on serving behind Ambassador.

Let's start by creating a simple `Host` and providing our own certificate in 
the `host-cert` secret.

```yaml
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: host-cert
```

Now lets, create a second `Host` for a different domain we want to serve behind
Ambassador. This second `Host` we can use Ambassador Edge Stack's automatic TLS
to get a certificate from Let's Encrypt.

```yaml
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: foo-host
spec:
  hostname: host.foo.com
  acmeProvider:
    email: julian@example.com
```

We now have two `Host`s with two different certificates.

## Configuring Routes

Now that we have two domains behind Ambassador, we can create routes for either
or both of them.

We do this by setting the `host` attribute of a `Mapping` to the domain the
`Mapping` should be created for. 

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin
spec:
  prefix: /httpbin/
  service: httpbin.org:80
  host_rewrite: httpbin.org
  host: host.example.com
```
Will create a `/httpbin/` endpoint for `host.example.com`
```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  mockbin
spec:
  prefix: /foo/
  service: foo-service
  host: host.foo.com
```
Will create a `/foo/` endpoint for `host.foo.com`

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: frontend
spec:
  prefix: /bar/
  service: bar-endpoint
```
Will create a `/bar/` endpoint for all `Host`s.