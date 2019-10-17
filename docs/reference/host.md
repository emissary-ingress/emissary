# Host Headers

Ambassador supports several different methods for managing the HTTP `Host` header.

## Using `host` and `host_regex`

A mapping that specifies the `host` attribute will take effect _only_ if the HTTP `Host` header matches the value in the `host` attribute. If `host_regex` is `true`, the `host` value is taken to be a regular expression, otherwise an exact string match is required.

You may have multiple mappings listing the same resource but different `host` attributes to effect `Host`-based routing. An example:

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-ui
spec:
  prefix: /
  service: tour1
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-ui2
spec:
  prefix: /
  host: tour.datawire.io
  service: tour2
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-ui3
spec:
  prefix: /
  host: "^tour[2-9]\\.datawire\\.io$"
  host_regex: true
  service: tour3
```

will map requests for `/` to

- the `tour2` service if the `Host` header is `tour.datawire.io`;
- the `toru3` service if the `Host` header matches `^tour[2-9]\\.datawire\\.io$`; and to
- the `tour1` service otherwise.

Note that enclosing regular expressions in quotes can be important to prevent backslashes from being doubled.

## Using `host_rewrite`

By default, the `Host` header is not altered when talking to the service -- whatever `Host` header the client gave to Ambassador will be presented to the service. For many microservices this will be fine, but if you use Ambassador to route to services that use the `Host` header for routing, it's likely to fail (legacy monoliths are particularly susceptible to this, as well as external services). You can use `host_rewrite` to force the `Host` header to whatever value that such target services need.

An example: the default Ambassador configuration includes the following mapping for `httpbin.org`:

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  httpbin
spec:
  prefix: /httpbin/
  service: httpbin.org:80
  host_rewrite: httpbin.org
```

As it happens, `httpbin.org` is virtually hosted, and it simply _will not_ function without a `Host` header of `httpbin.org`, which means that the `host_rewrite` attribute is necessary here.

## `host` and `method`

Internally:

- the `host` attribute becomes a `header` match on the `:authority` header; and
- the `method` attribute becomes a `header` match on the `:method` header.

You will see these headers in the diagnostic service if you use the `method` or `host` attributes.
