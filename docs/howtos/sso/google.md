# Google Single Sign-On

## Create an OAuth client in the Google API Console

To use Google as an IdP for Single Sign-On, you will first need to create an OAuth web application in the Google API Console.

1. Open the [Credentials page](https://console.developers.google.com/apis/credentials) in the API Console
2. Click `Create credentials > OAuth client ID`.
3. Select `Web application` and give it a name
4. Under **Restrictions**, fill in the **Authorized redirect URIs** with
   
   ```
   http(s)://{{AMBASSADOR_URL}}/.ambassador/oauth2/redirection-endpoint
   ```
5. Click `Create` 
6. Record the `client ID` and `client secret` in the pop-up window. You will need these when configuring Ambassador Edge Stack

## Set up Ambassador Edge Stack

After creating an OAuth client in Google, configuring Ambassador Edge Stack to make use of it for authentication is simple.

1. Create an [OAuth Filter](../../../topics/using/filters/oauth2) with the credentials from above:

    ```yaml
    apiVersion: getambassador.io/v2
    kind: Filter
    metadata:
      name: google
    spec:
      OAuth2:
        # Google openid-configuration endpoint can be found at https://accounts.google.com/.well-known/openid-configuration
        authorizationURL: https://accounts.google.com
        # Client ID from step 6 above
        clientID: CLIENT_ID
        # Secret created in step 6 above
        secret: CLIENT_SECRET
        # The protectedOrigin is the scheme and Host of your Ambassador endpoint
        protectedOrigins:
        - origin: http(s)://{{AMBASSADOR_URL}}
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
            - name: google
    ```
3. Apply both the `Filter` and `FilterPolicy` above with `kubectl`

    ```
    kubectl apply -f google-filter.yaml
    kubectl apply -f google-policy.yaml
    ```

Now any requests to `https://{{AMBASSADOR_URL}}/backend/get-quote/` will require authentication from Google.
