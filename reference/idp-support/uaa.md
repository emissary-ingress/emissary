# User Account and Authentication Service (UAA)

**IMPORTANT:** Ambassador Edge Stack requires the IdP to return a JWT signed by the RS256 algorithm (asymmetric key). Cloud Foundry's UAA defaults to symmetric key encryption which Ambassador Edge Stack cannot read. 

1. You will need to provide your own asymmetric key when configuring UAA in a file called `uaa.yml`. For example:

   ```yaml
   jwt:
      token:
         signing-key: |
            -----BEGIN RSA PRIVATE KEY-----
            MIIEpAIBAAKCAQEA7Z1HBM6QFqnIJ1UA3NWnYMuubt4XlfbP1/GopTWUmchKataM
            ...
            ...
            QSbJdIbUBwL8BcrfNw4ebp1DgTI9F45Re+evky0A82aL0/BvBHu8og==
            -----END RSA PRIVATE KEY-----
   ```

2. Create an OIDC Client:

   ```shell
   uaac client add ambassador --name ambassador-client --scope openid --authorized_grant_types authorization_code,refresh_token --redirect_uri {AMBASSADOR_URL}/.ambassador/oauth2/redirection-endpoint --secret CLIENT_SECRET
   ```

   **Note:** Change the value of `{AMBASSADOR_URL}` with the IP or DNS of your Ambassador load balancer.

## Configure Filter and FilterPolicy

Configure your OAuth `Filter` and `FilterPolicy` with the following:

   Use the clientID (`ambassador`) and secret (`CLIENT_SECRET`) from Step 2 to configure the OAuth `Filter`.

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Filter
   metadata:
     name: uaa_filter
     namespace: default
   spec:
     OAuth2:
       authorizationURL: {UAA_DOMAIN}
       clientURL: https://datawire-ambassador.com
       audience: {UAA_DOMAIN}
       clientID: ambassador
       secret: CLIENT_SECRET
   ```
  
   **Note:** The `authorizationURL` and `audience` are the same for UAA configuration.

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
           - name: uaa_filter ## Enter the Filter name from above
             arguments:
               scopes:
               - "openid"
   ```

**Note:** The `scopes` field was set when creating the client in Step 2. You can add any scopes you would like when creating the client.
