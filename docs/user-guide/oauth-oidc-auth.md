# Configuring OAuth/OIDC Authentication
---

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes which are used by various identify providers (IDPs). This guide will demonstrate configuration using the Auth0 IDP. 

## Install Ambassador Pro
You need to have Ambassador Pro installed in your cluster to add OAuth/OIDC Authentication.

1. Install your Ambassador Pro registry credentials secret. If you have lost this, email us at support@datawire.io.
2. Deploy Ambassador Pro

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-pro.yaml
```

**Note:** This deploys Ambassador Pro with the authentication service as a sidecar to Ambassador. This is the recommended deployment pattern. If you wish to deploy Ambassador Pro Authentication as an independent service use:

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/pro/auth.yaml
```

**Note:** The `namespace` field in the `ClusterRoleBinding` is configured to `default`. Make sure to download the yaml and change this if you wish to deploy in a non-`default` namespace.

3. Define the Ambassador Pro Authentication service

```
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador-pro
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  AuthService
      name:  authentication
      auth_service: ambassador-pro
      allowed_headers:
       - "Authorization"
       - "Client-Id"
       - "Client-Secret"
spec:
  selector:
    name: ambassador-pro
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

## Configuring the `oauth-oidc-config` ConfigMap
### Auth0 Default Configuration

OAuth and OIDC authentication is configured via a `ConfigMap` named `oauth-oidc-config`. The default installation creates this ConfigMap with default values that need to be changed.  Create a file named `oauth-oidc-config.yaml` (or `kubectl edit` the existing `ConfigMap`) and set the `AUTH_DOMAIN`, `AUTH_CLIENT_ID`, `AUTH_AUDIENCE`, and `AUTH_CALLBACK_URL` environment variables. (You'll need to create an Auth0 custom API if you haven't already.)

* `AUTH_DOMAIN` is your Auth0 domain, e.g., foo.auth0.com.
* `AUTH_CLIENT_ID` is the client ID of your application.
* `AUTH_AUDIENCE` is listed on the API page https://manage.auth0.com/#/apis
* `AUTH_CALLBACK_URL` is the URL where you want to send users once they've authenticated.
* `APP_SECURE` indicates if the domain is secured with TLS. Set to `not_secure` if the callback URL is unsecure and omit otherwise.

#### Example
- `AUTH_DOMAIN` = datawire-ambassador.auth0.com
- `AUTH_CLIENT_ID` = vdrLZ8Y6AASktot75tCaAif4u9xrrE_g

![](/images/Auth0_domain_clientID.png)

- `AUTH_AUDIENCE` = https://datawire-ambassador.auth0.com/api/v2/

![](/images/Auth0_audience.png)

- `AUTH_CALLBACK_URL` = https://datawire-ambassador.com/callback/

**`oauth-oidc-config.yaml`**

```
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-oidc-config
data:
  AUTH_CALLBACK_URL: https://datawire-ambassador.com/callback
  AUTH_DOMAIN: datawire-ambassador.auth0.com
  AUTH_AUDIENCE: https://datawire-ambassador.auth0.com/api/v2/
  AUTH_CLIENT_ID: vdrLZ8Y6AASktot75tCaAif4u9xrrE_g
```

#### Verify the Application
* Set `Token Endpoint Authentication Method` to `None`
* Add the value of `AUTH_CALLBACK_URL` to `Allowed Callback URLs`
* Add your domain to `Allowed Web Origins`

![](/images/Auth0_none.png)

* Deploy Ambassador Pro
	* Creates the Ambassador Pro deployment
	* Configuration ConfigMap
	* Cluster resources
	* [Policy](/reference/services/access-control) Custom Resource


### Auth0 Validation Mode Configuration

When deployed in validation mode, Ambassador Pro will validate configuration via the Auth0 management API. In the future, we may add more automatic configuration via the management API. 

#### Configuration
The `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE` and `AUTH_CLIENT_ID` environment variables need to be configured, same as the [default configuration](/user-guide/oauth-oidc-auth#auth0-default-configuration). An extra environment variable, `AUTH_CLIENT_SECRET` is also required for the validation configuration.

![](/images/Auth0_secret.png)

#### Verify the Application
* Set `Token Endpoint Authentication Method` to `POST`
* Add the value of `AUTH_CALLBACK_URL` to `Allowed Callback URLs`
* Add your domain to `Allowed Web Origins`

![](/images/Auth0_method_callback_origins.png)

* App is authorized to access Auth0 management api (APIs/Machine to Machine Applications/click the dropdown) and following scopes have been granted:
	* read:clients
	* read:grants
* App is set with the following grant types (Applications/Advanced Settings/Grant Types): 
	* Authorization Code
	* Client Credentials
* Deploy Ambassador Pro
	* Creates the Ambassador Pro deployment
	* Configuration ConfigMap
	* Cluster resources
	* [Policy](/reference/services/access-control) Custom Resource


## Test the Auth0 Application
Authentication policies are managed by the `policy` CRD we deployed in the configuration step. We can deploy an authentication policy and test this Auth0 application using the example `httpbin` service in the [YAML installation guide](/user-guide/getting-started#3-creating-your-first-route).

1. If applied, delete the Ambassador Pro authentication service.
2. Deploy the `httpbin` service.
3. Verify the service is working:

   ```
   $ curl http://$AMBASSADOR_IP/httpbin/ip
   {
    "origin": "35.205.31.151"
   }
   $ curl http://$AMBASSADOR_IP/httpbin/user-agent
   {
    "user-agent": "curl/7.54.0"
   }
   ```
4. Follow the configuration steps above to deploy Ambassador Pro authentication.
5. Resend the curl requests, you will notice it now requires authentication.
6. Deploy an `httpbin` authentication `policy`. Refer to the [Access Control](/reference/services/auth-policy) documentation for more information.
   
   ```
   apiVersion: stable.datawire.io/vibeta1
   kind: Policy
   metadata:
     name: httpbin-policy
   spec:
     rules:
      - host: "*"
        path: /callback
        public: true
      - host: "*"
        path: /httpbin/ip
        public: true
      - host: "*"
        path: /httpbin/user-agent/*
        public: false
      - host: "*"
        path: /httpbin/headers/*
        scopes: "read:test"
   ```
7. Test the policy worked with `cURL`:

   ```
   $ curl http://$AMBASSADOR_IP/httpbin/ip
   {
    "origin": "35.205.31.151"
   }
   $ curl http://$AMBASSADOR_IP/httpbin/user-agent
   <a href="https://xxx.auth0.com/authorize?audience=https://xxx.auth0.com/api/v2/&amp;response_type=code&amp;redirect_uri=http://35.226.13.0/callback&amp;client_id=Z6m3lwCot6GaThT4L142nkOKNPeDe87n&amp;state=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MzY2OTQ2MjglhdCI6MUzNjY5NDMyOCwianRpIjoiN2FjOThjZTQtYjdjZi00NTU3LTlkYTEtZGJjNzZjYzNjZjg4IiwibmJmIjowLCJwYXRoIjoiL2h0dHBiaW4vdXNi1hZ2VudCJ9.NtBA5deqPn5XI7vonca4tpgYNrM-212TiQhTZ_KzWos&amp;scope=offline_access openid profile">See Other</a>
   ```
8. Visit `https://$AMBASSADOR_IP/httpbin/user-agent` and you should be redirected to an Auth0 log in page. 
9. If you want to test with a JWT, you can get a JWT from Auth0. To do this, click on APIs, then the custom API you're using for the Ambassador Authentication service, and then the Test tab. Pass the JWT in the authorization: Bearer HTTP header:

```
   $ curl --header 'authorization: Bearer eyeJdfasdf...' http://$AMBASSADOR_IP/httpbin/user-agent
   {
     "user-agent": "curl/7.54.0"
   }
```
