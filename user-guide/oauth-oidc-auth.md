# Configuring OAuth/OIDC Authentication

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes for single sign-on with an external identity providers (IDP). Ambassador Pro has been tested with Keycloak and Auth0, although other OAuth/OIDC-compliant identity providers should work. Please contact us on [Slack](https://d6e.co/slack) if you have questions about IDPs not listed below.

## 1. Configure an OAuth2 filter

First, configure an OAuth2 filter for your identity provider. For information on how to configure your IDP, see the IDP configuration section below.

```
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: PROVIDER_NAME
  namespace: default
spec:
  OAuth2:
    authorizationURL: PROVIDER_URL
    clientURL: https://ambassador.default.svc.cluster.local
    audience: AUDIENCE
    clientID: CLIENT_ID
    secret: CLIENT_SECRET
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f oauth-filter.yaml`.

## 2. Create a Filter Policy

Once we have a properly configured OAuth2 filter, create a FilterPolicy that applies the filter.

```
---
apiVersion: getambassador.io/v1beta2
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
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f httpbin-filter-policy.yaml`. For more information about filters and filter policies, consult the [filter reference](/reference/filter-reference).

## IDP Configuration
You will need to configure your IDP to handle authentication requests. The way to do this varies by IDP.

### Auth0
With Auth0 as your IDP, you will need to create an `Application` to handle authentication requests from Ambassador Pro.

1. Navigate to Applications and Select "CREATE APPLICATION"

  ![](/images/create-application.png)

2. In the pop-up window, give the application a name and create a "Machine to Machine App"

  ![](/images/machine-machine.png)

3. Select the Auth0 Management API. Grant any scopes you may require. (You may grant none.)

  ![](/images/scopes.png)
  
4. In your newly created application, click on the Settings tab, add the Domain and Callback URLs for your service and ensure the "Token Endpoint Authentication Method" is set to `Post`. The default YAML installation of Ambassador Pro uses `/callback` for the URL, so the values should be the domain name that points to Ambassador, e.g., `example.com/callback` and `example.com`.

  ![](/images/Auth0_none.png)

5. Update the Auth0 Filter with values for `clientID`, `audience`, and `secret`. You can get the `Client ID` and `Client Secret` from your application settings:

![](/images/Auth0_secret.png)

The `audience` is the API Audience of your Auth0 Management API:

![](/images/Auth0_audience.png)

### Keycloak

With Keycloak as your IDP, you will need to create a `Client` to handle authentication requests from Ambassador Pro. The below instructions are known to work for Keycloak 4.8.

1. Create a new client: navigate to Clients and select `Create`. Use the following settings: 
   - Client ID: Any value; this value will be used in the `clientID` field of the Keycloak filter
   - Client Protocol: "openid-connect"
   - Root URL: Leave Blank

3. Click Save.

4. On the next screen configure the following options:
   - Access Type: "confidential"
   - Valid Redirect URIs: `*`

5. Click Save.
6. Navigate to the `Mappers` tab in your Client and click `Create`.
7. Configure the following options:
   - Protocol: "openid-connect".
   - Name: Any value; this value will be used in the `audience` field of the Keycloak filter
   - Mapper Type: select "Audience"
   - Included Client Audience: select from the dropdown the name of your Client.

7. Click Save.

8. Configure client scopes as desired in "Client Scopes". It's possible to setup Keycloak to not use scopes by removing all of them from "Assigned Default Client Scopes".

9. Add the values for `clientID`, `audience`, and `secret` to your Keycloak filter. The value for `secret` can be obtained from the credentials tab.

## Configure Authentication Across Multiple Domains (Optional)
Ambassador Pro supports authentication for multiple domains where each domain is issued its own access token. For example, imagine you're hosting both `domain1.example.com` and `domain2.example.com` on the same cluster. With multi-domain support, users will receive separate authentication tokens for `domain1` and `domain2`.

To configure multi-domain access, you will need to create another authentication endpoint with your IDP (see [Configure your IDP](/user-guide/oauth-oidc-auth/#configure-your-idp)) and create another `Filter` for the new domain.

Example:

```
---
apiVersion: getambassador.io/v1beta1
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
apiVersion: getambassador.io/v1beta1
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

The [filter reference](/reference/filter-reference) covers the specifics of filters and filter policies in much more detail.