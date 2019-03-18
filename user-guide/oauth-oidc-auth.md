# Configuring OAuth/OIDC Authentication

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes for single sign-on with an external identity providers (IDP). Ambassador Pro has been tested with Keycloak and Auth0, although other OAuth/OIDC-compliant identity providers should work. Please contact us on [Slack](https://d6e.co/slack) if you have questions about IDPs not listed below.

## 1. Configure an OAuth2 filter

First, configure an OAuth2 filter for your identity provider. For information on how to configure your IDP, see the IDP configuration section below.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: auth_filter
  namespace: default
spec:
  OAuth2:
    authorizationURL: PROVIDER_URL ## URL where Ambassador Pro can find OAuth2 descriptor
    clientURL: AMBASSADOR_URL ## URL your IDP will redirect back to. Typically the same as the requests host.
    audience: AUDIENCE ## OIDC Audience
    clientID: CLIENT_ID ## OAuth2 client from your IDP
    secret: CLIENT_SECRET ## Secret used to access OAuth2 client
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f oauth-filter.yaml`.

## 2. Create a Filter Policy

Once we have a properly configured OAuth2 filter, create a FilterPolicy that applies the filter.

```yaml
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
          arguments:
            scopes:
            - "scope1"
            - "scope2"
```

Save the configuration to a file and apply it to the cluster: `kubectl apply -f httpbin-filter-policy.yaml`. For more information about filters and filter policies, consult the [filter reference](/reference/filter-reference).

## IDP Configuration
You will need to configure your IDP to handle authentication requests. The way to do this varies by IDP.

### Auth0
With Auth0 as your IDP, you will need to create an `Application` to handle authentication requests from Ambassador Pro.

1. Navigate to Applications and Select "CREATE APPLICATION"

  ![](/images/create-application.png)

2. In the pop-up window, give the application a name (this will be the `authorizationURL` in your `Filter`) and create a "Machine to Machine App"

  ![](/images/machine-machine.png)

3. Select the Auth0 Management API. Grant any scopes you may require. (You may grant none.) 

  ![](/images/scopes.png)
  
4. In your newly created application, click on the Settings tab, add the Domain and Callback URLs for your service and ensure the "Token Endpoint Authentication Method" is set to `Post`. The default YAML installation of Ambassador Pro uses `/callback` for the URL, so the values should be the domain name that points to Ambassador, e.g., `example.com/callback` and `example.com`.

  ![](/images/Auth0_none.png)

5. Update the Auth0 `Filter` and `FilterPolicy`. You can get the `ClientID` and `secret` from your application settings:

   ![](/images/Auth0_secret.png)

   The `audience` is the API Audience of your Auth0 Management API:

   ![](/images/Auth0_audience.png)

   ```yaml
   ---
   apiVersion: getambassador.io/v1beta2
   kind: Filter
   metadata:
     name: auth0_filter
     namespace: default
   spec:
     OAuth2:
       authorizationURL: https://datawire-ambassador.auth0.com
       clientURL: https://datawire-ambassador.com
       audience: https://datawire-ambassador.auth0.com/api/v2/
       clientID: fCRAI7svzesD6p8Pv22wezyYXNg80Ho8
       secret: CLIENT_SECRET
   ```

   ```yaml
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
           - name: auth0_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
   ```

  **Note:** By default, Auth0 requires the `openid` scope. 

### Keycloak

With Keycloak as your IDP, you will need to create a `Client` to handle authentication requests from Ambassador Pro. The below instructions are known to work for Keycloak 4.8.

1. Under "Realm Settings", record the "Name" of the realm your client is in. This will be needed to configure your `authorizationURL`.

2. Create a new client: navigate to Clients and select `Create`. Use the following settings: 
   - Client ID: Any value (e.g. `ambassador`); this value will be used in the `clientID` field of the Keycloak filter
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
   - Name: Any string. This is just a name for the Mapper
   - Mapper Type: select "Audience"
   - Included Client Audience: select from the dropdown the name of your Client. This will be used as the `audience` in the Keycloak `Filter`.

8. Click Save.

9. Configure client scopes as desired in "Client Scopes" (e.g. `offline_access`). It's possible to setup Keycloak to not use scopes by removing all of them from "Assigned Default Client Scopes". 
   
   **Note:** All "Assigned Default Client Scopes" must be included in the `FilterPolicy` scopes argument. 

10. Update the Keycloak `Filter` and `FilterPolicy`

   ```yaml
   ---
   apiVersion: getambassador.io/v1beta2
   kind: Filter
   metadata:
     name: keycloak_filter
     namespace: default
   spec:
     OAuth2:
       authorizationURL: https://{KEYCLOAK_URL}/auth/realms/{KEYCLOAK_REALM}
       clientURL: https://datawire-ambassador.com
       audience: ambassador
       clientID: ambassador
       secret: CLIENT_SECRET
   ```

   ```yaml
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
           - name: keycloak_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "offline_access"
   ```

### Okta

1. Create an an OIDC application

   - Select `Applications`
   - Select `Add Application`
   - Choose `Web` and click next
   - Give it a name, enter the URL of your Ambassador load balancer in `Base URIs` and the callback URL `{AMBASSADOR_URL}/callback` as the `Login redirect URIs`

2. Copy the `Client ID` and `Client Secret` and use them to fill in the `ClientID` and `Secret` of you Okta OAuth `Filter`.

3. Get the `audience` configuration

   - Select `API` and `Authorization Servers`
   - You can use the default `Authorization Server` or create your own.
   - If you are using the default, the `audience` of your Okta OAuth `Filter` is `api://default`
   - The value of the `authorizationURL` is the `Issuer URI` of the `Authorization Server`

4. Configure your OAuth `Filter` and `FilterPolicy`

   ```yaml
   ---
   apiVersion: getambassador.io/v1beta2
   kind: Filter
   metadata:
     name: okta_filter
     namespace: default
   spec:
     OAuth2:
       authorizationURL: https://{OKTA_DOMAIN}.okta.com/oauth2/default
       clientURL: https://datawire-ambassador.com
       audience: api://default
       clientID: CLIENT_ID
       secret: CLIENT_SECRET
   ```

   ```yaml
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
           - name: okta_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
               - "profile"
   ```

   **Note:** Scopes `openid` and `profile` are required at a minimum. Other scopes can be added to the `Authorization Server`

### Cloud Foundry User Account and Authentication Service (UAA)

1. Create an OIDC Client

   ```shell
   uaac client add ambassador --name ambassador-client --scope openid --authorized_grant_types authorization_code,refresh_token --redirect_uri {AMBASSADOR_URL}/callback --secret CLIENT_SECRET
   ```

   **Note:** Change the value of `{AMBASSADOR_URL}` with the IP or DNS of your Ambassador load balancer.

2. Configure you OAuth `Filter` and `FilterPolicy`

   Use the id (ambassador) and secret (CLIENT_SECRET) from step 1 to configure the OAuth `Filter`.

   ```yaml
   ---
   apiVersion: getambassador.io/v1beta2
   kind: Filter
   metadata:
     name: uaa_filter
     namespace: default
   spec:
     OAuth2:
       authorizationURL: {UAA_DOMAIN}/{ZONE}
       clientURL: https://datawire-ambassador.com
       audience: {UAA_DOMAIN}
       clientID: ambassador
       secret: CLIENT_SECRET
   ```
   **Note:** The `authorizationURL` and `audience` are the same for UAA configuration. 

   ```yaml
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
           - name: uaa_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
   ```

   **Note:** The `scopes` field was set when creating the client in step 1. You can add any scopes you would like when creating the client.


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
