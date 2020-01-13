# Auth0

With Auth0 as your IdP, you will need to create an `Application` to handle authentication requests from Ambassador Edge Stack.

1. Navigate to Applications and Select "CREATE APPLICATION"

  ![](../../../doc-images/create-application.png)

2. In the pop-up window, give the application a name and create a "Machine to Machine App"

  ![](../../../doc-images/machine-machine.png)

3. Select the Auth0 Management API. Grant any scopes you may require. (You may grant none.) 

  ![](../../../doc-images/scopes.png)
  
4. In your newly created application, click on the Settings tab, add the Domain and Callback URLs for your service and ensure the "Token Endpoint Authentication Method" is set to `Post`. The default YAML installation of Ambassador Edge Stack uses `/.ambassador/oauth2/redirection-endpoint` for the URL, so the values should be the domain name that points to Ambassador, e.g., `example.com/.ambassador/oauth2/redirection-endpoint` and `example.com`.

  ![](../../../doc-images/Auth0_none.png)
  
  Click Advanced Settings > Grant Types and check "Authorization Code"

## Configure Filter and FilterPolicy


Update the Auth0 `Filter` and `FilterPolicy`. You can get the `ClientID` and `secret` from your application settings:


   ![](../../../doc-images/Auth0_secret.png)

   The `audience` is the API Audience of your Auth0 Management API:

   ![](../../../doc-images/Auth0_audience.png)

   The `authorizationURL` is your Auth0 tenant URL.

   ```yaml
   ---
   apiVersion: getambassador.io/v2
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
           - name: auth0_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
   ```

  **Note:** By default, Auth0 requires the `openid` scope. 


