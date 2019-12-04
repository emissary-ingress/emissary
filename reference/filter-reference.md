# Filters

Filters are used to extend Ambassador Edge Stack to modify or
intercept an HTTP request before sending to your backend service.  You
may use any of the built-in Filter types, or use the `Plugin` filter
type to run custom code written in the Go programming language, or use
the `External` filter type to call run out to custom code written in a
programming language of your choice.

Filters are created with the `Filter` resource type, which contains global arguments to that filter.  Which Filter(s) to use for which HTTP requests is then configured in [FilterPolicy](/reference/filterpolicy-definition) resources, which may contain path-specific arguments to the filter.

For more information about developing filters, see the [Filter Development Guide](/docs/guides/filter-dev-guide).

## `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource
spec depends on the filter type:

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name:      "string"      # required; this is how to refer to the Filter in a FilterPolicy
  namespace: "string"      # optional; default is the usual `kubectl apply` default namespace
spec:
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value
  FILTER_TYPE:
    GLOBAL_FILTER_ARGUMENTS
```

Currently, Ambassador supports four filter types:

* [External](/reference/external-filter-type)
* [JWT](/reference/jwt-filter-type)
* [OAuth2](/reference/oauth-filter-type)
* [Plugin](/docs/guides/filter-dev-guide#filter-type-plugin)

## Installing self-signed certificates

The `JWT` and `OAuth2` filters speak to other servers over HTTP or HTTPS.  If those servers are configured to speak HTTPS using a
self-signed certificate, attempting to talk to them will result in an error mentioning `ERR x509: certificate signed by unknown authority`. You can fix this by installing that self-signed certificate in to the
Pro container following the standard procedure for Alpine Linux 3.8: Copy the certificate to `/usr/local/share/ca-certificates/` and then run `update-ca-certificates`.  Note that the `amb-sidecar` image sets `USER 1000`, but that `update-ca-certificates` needs to be run as root.

```Dockerfile
FROM quay.io/datawire/ambassador_pro:amb-sidecar-$aproVersion$
USER root
COPY ./my-certificate.pem /usr/local/share/ca-certificates/my-certificate.crt
RUN update-ca-certificates
USER 1000
```

When deploying Ambassador Edge Stack, refer to that custom Docker image,
rather than to `quay.io/datawire/ambassador_pro:amb-sidecar-$aproVersion$`
