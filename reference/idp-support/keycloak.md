# Keycloak

With Keycloak as your IdP, you will need to create a `Client` to handle authentication requests from Ambassador Edge Stack. The below instructions are known to work for Keycloak 4.8.

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

9. Configure client scopes as desired in "Client Scopes" (e.g. `offline_access`). It's possible to set up Keycloak to not use scopes by removing all of them from `"Assigned Default Client Scopes"`.
   
   **Note:** All "Assigned Default Client Scopes" must be included in the `FilterPolicy` scopes argument.

## Configure Filter and FilterPolicy

Update the Keycloak `Filter` and `FilterPolicy` with the following:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
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
           - name: keycloak_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "offline_access"
   ```
