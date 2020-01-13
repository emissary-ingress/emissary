# Salesforce Single Sign-On

## Set up Saleforce

To use Salesforce as your IdP, you will first need to register an OAuth application with your Salesforce tenant. This guide will walk you through the most basic setup via the "Salesforce Classic Experience".

1. In the `Setup` page, under `Build` click the dropdown next to `Create` and select `Apps`

2. Under `Connected Apps` at the bottom of the page, click on `New` at the top

3. Fill in the following fields with whichever values you want

    - Connected App Name
    - API Name
    - Contact Email 

4. Under `API (Enable OAuth Settings)` check the box next to `Enable OAuth Settings`

5. Fill in the `Callback URL` section with `https://{{AMBASSADOR_HOST}}/.ambassador/oauth2/redirection-endpoint`

6. Under `Selected OAuth Scopes` you must select the `openid` scope at the minimum. Select any other scopes you want included in the response as well.

7. Click `Save` and `Continue` to create the application

8. Record the `Consumer Key` and `Consumer Secret` values from the `API (Enable OAuth Settings)` section in the newly created application's description page.

After waiting for salesforce to register the application with their servers, you should be ready to configure Ambassador Edge Stack to Salesforce as an IdP.

## Set up Ambassador Edge Stack

After configuring an OAuth application in Salesforce, configuring Ambassador Edge Stack to make use it for authentication is simple.

1. Create an [OAuth Filter](../../filter-reference#filter-type-oauth2) with the credentials from above

    ```yaml
    apiVersion: getambassador.io/v2
    kind: Filter
    metadata:
      name: salesforce
    spec:
      OAuth2:
        # Salesforce's generic OpenID configuration endpoint at https://login.salesforce.com/ will work but you can also use your custom Salesforce domain i.e.: http://datawire.my.salesforce.com
        authorizationURL: https://login.salesforce.com/
        # The clientURL is the scheme and Host of your Ambassador endpoint
        clientURL: https://{{AMBASSADOR_HOST}}
        # Consumer Key from above
        clientID: {{Consumer Key}}
        # Consumer Secret from above
        secret: {{Consumer Secret}}
    ```

2. Create a [FilterPolicy](../../filter-reference#filterpolicy-definition) to use the `Filter` created above

    ```yaml
    apiVersion: getambassador.io/v2
    kind: FilterPolicy
    metadata:
      name: oauth-policy
    spec:
      rules:
          # Requires authentication on requests from any hostname
        - host: "*"
          # Tells Ambassador Edge Stack to apply the Filter only on request to the quote /backend/get-quote/ endpoint
          path: /backend/get-quote/
          # Identifies which Filter to use for the path and hose above
          filters:
            - name: salesforce
            # Any additional scopes granted in step 6 above can be requested with the arguments field
            #  arguments:
            #    scopes:
            #    - refresh_token
                
    ```

3. Apply both the `Filter` and `FilterPolicy` above with `kubectl`

    ```
    kubectl apply -f salesforce-filter.yaml
    kubectl apply -f oauth-policy.yaml
    ```

Now any requests to `https://{{AMBASSADOR_URL}}/backend/get-quote/` will require authentication from Salesforce.
