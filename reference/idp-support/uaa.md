# Cloud Foundry User Account and Authentication Service (UAA)

**IMPORTANT:** Ambassador Edge Stack requires the IDP return a JWT signed by the RS256 algorithm (asymmetric key). UAA defaults to symmetric key encryption which Ambassador Edge Stack cannot read. You will need to provide your own asymmetric key when configuring UAA. e.g.




`uaa.yml`
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


1. Create an OIDC Client

   ```shell
   uaac client add ambassador --name ambassador-client --scope openid --authorized_grant_types authorization_code,refresh_token --redirect_uri {AMBASSADOR_URL}/callback --secret CLIENT_SECRET
   ```

   **Note:** Change the value of `{AMBASSADOR_URL}` with the IP or DNS of your Ambassador load balancer.

2. Configure you OAuth `Filter` and `FilterPolicy`

   Use the id (ambassador) and secret (CLIENT_SECRET) from step 1 to configure the OAuth `Filter`.

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

   **Note:** The `scopes` field was set when creating the client in step 1. You can add any scopes you would like when creating the client.

   <div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
