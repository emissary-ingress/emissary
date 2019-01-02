# Configuring OAuth/OIDC Authentication
---

Ambassador Pro adds native support for the OAuth and OIDC authentication schemes for single sign-on with various identity providers (IDPs). This guide will demonstrate configuration using Auth0 as your IDP. 

**Note:** If you need to use an IDP other than Auth0, please contact us. We are currently testing support for other IDPs, including Keycloak, Okta, and AWS Cognito.

## Create an Authentication Application with your IDP
You will need to have a running application with your IDP to integrate with Ambassador Pro. 

#### Auth0
You will need to create an application with Auth0 before integrating it with Ambassador Pro. 

1. Navigate to Applications and Select "CREATE APPLICATION"

  ![](/images/create-application.png)

2. In the pop-up window, give the application a name and create a "Machine to Machine App"

  ![](/images/machine-machine.png)

3. Select the Auth0 Management API and grant all scopes to the Application.

  ![](/images/scopes.png)
  
4. In your newly created application, list the Domain and Callback URLs for your service and set the "Token Endpoint Authentication Method" to `None`.

  ![](/images/Auth0_none.png)

## Set Your Auth Provider
You first need to tell Ambassador Pro which URL to redirect to for authentication. If you are using Auth0, this URL will be the Domain of your Auth0 application and which can be found here:

![](/images/Auth0_domain_clientID.png)

```
env:
# Auth provider's abolute url: {scheme}://{host}
- name: AUTH_PROVIDER_URL
  value: https://datawire-ambassador.auth0.com
```

## Configure your Authentication Tenants
Ambassador Pro's authentication service is configured per application you need authenticating. This is configured with the `Tenant` custom resource definition. 

To configure your tenant, create the following YAML and put it in a file called `tenants.yaml`.

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Tenant
metadata:
  name: tenant
spec:
  tenants:
  
    # The URL used to access your app.
    - tenantUrl: {scheme}://{hostname or ip}
    # The API Audience that is listening for authentication requests
      audience: <AUDIENCE>
    # Client ID from your authentication application
      clientId: <CLIENT_ID>
    # Client Secret from your authentication application
      secret: <CLIENT_SECRET>
```

If you are using Auth0, grab the Client ID and Client Secret from your application settings:

![](/images/Auth0_secret.png)

The `audience` is the API Audience of your Auth0 Management API:

![](/images/Auth0_audience.png)

Apply the YAML with `kubectl`

```
kubectl apply -f tenants.yaml
```

## Configure Authentication Across Multiple Domains

Ambassador Pro allows authentication from multiple domains. This is easily configured in your `tenants.yaml` file. Each tenant object is processed separately allowing for you to define multiple tenants authenticating with the same IDP. 

Example:

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Tenant
metadata:
  name: tenant
spec:
  tenants:
  
    # Domain 1
    - tenantUrl: http://domain1.example.com
      audience: https://datawire-ambassador.auth0.com/api/v2/
      clientId: <CLIENT_ID>
      secret: <CLIENT_SECRET>
    
    # Domain 2
    - tenantUrl: http://domain2.example.com
      audience: https://datawire-ambassador.auth0.com/api/v2/
      clientId: <CLIENT_ID>
      secret: <CLIENT_SECRET>
```

This will configure Ambassador Pro to require authentication from requests to `http://domain1.example.com` and `http://domain2.example.com`. Ambassador Pro will then create a separate SSO token for each domain.

**Note:** ensure both domains and callback URLs are listed in `Allowed Web Origins` and `Allowed Callback URLs` respectively. 

## Test Authentication
After applying Ambassador Pro and the `tenants.yaml` file, Ambassador Pro should be configured to authenticate with your IDP. 

You can use any service to test this. From a web browser, attempt to access your service e.g. `http://domain1.example.com/httpbin/`, and you should be redirected to an Auth0 login page. Log in using your credentials and you should be redirected to your application. 

Next, test SSO by attempting to access the application from a different tab. You should be sent to your application without being redirected to Auth0. 

## Configure Access Controls
By default, Ambassador Pro will require all requests be authenticated before passing through. If some services or resources do not require authentication, Ambassador Pro allows for you to configure which services you want authenticated. This is done with the `Policy` custom resource definition. 

This is an example policy for the `httpbin` service defined in the [YAML instalation guide](/user-guide/getting-started#3-creating-your-first-route).

```
---
apiVersion: stable.datawire.io/v1beta1
kind: Policy
metadata:
  name: policy
spec:
  rules:
  - host: "*"
    path: /httpbin/ip
    public: true
  - host: "*"
    path: /httpbin/user-agent
    public: false
```
This policy will tell Ambassador Pro to not require authentication for requests to `http://example.com/httpbin/ip`. See [Access Control](/reference/services/access-control) for more information.

