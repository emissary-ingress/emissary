# Upgrading to Ambassador Pro from Open Source Ambassador

<div class="ambassador-pro-banner">
All of the functionality previously available in Ambassador Pro is now available for free with the
Ambassador Edge Stack Community Edition! <a href="/editions">Click here to learn more →</a>
</div>

If you are already using Ambassador open-source, upgrading to using Ambassador Pro is straight-forward. In this demo, we will walk-through integrating Ambassador Pro into your currently running Ambassador instance and show how quickly you can secure your APIs with JWT authentication.

Information about open-source code used in Ambassador Pro can be found in `/*.opensource.tar.gz` files in each Docker image.

## 1. Clone the Ambassador Pro Configuration Repository

Ambassador Pro is a module that communicates with Ambassador, exposing the various Pro services to Ambassador. Ambassador Pro is typically deployed as a sidecar service to Ambassador, allowing for it to communicate with Ambassador locally. While this is the recommended deployment topology, for evaluation purposes it is simpler to deploy Ambassador Pro as a separate service in your Kubernetes cluster.

We provide a reference architecture to demonstrate how easy it is to use the services provided by Ambassador Pro.

```
git clone https://github.com/datawire/pro-ref-arch
```

## 2. License Key

Copy `env.sh.example` to `env.sh`, and add your specific license key to the `env.sh` file.

**Note:** Ambassador Pro will not start without a valid license key.

## 3. Deploy Ambassador Pro

Deploy Ambassador Pro using the `Makefile` in the root of the `pro-ref-arch` directory.

```
cd pro-ref-arch

make apply-upgrade-to-pro
```

This `make` command will use `kubectl` to deploy Ambassador Pro alongside your Ambassador deployment. It will also redeploy the httpbin and QoTM services which are used for demo purposes.

Verify that Ambassador Pro is running:

```
kubectl get pods | grep ambassador

ambassador-79494c799f-vj2dv             2/2     Running   0          1h
ambassador-pro-6545769c68-vnnzz         1/1     Running   1          23h
ambassador-pro-redis-6db64c5685-4k8fn   1/1     Running   0          23h
```

By default, Ambassador Pro uses ports 8500-8503.  If for whatever
reason those assignments are problematic (perhaps you [set
`service_port`](../../reference/running/#running-as-non-root) to one of
those), you can set adjust these by setting environment variables:

| Purpose                        | Variable         | Default |
| ---                            | ---              | ---     |
| Filtering AuthService (gRPC)   | `APRO_AUTH_PORT` | 8500    |
| RateLimitService (gRPC)        | `GRPC_PORT`      | 8501    |
| RateLimitService debug (HTTP)  | `DEBUG_PORT`     | 8502    |
| RateLimitService health (HTTP) | `PORT`           | 8503    |

If you have deployed Ambassador with
[`AMBASSADOR_NAMESPACE`, `AMBASSADOR_SINGLE_NAMESPACE`](../../reference/running/#namespaces), or
[`AMBASSADOR_ID`](../../reference/running/#ambassador_id)
set, you will also need to set them in the Pro container.

**Note:** Ambassador Pro will replace your current `AuthService` implementation. Remove your current `AuthService` annotation before deploying Ambassador Pro. If you would like to keep your current `AuthService`, remove the `AuthService` annotation from the `ambassador-pro.yaml` file.

## 4. Configure JWT Authentication

Now that you have Ambassador Pro running, we'll show a few features of Ambassador Pro. We'll start by configuring Ambassador Pro's JWT authentication filter.

```
make apply-jwt
```

This will configure the following `FilterPolicy`:

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: httpbin-filterpolicy
  namespace: default
spec:
  # everything defaults to private; you can create rules to make stuff
  # public, and you can create rules to require additional scopes
  # which will be automatically checked
  rules:
  - host: "*"
    path: /jwt-httpbin/*
    filters:
    - name: jwt-filter
  - host: "*"
    path: /httpbin/*
    filters: null
```


We'll now test Ambassador Pro with the `httpbin` service. First, curl to the `httpbin` URL This URL is public, so it returns successfully without an authentication token.

```
$ curl -k https://$AMBASSADOR_URL/httpbin/ip # No authentication token
{
  "origin": "108.20.119.124, 35.194.4.146, 108.20.119.124"
}
```

Send a request to the `jwt-httpbin` URL, which is protected by the JWT filter. This URL is not public, so it returns a 401.

```
$ curl -i -k https://$AMBASSADOR_URL/jwt-httpbin/ip # No authentication token
HTTP/1.1 401 Unauthorized
content-length: 58
content-type: text/plain
date: Mon, 04 Mar 2019 21:18:17 GMT
server: envoy
```

Finally, send a request with a valid JWT to the `jwt-httpbin` URL, which will return successfully.

```
$ curl -k --header "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." https://$AMBASSADOR_IP/jwt-httpbin/ip
{
  "origin": "108.20.119.124, 35.194.4.146, 108.20.119.124"
}
```

**Important:** Many modules in the reference architecture assume HTTPS. You will need to adjust the `cURL` requests if your Ambassador installation is not secured with TLS.

## 5. Configure Additional Ambassador Pro Services

Ambassador Pro has many more features such as rate limiting, OAuth integration, and more.

### Enabling Rate limiting

For more information on configuring rate limiting, consult the [Advanced Rate Limiting Tutorial](../advanced-rate-limiting) for information on configuring rate limits.

### Enabling Single Sign-On

 For more information on configuring the OAuth filter, see the [Single Sign-On with OAuth and OIDC](../oauth-oidc-auth) documentation.

### Enabling Service Preview

Service Preview requires a command-line client, `apictl`. For instructions on configuring Service Preview, see the [Service Preview tutorial](../../docs/dev-guide/service-preview).

### Enabling Consul Connect integration

Ambassador Pro's Consul Connect integration is deployed as a separate Kubernetes service. For instructions on deploying Consul Connect, see the [Consul Connect integration guide](../consul).
