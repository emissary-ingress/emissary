# Okta

1. Create an an OIDC application

   **Note:** If you have a [standard Okta account](https://www.okta.com) you must first navigate to your Okta Org's admin portal (step 1). [Developer accounts](https://developer.okta.com) can skip to step 2.
   
   - Go to your org and click `Admin` in the top right corner to access the admin portal
   - Select `Applications`
   - Select `Add Application`
   - Choose `Web` and `OpenID Connect`. Then click `Create`.
   - Give it a name, enter the URL of your Ambassador load balancer in `Base URIs` and the callback URL `{AMBASSADOR_URL}/.ambassador/oauth2/redirection-endpoint` as the `Login redirect URIs`

2. Copy the `Client ID` and `Client Secret` and use them to fill in the `ClientID` and `Secret` of you Okta OAuth `Filter`.

3. Get the `audience` configuration

   - Select `API` and `Authorization Servers`
   - You can use the default `Authorization Server` or create your own.
   - If you are using the default, the `audience` of your Okta OAuth `Filter` is `api://default`
   - The value of the `authorizationURL` is the `Issuer URI` of the `Authorization Server`

## Configure Filter and FilterPolicy

Configure your OAuth `Filter` and `FilterPolicy` with the following:


   ```yaml
   ---
   apiVersion: getambassador.io/v2
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
           - name: okta_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
               - "profile"
   ```

**Note:** Scopes `openid` and `profile` are required at a minimum. Other scopes can be added to the `Authorization Server`
