# Configuring OAuth/OIDC Authentication
---

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes for single sign-on with various identity providers (IDPs). This guide will demonstrate configuration using the Auth0 IDP. 
## Configuring Environment Variables
Auth0 integration supports two different configuration patterns. The [default configuration](/user-guide/oauth-oidc-auth#auth0-default-configuration) integrates Auth0 with Ambassador Pro without verification from the Auth0 management API. If you want the Auth0 management API to verify your application configuration, follow the [validation mode configuration](/user-guide/oauth-oidc-auth#auth0-validation-mode-configuration).

### Auth0 Default Configuration

Integrating Auth0 with the Ambassador Pro Authentication service is done by setting evironment variables in the deployment manifest. In your deployment file, configure the `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE`, and `AUTH_CLIENT_ID` environment variables based on your Auth0 configuration. (You'll need to create an Auth0 custom API if you haven't already.)

* `AUTH_DOMAIN` is your Auth0 domain, e.g., foo.auth0.com.
* `AUTH_CLIENT_ID` is the client ID of your application.
* `AUTH_AUDIENCE` is listed on the API page https://manage.auth0.com/#/apis
* `AUTH_CALLBACK_URL` is the URL where you want to send users once they've authenticated.

#### Configuration
- `AUTH_DOMAIN` = datawire-ambassador.auth0.com
- `AUTH_CLIENT_ID` = vdrLZ8Y6AASktot75tCaAif4u9xrrE_g

![](/images/Auth0_domain_clientID.png)

- `AUTH_AUDIENCE` = https://datawire-ambassador.auth0.com/api/v2/

![](/images/Auth0_audience.png)

- `AUTH_CALLBACK_URL` = https://datawire-ambassador.com/callback


1. Configure the environment variables with the correct values

Ex.

```
env:
- name: AUTH_CALLBACK_URL
  value: https://datawire-ambassador.com/callback
- name: AUTH_DOMAIN
  value: datawire-ambassador.auth0.com
- name: AUTH_AUDIENCE
  value: https://datawire-ambassador.auth0.com/api/v2/
- name: AUTH_CLIENT_ID
  value: vdrLZ8Y6AASktot75tCaAif4u9xrrE_g
```

2. Set `Token Endpoint Authentication Method` to `None`
3. Add the value of `AUTH_CALLBACK_URL` to `Allowed Callback URLs`
4. Add your domain to `Allowed Web Origins`
5. Deploy Ambassador Pro
	* Creates the Ambassador Pro deployment
	* Cluster resources
	* [Policy](/reference/services/access-control) Custom Resource
6. [Test the application.](/user-guide/oauth-oidc-auth/#test-the-auth0-application)

![](/images/Auth0_none.png)


### Auth0 Validation Mode Configuration
When deployed in validation mode, Ambassador Pro will validate configuration via the Auth0 management API. In the future, we may add more automatic configuration via the management API. 

#### Configuration
The `AUTH_CALLBACK_URL`, `AUTH_DOMAIN`, `AUTH_AUDIENCE` and `AUTH_CLIENT_ID` environment variables need to be configured, same as the [default configuration](/user-guide/oauth-oidc-auth#auth0-default-configuration). An extra environment variable, `AUTH_CLIENT_SECRET` is also required for the validation configuration.

![](/images/Auth0_secret.png)

1. Configure the environment variables with the correct values

Ex.

```
env:
- name: AUTH_CALLBACK_URL
  value: https://datawire-ambassador.com/callback
- name: AUTH_DOMAIN
  value: datawire-ambassador.auth0.com
- name: AUTH_AUDIENCE
  value: https://datawire-ambassador.auth0.com/api/v2/
- name: AUTH_CLIENT_ID
  value: vdrLZ8Y6AASktot75tCaAif4u9xrrE_g
- name: AUTH_CLIENT_SECRET
  value: <CLIENT SECRET>
```
2. Set `Token Endpoint Authentication Method` to `POST`
3. Add the value of `AUTH_CALLBACK_URL` to `Allowed Callback URLs`
4. Add your domain to `Allowed Web Origins`

![](/images/Auth0_method_callback_origins.png)

5. Authorize the application to access Auth0 management api (APIs/Machine to Machine Applications/click the dropdown) and following scopes have been granted:
	* read:clients
	* read:grants
6. Set the following grant types (Applications/Advanced Settings/Grant Types): 
	* Authorization Code
	* Client Credentials
7. Deploy Ambassador Pro
	* Creates the Ambassador Pro deployment
	* Cluster resources
	* [Policy](/reference/services/access-control) Custom Resource
8. [Test the application.](/user-guide/oauth-oidc-auth/#test-the-auth0-application)


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
4. Deploy Ambasador Pro Authentication 

```
$ kubectl apply -f ambassador-pro-auth.yaml
$ kubectl apply -f ambassador-pro-auth-service.yaml
```
5. Resend the curl requests, you will notice it now requires authentication.
6. Deploy an `httpbin` authentication `policy`. Refer to the [Access Control](/reference/services/access-control) documentation for more information.
   
   ```
   apiVersion: stable.datawire.io/v1beta1
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
8. Visit `http://$AMBASSADOR_IP/httpbin/user-agent` and you should be redirected to an Auth0 log in page. 
9. If you want to test with a JWT, you can get a JWT from Auth0. To do this, click on APIs, then the custom API you're using for the Ambassador Authentication service, and then the Test tab. Pass the JWT in the authorization: Bearer HTTP header:

```
   $ curl --header 'authorization: Bearer eyeJdfasdf...' http://$AMBASSADOR_IP/httpbin/user-agent
   {
     "user-agent": "curl/7.54.0"
   }
```
