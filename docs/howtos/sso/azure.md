# Azure AD

## Set up Azure AD

To use Azure as your IdP, you will first need to register an OAuth application with your Azure tenant.

1. Follow the steps in the Azure documentation [here](https://docs.microsoft.com/en-us/azure/active-directory/develop/v1-protocols-openid-connect-code#register-your-application-with-your-ad-tenant) to register your application. Make sure to select "web application" (not native application) when creating your OAuth application.

2. After you have registered your application, click on `App Registrations` in the navigation panel on the left and select the application you just created.

3. Make a note of both the client and tenant IDs as these will be used later when configuring Ambassador Edge Stack.

4. Click on `Authentication` in the left sidebar.

      - Under `Redirect URIs` at the top, add a `Redirect URI` with the type `Web` and set it to `https://{{AMBASSADOR_URL}}/.ambassador/oauth2/redirection-endpoint`
        
        **Note:** Azure AD requires the redirect endpoint to handle TLS
      - Under `Advanced settings`, make sure the application is issuing `Access tokens` by checking next to the box that says `Access tokens`
      - Under `Supported account types` select whichever option fits your use case

5. Click on `Certificates & secrets` in the left sidebar. Click `+ New client secret` and set the expiration date you wish. Copy the value of this secret somewhere. You will need it when configuring Ambassador Edge Stack.

## Set Up the Ambassador Edge Stack

After configuring an OAuth application in Azure AD, configuring Ambassador Edge Stack to make use of it for authentication is simple.

1. Create an [OAuth Filter](../../../topics/using/filters/oauth2) with the credentials from above:

    ```yaml
    apiVersion: getambassador.io/v2
    kind: Filter
    metadata:
      name: azure-ad
    spec:
      OAuth2:
        # Azure AD openid-configuration endpoint can be found at https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration
        authorizationURL: https://login.microsoftonline.com/{{TENANT_ID}}/v2.0 
        # Client ID from step 3 above
        clientID: CLIENT_ID
        # Secret created in step 5 above
        secret: CLIENT_SECRET
        # The protectedOrigin is the scheme and Host of your Ambassador endpoint
        protectedOrigins:
        - origin: https://{{AMBASSADOR_URL}}
    ```

2. Create a [FilterPolicy](../../../topics/using/filters/) to use the `Filter` created above

    ```yaml
    apiVersion: getambassador.io/v2
    kind: FilterPolicy
    metadata:
      name: azure-policy
    spec:
      rules:
          # Requires authentication on requests from any hostname
        - host: "*"
          # Tells Ambassador Edge Stack to apply the Filter only on request to the quote /backend/get-quote/ endpoint 
          path: /backend/get-quote/
          # Identifies which Filter to use for the path and hose above
          filters:
            - name: azure-ad
    ```

3. Apply both the `Filter` and `FilterPolicy` above with `kubectl`

    ```
    kubectl apply -f azure-ad-filter.yaml
    kubectl apply -f azure-policy.yaml
    ```

Now any requests to `https://{{AMBASSADOR_URL}}/backend/get-quote/` will require authentication from Azure AD.
