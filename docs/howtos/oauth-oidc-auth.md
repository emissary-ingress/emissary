# Single Sign-On with OAuth & OIDC

Ambassador Edge Stack adds native support for configuring single sign-on with OAuth and OIDC authentication schemes for single sign-on with an external identity provider (IdP). Ambassador Edge Stack has been tested with Keycloak, Auth0, Okta, and UAA although other OAuth/OIDC-compliant identity providers should work. Please contact us on [Slack](https://d6e.co/slack) if you have questions about IdPs not listed below.

## 1. Configure an OAuth2 Filter

First, configure an OAuth2 filter for your identity provider. For information on how to configure your IdP, see the IdP configuration section below.

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: auth_filter
  namespace: default
spec:
  OAuth2:
    authorizationURL: PROVIDER_URL ## URL where Ambassador Edge Stack can find OAuth2 descriptor
    clientURL: AMBASSADOR_URL ## URL your IdP will redirect back to. Typically the same as the requests host.
    audience: AUDIENCE ## OIDC Audience
    clientID: CLIENT_ID ## OAuth2 client from your IdP
    secret: CLIENT_SECRET ## Secret used to access OAuth2 client
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f oauth-filter.yaml`.

## 2. Create a Filter Policy

Once we have a properly configured OAuth2 filter, create a FilterPolicy that applies the filter.

```yaml
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: httpbin-policy
  namespace: default
spec:
  rules:
    - host: "*"
      path: /httpbin/ip
      filters:
        - name: auth_filter ## Enter the Filter name from above
          arguments:
            scopes:
            - "scope1"
            - "scope2"
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f httpbin-filter-policy.yaml`. For more information about filters and filter policies, consult the [filter reference](../../topics/using/filters/).

## IdP Configuration

You will need to configure your IdP to handle authentication requests. The way to do this varies by IdP.

- [Auth0](../sso/auth0)
- [Azure AD](../sso/azure)
- [Google](../sso/google)
- [Keycloak](../sso/keycloak)
- [Okta](../sso/okta)
- [OneLogin](../sso/onelogin)
- [Salesforce](../sso/salesforce)
- [UAA](../sso/uaa)

## Configure Authentication Across Multiple Domains (Optional)

Ambassador Edge Stack supports authentication for multiple domains where each domain is issued its own access token. For example, imagine you're hosting both `domain1.example.com` and `domain2.example.com` on the same cluster. With multi-domain support, users will receive separate authentication tokens for `domain1` and `domain2`.

To configure multi-domain access, you will need to create another authentication endpoint with your IdP and create another `Filter` for the new domain.

Example:

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: domain1-tenant
spec:
  OAuth2:
    - authorizationURL: https://example.auth0.com
      clientURL: http://domain1.example.com
      audience: https://example.auth0.com/api/v2/
      clientId: <APP1_CLIENT_ID>
      secret: <APP1_CLIENT_SECRET>
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: domain2-tenant
spec:
  OAuth2:
    - authorizationURL: https://example.auth0.com
      clientURL: http://domain2.example.com
      audience: https://example.auth0.com/api/v2/
      clientId: <APP2_CLIENT_ID>
      secret: <APP2_CLIENT_SECRET>
```

Create a separate `FilterPolicy` that specifies which specific filters are applied to particular hosts or URLs.

## Further reading

The [filter reference](../../topics/using/filters/) covers the specifics of filters and filter policies in much more detail.
