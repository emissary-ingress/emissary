# Configuring OAuth/OIDC Authentication
---

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes for single sign-on with an external identity providers (IDP). This guide will demonstrate configuration using Auth0 as your IDP. 

**Note:** If you need to use an IDP other than Auth0, please [Slack](https://d6e.co/slack) or email us. We are currently testing support for other IDPs, including Keycloak, Okta, and AWS Cognito.

## Configure your IDP
You will need to configure your IDP to handle authentication requests. The way to do this varies by IDP.

#### Auth0
With Auth0 as your IDP, you will need to create an `Application` to handle authentication requests from Ambassador Pro.

1. Navigate to Applications and Select "CREATE APPLICATION"

  ![](/images/create-application.png)

2. In the pop-up window, give the application a name and create a "Machine to Machine App"

  ![](/images/machine-machine.png)

3. Select the Auth0 Management API. Grant any scopes you may require. (You may grant none.)

  ![](/images/scopes.png)
  
4. In your newly created application, click on the Settings tab, add the Domain and Callback URLs for your service and ensure the "Token Endpoint Authentication Method" is set to `Post`. The default YAML installation of Ambassador Pro uses `/callback` for the URL, so the values should be the domain name that points to Ambassador, e.g., `example.com/callback` and `example.com`.

  ![](/images/Auth0_none.png)


## Configure your Authentication Tenants

**Note:** Ensure your [authentication provider](/user-guide/ambassador-pro-install/#5-single-sign-on) is set in your Ambassador Pro deployment before configuring authentication tenants.

Ambassador Pro is integrated with your IDP via the `Tenant` custom resource definition. This is where you will tell Ambassador Pro which hosts to require authentication from and what client to use for authentication. 

To configure your tenant, create the following YAML and put it in a file called `tenants.yaml`.

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Tenant
metadata:
  name: domain1-tenant
spec:
  tenants:
    # The URL used to access your app.
    - tenantUrl: {scheme}://{hostname or ip}
    # The API Audience that is listening for authentication requests
      audience: https://example.auth0.com/api/v2/
    # Client ID from your authentication application
      clientId: <CLIENT_ID>
    # Client Secret from your authentication application
      secret: <CLIENT_SECRET>
```

If you are using Auth0, get the `Client ID` and `Client Secret` from your application settings:

![](/images/Auth0_secret.png)

The `audience` is the API Audience of your Auth0 Management API:

![](/images/Auth0_audience.png)

Apply the YAML with `kubectl`.

```
kubectl apply -f tenants.yaml
```

## Configure Authentication Across Multiple Domains (Optional)
Ambassador Pro supports authentication for multiple domains where each domain is issued its own access token. For example, imagine you're hosting both `domain1.example.com` and `domain2.example.com` on the same cluster. With multi-domain support, users will receive separate authentication tokens for `domain1` and `domain2`.

To configure multi-domain access, you will need to create another authentication endpoint with your IDP (see [Configure your IDP](/user-guide/oauth-oidc-auth/#configure-your-idp)) and create another `Tenant` for the new domain.

Example:

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Tenant
metadata:
  name: domain1-tenant
spec:
  tenants:
    # Domain 1
    - tenantUrl: http://domain1.example.com
      audience: https://example.auth0.com/api/v2/
      clientId: <APP1_CLIENT_ID>
      secret: <APP1_CLIENT_SECRET>
```

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Tenant
metadata:
  name: domain2-tenant
spec:
  tenants:
    # Domain 2
    - tenantUrl: http://domain2.example.com
      audience: https://example.auth0.com/api/v2/
      clientId: <APP2_CLIENT_ID>
      secret: <APP2_CLIENT_SECRET>
```

This will tell Ambassador Pro to configure separate access tenants for `http://domain1.example.com` and `http://domain2.example.com`. After a subsequent login to either domain, Ambassador Pro will create a separate SSO token for just that domain.

## Test Authentication
After applying Ambassador Pro and the `tenants.yaml` file, Ambassador Pro should be configured to authenticate with your IDP. 

You can use any service to test this. From a web browser, attempt to access your service (e.g., `http://domain1.example.com/httpbin/`) and you should be redirected to a login page. Log in using your credentials and you should be redirected to your application.

Next, test SSO by attempting to access the application from a different tab. You should be sent to your application without being redirected to the login page.

You can also use a JWT for authentication through Ambassador Pro. To do this, click on APIs, the API you're using for the Ambassador Authentication service, and then the Test tab. Run the curl command given to get the JWT.

![](/images/Auth0_JWT.png)

After you have the JWT, use it to send a test `curl` to your app by passing it in the `authorization:` header.

```
$ curl --header 'authorization: Bearer eyeJdfasdf...' http://datawire-ambassador.com/httpbin/user-agent
{
  "user-agent": "curl/7.54.0"
}
```

## Configure Access Controls
By default, Ambassador Pro will require all requests be authenticated before passing through. If some services or resources do not require authentication, Ambassador Pro allows for you to configure which services you want authenticated. This is done with the `Policy` custom resource definition. 

This is an example policy for the `httpbin` service defined in the [YAML installation guide](/user-guide/getting-started#3-creating-your-first-route).

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Policy
metadata:
  name: policy
spec:
  rules:
  - host: example.com
    path: /httpbin/ip
    public: true
    scope: openid
  - host: example.com
    path: /httpbin/user-agent
    public: false
    scope: openid
```
This policy will tell Ambassador Pro to not require authentication for requests to `http://example.com/httpbin/ip`. See [Access Control](/reference/services/access-control) for more information.

**Note:** `scope: openid` is required if your authentication server is OIDC Conformant.

