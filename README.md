### User documentation
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

### Testing end-to-end
```
$ make e2e_build && make e2e_test 
```
*NOTE:* Auth0 account login credentials can be found in Keybase under `/datawireio/global/ambassador-oauth-ci.txt`.

### Formatting
```
$ make format
```

### Releasing
1. Create a git tag, e.g. git tag 0.0.x-rc
2. Push your tag, e.g. git push 0.0.x-rc

When a branch is tagged, the [CI](https://travis-ci.com/datawire/ambassador-oauth) will build, deploy and test end-to-end you branch tag before pushing the new image to the [docker registry](https://quay.io/repository/datawire/ambassador-pro?tab=tags). Not that the CI does not check tag hierarchy, so make sure that the new tag makes sense. Only tags with format `x.x.x` or `x.x.x-rc` will be accepted.    

### CI & Images
* CI [repo](https://travis-ci.com/datawire/ambassador-oauth) will build and test on every commit.
* An docker image will be pushed to [docker registry](https://quay.io/repository/datawire/ambassador-pro?tab=tags) on every pull-request. The images will be tagged with the correspondent PR number prefixed with `pull-`.

## Manual deployment:
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
7. If OSx, run `$ brew install md5sha1sum` 
8.  Run `make deploy`.  
9. By running `$ kubectl get services -n datawire`, you should see something like these:
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
5. Go to `http://{EXTERNAL IP}/httpbin/user-agent`. You should be prompt with the 3-leg auth flow again.
