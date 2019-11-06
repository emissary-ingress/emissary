# Access Control
---
Ambassador Edge Stack's `FilterPolicy` custom resource definition (CRD) gives you fine-grained control over filters. Since authentication and access control is implemented in specific filters, the `FilterPolicy` CRD can be used for access control as well.



## Authentication Policy 
A `rule` for the `FilterPolicy` CRD is a set of hosts, paths, and filters that indicate which filters should be applied to a given path or host.

### Rule Configuration Values
| Value     | Example    | Description |
| -----     | -------    | -----------                  |
| `host`    | `*`, `foo.com` | the Host that a given rule should match |
| `path`    | `/foo/url/`    | the URL path that a given rule should match to |
| `filters`  | `name: keycloak`       | the name of a given filter to be applied|

The wildcard `*` is supported for both `path` and `host`.

### Examples
The following policy shows how the `filter` named `keycloak` is applied to requests to `/httpbin/headers`, while requests to `/httpbin/ip` are public.

```
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: httpbin-policy
  namespace: default
spec:
  rules:
    - host: "*"
      path: /httpbin/ip
      filters: null # make this path public
    - host: "*"
      path: /httpbin/headers
      filters:
        - name: keycloak
```

#### Multiple Domains

In this example, the `foo-keycloak` filter is used for requests to `foo.bar.com`, while the `example-auth0` filter is used for requests to `example.com`. This configuration is useful if you are hosting multiple domains in the same cluster.

```
apiVersion: getambassador.io/v1beta1
kind: Policy
metadata:
  name: multi-domain-policy
spec:
  rules:
  - host: foo.bar.com
    path: *
    filters:
      - name: foo-keycloak
  - host: example.com
    path: *
    filters:
      - name: example-auth0
```

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
