# Ambassador Pro

Ambassador Pro is a set of paid add-on modules to the Ambassador open source API Gateway. The first functionality that will be available in Ambassador Pro is authentication. If you're interested in being an early adopter of Ambassador Pro, contact hello@datawire.io.

## Installation

Note: Ambassador Pro currently installs as an independent service. In the future, we expect to support a sidecar deployment model.

1. Install Ambassador.
2. Clone this repository; you'll need the YAML configuration.
   
   ```
   git clone https://github.com/datawire/ambassador-pro
   ```

2. In the `ambassador-pro.yaml`, configure the `AUTH0_DOMAIN` and `AUTH0_AUDIENCE` environment variables based on your Auth0 configuration. (You'll need to create a custom API if you haven't already.)
   * The AUTH0_DOMAIN is your Auth0 domain, e.g., foo.auth0.com.
   * AUTH0_AUDIENCE is listed on the API page https://manage.auth0.com/#/apis
   * Configure the `namespace` field appropriately for the `ClusterRoleBinding`, and is the namespace where your Ambassador and Ambassador Pro service is deployed.
3. Deploy the authentication service: `kubectl apply -f ambassador-pro.yaml`.
4. Create the policy CRD: `kubectl apply -f policy-crd.yaml`.

## Quick Start / Example

Authentication policies are managed using the `policy` CRD. Note that this CRD is in alpha state, and the schema is *subject to change*.

In this quick start, we'll create a route to the public httpbin.org service and manage access to the service.

1. Deploy the `httpbin` service: `kubectl apply -f httpbin.yaml`.
2. Verify that the httpbin service works correctly, e.g,

   ```
   $ curl http://$AMBASSADOR_IP/httpbin/ip
   {
    "origin": "35.205.31.151"
   }
   $ curl http://$AMBASSADOR_IP/httpbin/user-agent
   {
    "user-agent": "curl/7.54.0"
   }
   ```

3. Deploy the sample security policy: `kubectl apply -f httpbin-policy.yaml`. This policies gives public access to `/httpbin/ip` but restricts access to `/httpbin/user-agent`.
4. Test this out:

   ```
   $ curl http://$AMBASSADOR_IP/httpbin/ip
   {
    "origin": "35.205.31.151"
   }
   $ curl http://$AMBASSADOR_IP/httpbin/user-agent
   {"message":"unauthorized"}
   ```

5. Get a JWT from Auth0. To do this, click on APIs, then the custom API you're using for the Ambassador Authentication service, and then the Test tab.

   ```
   $ curl --header 'authorization: Bearer eyeJdfasdf...' http://$AMBASSADOR_IP/httpbin/user-agent
   {
     "user-agent": "curl/7.54.0"
   }
   ```

## Configuration

Ambassador Authentication supports creating arbitrary rules for authentication. By default, Ambassador Authentication returns unauthenticated requests. Each rule can have the following values:

| Value     | Example    | Description |
| -----     | -------    | -----------                  |
| `host`    | "*", "foo.com" | the Host that a given rule should match |
| `path`    | "/foo/url/"    | the URL path that a given rule should match to |
| `public`  | true           | a boolean that indicates whether or not authentication is required; default false |
| `scopes`  | "read:contacts" | the rights that need to be granted in a given API |

The following is a complete example of a policy:

```
apiVersion: stable.datawire.io/v1beta1
kind: Policy
metadata:
  name: httpbin-policy
spec:
  # everything defaults to private; you can create rules to make stuff
  # public, and you can create rules to require additional scopes
  # which will be automatically checked
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