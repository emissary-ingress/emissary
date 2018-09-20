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

1. Get Ambassador's external address

2. Edit `scripts/authorization-srv.yaml` by changing `AUTH_CALLBACK_URL` to point to the external address.
```
e.g. http://{EXTERNAL IP}/callback
```
1. Create an Auth0 application and set your callback to `http://{EXTERNAL IP}/callback`. In the app `Connections`, make sure that `google-oauth2` is enabled.

2. Edit `scripts/authorization-srv.yaml` by changing `AUTH_DOMAIN` to your Auth0 app domain.
```
e.g.  {USERNAME}.auth0.com
```

8.  Edit `scripts/authorization-srv.yaml` by changing `AUTH_AUDIENCE` to your Auth0 app audience.
```
e.g.  https://{USERNAME}.auth0.com/api/v2/
```

9.  Edit `scripts/authorization-srv.yaml` by changing `AUTH_CLIENT_ID` to your Auth0 app client-id.

10.  Run these commands:
```
$ kubectl apply -f scripts/httpbin.yaml 
$ kubectl apply -f scripts/policy-crd.yaml
$ kubectl apply -f scripts/httpbin-policy.yaml
$ kubectl apply -f scripts/authorization-srv.yaml
```

11. By running `$ kubectl get services -n datawire`, you should see something like these:
```
ambassador         LoadBalancer   10.31.248.239   35.230.19.92   80:30664/TCP     16m
ambassador-admin   NodePort       10.31.240.190   <none>         8877:30532/TCP   16m
auth0-service      ClusterIP      10.31.254.65    <none>         80/TCP           12m
httpbin            NodePort       10.31.245.125   <none>         80:30641/TCP     13m
```

## From any browser:

1. Go to `http://{EXTERNAL IP}/httpbin/ip`. Your IP address should be displayed in a JSON message.
2. Go to `http://{EXTERNAL IP}/httpbin/headers`. This should take you to the 3-leg auth flow. By signing in with your Google account, you should get redirect back to the original URL and headers should be displayed.
3. Go to `http://{EXTERNAL IP}/httpbin/user-agent`. This should display your `user-agent` without asking for authorization.
4. Open you browser's admin tool and delete your access_token cookie.
5. Go to `http://{EXTERNAL IP}/httpbin/user-agent`. You should be prompt with the 3-leg auth flow again.