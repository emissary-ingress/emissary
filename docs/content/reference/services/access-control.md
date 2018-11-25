# Access Control
---
Ambassador Pro authentication is managed with the `Policy` custom resource definition (CRD). This resource allows you to specify which routes should and should not be authenticated by the authentication service. By default, all routes require authentication from the IDP with either a JWT or via a login service. 

## Authentication Policy 
A `rule` for the `Policy` CRD is a set of hosts, paths, and permission settings that indicate which routes require authentication from Ambassador Pro as well as the access rights that particular API needs. The default rule is to require authentication from all paths on all hosts. 

### Rule Configuration Values
| Value     | Example    | Description |
| -----     | -------    | -----------                  |
| `host`    | "*", "foo.com" | the Host that a given rule should match |
| `path`    | "/foo/url/"    | the URL path that a given rule should match to |
| `public`  | true           | a boolean that indicates whether or not authentication is required; default false |
| `scopes`  | "read:test" | the rights that need to be granted in a given API |

### Examples
The following policy is shown in the [OAuth/OIDC Authentication](/user-guide/oauth-oidc-auth#test-the-auth0-application) guide and is used to secure the example `httpbin` service. 

```
apiVersion: stable.datawire.io/v1beta1
kind: Policy
metadata:
  name: httpbin-policy
spec:
  rules:
  - host: "*"
    path: /httpbin/ip
    public: true
  - host: "*"
    path: /httpbin/user-agent/*
    public: false
  - host: "*"
    path: /httpbin/headers/*
    scopes: "read:test"
```
The `Policy` defines rules based on matching the `host` and `path` to a request and refers to the `public` attribute to decide whether or not it needs to be authenticated. Since both `host` and `path` support wildcards, it is easy to configure an entire mapping to need to be authenticated or not. 

```
apiVersion: stable.datawire.io/v1beta
kind: Policy
metadata:
  name: mappings-policy
spec:
  rules:
  - host: "*"
    path: /httpbin/*
    public: true
  - host:
    path: /qotm/*
    public: false
  - host: "*"
    path: /*
    public: false
```
The above `policy` configures Ambassador Pro authentication to

1. Not require authentication for the `httpbin` mapping.
2. Require authentication for the `qotm` mapping.
3. Explicitly express the default requiring authentication for all routes. 

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Policy
metadata:
  name: default-policy
spec:
  rules:
  - host: "*"
    path: /*
    public: true
```
This policy will change the default to not require authentication for all routes. **Note** Rules applied to higher-level paths, e.g. `/qotm/`, will take precedence over ones applied to lower-level paths, e.g `/`.