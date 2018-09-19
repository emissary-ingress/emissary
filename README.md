## User documentation

If you just want to try to run this as an end user, the end user documentation is here:

https://github.com/datawire/ambassador-pro

## Developer Documentation

### Installing
```
$ make install
```

### Testing
```
$ make test
```

### Formatting
```
$ make format
```

### Releasing
```
$ docker build . -t quay.io/ambassador-pro/ambassador-pro:0.x
$ docker push quay.io/datawire/ambassador-pro:0.x
```

## Setup and deployment:

1. Create a k8s cluster.
2. Make sure that you have cluster-admin permissions:
```
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user user@datawire.io
```

3. Deploy Ambassador
```
$ kubectl apply -f scripts/ambassador.yaml
```
4. Get Ambassador's external address (EXTERNAL_IP)

5. Create an Auth0 application and set your callback to `http://{EXTERNAL IP}/callback`. In the app `Connections`, make sure that `google-oauth2` is enabled.

6. Copy `env.sh.in to env.sh` and fill in the following variables:

   - Set EXTERNAL_IP to your ambassador's external address from step 4.
   - Set AUTH_DOMAIN to your Auth0 app domain.
   - Set AUTH_AUDIENCE to your Auth0 app audience.
   - Set AUTH_CLIENT_ID to your Auth0 app client-id.

7.  Run `make deploy`

8. By running `$ kubectl get services -n datawire`, you should see something like these:
```
ambassador         LoadBalancer   10.31.248.239   35.230.19.92   80:30664/TCP     16m
ambassador-admin   NodePort       10.31.240.190   <none>         8877:30532/TCP   16m
auth0-service      ClusterIP      10.31.254.65    <none>         80/TCP           12m
httpbin            NodePort       10.31.245.125   <none>         80:30641/TCP     13m
```

## Manual testing:

1. Go to `http://{EXTERNAL IP}/httpbin/ip`. Your IP address should be displayed in a JSON message.
2. Go to `http://{EXTERNAL IP}/httpbin/headers`. This should take you to the 3-leg auth flow. By signing in with your Google account, you should get redirect back to the original URL and headers should be displayed.
3. Go to `http://{EXTERNAL IP}/httpbin/user-agent`. This should display your `user-agent` without asking for authorization.
4. Open you browser's admin tool and delete your access_token cookie.
5. Go to `http://{EXTERNAL IP}/httpbin/user-agent`. You should be prompt with the 3-leg auth flow again.6
