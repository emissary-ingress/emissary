### Example files:

ambassador.yaml     - Load-balancer deployment and service.
ambassador-pro.yaml - Authorization deployment and service.
httpbin.yaml        - Service example that will seat behind the load balancer.
policy.yaml         - Tells which resources the Ambassador-Pro service should protect.
tenants.yaml        - Tells which applications should Ambassador-Pro protects.


### Configuration:

1. In ambassador-pro.yaml, search for `AUTH_PROVIDER_URL`. This is the absolute URL that allows Ambassador-Pro communicating with the auth provider. In this case, Auth0.com.
2. In tenants.yaml, at least on app configuration needs to be supplied to `tenants` list. Follow the instructions in the manifest on how to configure an application.


Verifying Ambassador-Pro deployment

The logs should show something like this:
```
time="2018-12-27 22:01:13" level=info msg="loading tenant domain={ APP DOMAIN }, client_id={ APP CLIENT ID }" MAIN=controller
time="2018-12-27 22:02:24" level=info msg="loading rule host=*, path=/httpbin/ip, public=true, scope=offline_access email" MAIN=controller
time="2018-12-27 22:02:24" level=info msg="loading rule host=*, path=/httpbin/user-agent, public=false, scope=offline_access brogramemrs" MAIN=controller
time="2018-12-27 22:02:24" level=info msg="loading rule host=*, path=/qotm, public=true, scope=offline_access dogs" MAIN=controller
```

This error means that no app has been supplied in the list of tenants:
```
time="2018-12-27 23:22:05" level=error msg="0 tenant apps configured" MAIN=controller
```


### Testing Single Application:

1. From a browser call the following: `{ TENANT URL }/httpbin/ip`

Client should not get redirected to Auth0 and an IP should be displayed.

2. From a browser call the following: `{ TENANT URL }/httpbin/user-agent`    

Client should get redirected to Auth0 after a successful login, the user-agent information should be displayed.


### Testing Multiple Applications:

1. Configure a second app domain in Auth0 and add it to the `tenants` list.
2. From a browser: `{ TENANT URL 2 }/httpbin/ip`

Client should not get redirected to Auth0 and an IP should be displayed.

3. From a browser: `{ TENANT URL 2 }/httpbin/user-agent`

Client should get redirected to Auth0 after a successful login, the user-agent information should be displayed.

4. Open a new tab: `{ TENANT URL 1 }/httpbin/user-agent`

If it's the first time calling this URL, the client should be redirected to Auth0. Note that URL 1 and URL 2 will have their own access_token cookies and access should be grant independently from each other.
