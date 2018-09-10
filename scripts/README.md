## Setup and deployment:

1. Create a k8s cluster with at least 3 nodes.
2. Make sure that you have cluster-admin permissions:
```
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user user@datawire.io
```

3. Deploy Ambassador
```
$ kubectl apply -f scripts/ambassador.yaml
```

4. Get Ambassador LB IP address

5. Edit `scripts/authorization-srv.yaml` by changing `AUTH_CALLBACK_URL` to point to your LB IP address.
```
e.g. http://{YOUR LB IP}/callback
```
6. Create an Auth0 application and set your callback to `http://{YOUR LB IP}/callback` In app `Connections`, make sure that `google-oauth2` is enabled.

7. Edit `scripts/authorization-srv.yaml` by changing `AUTH_DOMAIN` to your Auth0 App domain.
```
e.g.  gsagula.auth0.com
```

8.  Edit `scripts/authorization-srv.yaml` by changing `AUTH_AUDIENCE` to your Auth0 App audience.
```
e.g.  https://gsagula.auth0.com/api/v2/
```

9.  Edit `scripts/authorization-srv.yaml` by changing `AUTH_CLIENT_ID` to your Auth0 App client-id.
```
e.g. -_lWmw3zOpFXdY6XR9cgk-vfSdtYwaC6
```

10.  Deploy the following:
```
$ kubectl apply -f scripts/httpbin.yaml 
$ kubectl apply -f scripts/policy-crd.yaml
$ kubectl apply -f scripts/httpbin-policy.yaml
$ kubectl apply -f scripts/authorization-srv.yaml
```
###From any browser:

1. Go to `http://{YOUR LB IP}/httpbin/ip`. Your IP address should be displayed in a JSON message.

2. Go to `http://{YOUR LB IP}/httpbin/headers`. This should take you to the 3-leg auth flow. By signing in with your Google account, you should get redirect back to the original URL and headers should be displayed.
3. Go to `http://{YOUR LB IP}/httpbin/user-agent`. This should display your `user-agent` without asking for authorization.
4. Open you browser's admin tool and delete your access_token cookie.
5. Go to `http://{YOUR LB IP}/httpbin/user-agent`. You should be prompt with the 3-leg auth flow again.