## Redirects

To effect an HTTP 301 `Redirect`, the `Mapping` **must** set `host_redirect` to `true`, with `service` set to the host to which the client should be redirected:

```yaml
apiVersion: ambassador/v0
kind:  Mapping
name:  redirect_mapping
prefix: /redirect/
service: httpbin.org
host_redirect: true
```

Using this `Mapping`, a request to `http://$AMBASSADOR_URL/redirect/` will result in an HTTP 301 `Redirect` to `http://httpbin.org/redirect/`.

The `Mapping` **may** also set `path_redirect` to change the path portion of the URL during the redirect:

```yaml
apiVersion: ambassador/v0
kind:  Mapping
name:  redirect_mapping
prefix: /redirect/
service: httpbin.org
host_redirect: true
path_redirect: /ip
```

Here, a request to `http://$AMBASSADOR_URL/redirect/` will result in an HTTP 301 `Redirect` to `http://httpbin.org/ip`. As always with Ambassador, attention paid to the trailing `/` on a URL is helpful!
