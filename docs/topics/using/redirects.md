# Redirects

Ambassador Edge Stack can perform 3xx redirects on `Mapping`s to a different host, with various options to redirect the path and to return a different 3xx response code instead of the default 301.

## Schema

| Name | Type | Description |
| --- | --- | --- |
| `spec.host_redirect` | Boolean | This is *required* to be set to `true` to use any type of redirect, otherwise the request will be proxied instead of redirected. |
| `spec.path_redirect` | String | Optional, changes the path for the redirect. |
| `spec.prefix_redirect` | String | Optional, matches the `prefix` value and replaces it with the `prefix_redirect` value. |
| `spec.regex_redirect` | String | Optional, uses regex to match and replace the `prefix` value. |
| `spec.redirect_response_code` | Integer | Optional, changes the response code from the default 301, valid values are 301, 302, 303, 307, and 308. |
| `spec.config.  x_forwarded_proto_redirect` | Boolean | Redirect only the originating HTTP requests to HTTPS, allowing the originating HTTPS requests to pass through. |
| `spec.config.  use_remote_address` | Boolean | Required to be set to `false` to use the `x_forwarded_proto_redirect` feature. |

## Examples

### Basic Redirect

To effect any type of HTTP `Redirect`, the `Mapping` *must* set `host_redirect` to `true`, with `service` set to the host to which the client should be redirected:

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /redirect/
  service: httpbin.org
  host_redirect: true
```

Using this `Mapping`, a request to `http://$AMBASSADOR_URL/redirect/` will be redirected to `http://httpbin.org/redirect/`.

> As always with Ambassador Edge Stack, the trailing `/` on any URL with a
`Mapping` is required!

### Path Redirect

The `Mapping` may optionally also set additional properties to customize the behavior of the HTTP redirect response.

To also change the path portion of the URL during the redirect, set `path_redirect`:

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /redirect/
  service: httpbin.org
  host_redirect: true
  path_redirect: /ip
```

Here, a request to `http://$AMBASSADOR_URL/redirect/` will be redirected to `http://httpbin.org/ip/`. 

### Prefix Redirect

To change only a prefix of the path portion of the URL, set `prefix_redirect`:

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /redirect/path/
  service: httpbin.org
  host_redirect: true
  prefix_redirect: /ip
```

Now, a request to `http://$AMBASSADOR_URL/redirect/path/` will  be redirected to `http://httpbin.org/ip/`. The prefix `/redirect/path/` was matched and replaced with `/ip/`.

### Regex Redirect

`regex_redirect` matches a regular expression to replace instead of a fixed prefix.
[See more information about using regex with Edge Stack](../rewrites/#regex_rewrite).

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /foo/
  service: httpbin.org
  host_redirect: true
  regex_redirect:
    pattern: '/foo/([0-9]*)/list'
    substitution: '/bar/\1'
```
A request to `http://$AMBASSADOR_URL/foo/12345/list` will be redirected to 
`http://$AMBASSADOR_URL/bar/12345`.

### Redirect Response Code

To change the HTTP response code return by the Edge Stack, set `redirect_reponse_code`. If this is not set, 301 is returned by default. Valid values include 301, 302, 303, 307, and 308.  This
can be used with any type of redirect.

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /ip/
  service: httpbin.org
  host_redirect: true
  redirect_response_code: 302
```

A request to `http://$AMBASSADOR_URL/ip/` will result in an HTTP 302 redirect to `http://httpbin.org/ip`.

### X-FORWARDED-PROTO Redirect

In cases when TLS is being terminated at an external layer 7 load balancer, then you would want to redirect only the originating HTTP requests to HTTPS, and let the originating HTTPS requests pass through.

This distinction between an originating HTTP request and an originating HTTPS request is done based on the `X-FORWARDED-PROTO` header that the external layer 7 load balancer adds to every request it forwards after TLS termination.

To enable this `X-FORWARDED-PROTO` based HTTP to HTTPS redirection, add a `x_forwarded_proto_redirect: true` field to `ambassador Module`'s configuration. Note that when this feature is enabled `use_remote_address` MUST be set to false.

An example configuration is as follows -

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    x_forwarded_proto_redirect: true
    use_remote_address: false
```

Note: Setting `x_forwarded_proto_redirect: true` will impact all your Ambassador Edge Stack mappings. Every HTTP request to Ambassador Edge Stack will only be allowed to pass if it has an `X-FORWARDED-PROTO: https` header.
