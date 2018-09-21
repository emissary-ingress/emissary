## User documentation

If you just want to try to run this as an end user, the end user documentation is here:

https://github.com/datawire/ambassador-pro

## Developer Documentation

### Cloning
Project must be cloned in `$GOPATH/src/github.com/datawire/ambassador-oauth/`.

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

## CI & Images
* CI [repo](https://travis-ci.com/datawire/ambassador-oauth) will build and test on every commit.
* An docker image will be pushed to [docker registry](https://quay.io/repository/datawire/ambassador-pro?tab=tags) on every pull-request. The images will be tagged with the correspondent PR number.

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

5. Create an Auth0 application and set your callback to `http://{EXTERNAL IP}/callback`. In the app `Connections`, make sure that `google-oauth2` is enabled and that your "Token Endpoint Authetication Method" is set to "Post" or "None".

6. Copy `env.sh.in to env.sh` and fill in the variables according to the comments.

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
