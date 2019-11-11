# Authentication

Ambassador Edge Stack can authenticate incoming requests before routing them to a backing service. In this tutorial, we'll configure Ambassador Edge Stack to use an external third party authentication service.

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Edge Stack Getting Started](/user-guide/getting-started) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started), you'll have a Kubernetes cluster running Ambassador Edge Stack and the tour service. Let's walk through adding authentication to this setup.

Want to use all the features without worrying about passing the limit? [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) allows you to use Ambassador Edge Stack for commericial and unlimited use. 


## 1. Deploy the authentication service

Ambassador Edge Stack delegates the actual authentication logic to a third party authentication service. We've written a [simple authentication service](https://github.com/datawire/ambassador-auth-service) that:

- listens for requests on port 3000;
- expects all URLs to begin with `/extauth/`;
- performs HTTP Basic Auth for all URLs starting with `/backend/get-quote/` (other URLs are always permitted);
- accepts only user `username`, password `password`; and
- makes sure that the `x-qotm-session` header is present, generating a new one if needed.

Ambassador Edge Stack routes _all_ requests through the authentication service: it relies on the auth service to distinguish between requests that need authentication and those that do not. If Ambassador Edge Stack cannot contact the auth service, it will return a 503 for the request; as such, **it is very important to have the auth service running before configuring Ambassador Edge Stack to use it.**

Here's the YAML we'll start with:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: example-auth
spec:
  type: ClusterIP
  selector:
    app: example-auth
  ports:
  - port: 3000
    name: http-example-auth
    targetPort: http-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-auth
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: example-auth
  template:
    metadata:
      labels:
        app: example-auth
    spec:
      containers:
      - name: example-auth
        image: datawire/ambassador-auth-service:2.0.0
        imagePullPolicy: Always
        ports:
        - name: http-api
          containerPort: 3000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
```

Note that the service does _not_ yet contain any Ambassador Edge Stack annotations. This is intentional: we want the service running before we tell Ambassador about it.

The YAML above is published at getambassador.io, so if you like, you can just do

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-auth.yaml
```

to spin everything up. (Of course, you can also use a local file, if you prefer.)

Wait for the pod to be running before continuing. The output of `kubectl get pods` should look something like -
```console
$ kubectl get pods
NAME                            READY     STATUS    RESTARTS   AGE
example-auth-6c5855b98d-24clp   1/1       Running   0          4m
```
Note that the `READY` field says `1/1` which means the pod is up and running.

## 2. Configure Ambassador Edge Stack authentication

Once the auth service is running, we need to tell Ambassador Edge Stack about it. The easiest way to do that is to annotate the `example-auth` service. While we could use `kubectl patch` for this, it's simpler to just modify the service definition and re-apply. Here's the new YAML:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: example-auth
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v2
      kind:  AuthService
      name:  authentication
      auth_service: "example-auth:3000"
      path_prefix: "/extauth"
      allowed_request_headers:
      - "x-qotm-session"
      allowed_authorization_headers:
      - "x-qotm-session"
spec:
  type: ClusterIP
  selector:
    app: example-auth
  ports:
  - port: 3000
    name: http-example-auth
    targetPort: http-api
```

This configuration tells Ambassador Edge Stack about the auth service, notably that it needs the `/extauth` prefix, and that it's OK for it to pass back the `x-qotm-session` header. Note that `path_prefix` and `allowed_headers` are optional.

If the auth service uses a framework like [Gorilla Toolkit](http://www.gorillatoolkit.org) which enforces strict slashes as HTTP path separators, it is possible to end up with an infinite redirect where the auth service's framework redirects any request with non-conformant slashing. This would arise if the above example had ```path_prefix: "/extauth/"```, the auth service would see a request for ```/extauth//backend/get-quote/``` which would then be redirected to ```/extauth/backend/get-quote/``` rather than actually be handled by the
authentication handler. For this reason, remember that the full path of the incoming request including the leading slash, will be appended to```path_prefix``` regardless of non-conformant slashing.

You can apply this file from getambassador.io with

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-auth-enable.yaml
```

or, again, apply it from a local file if you prefer.

Ambassador Edge Stack will see the annotations and reconfigure itself within a few seconds.

## 3. Test authentication

If we `curl` to a protected URL:

```shell
$ curl -v $AMBASSADORURL/backend/get-quote/
```

We get a 401, since we haven't authenticated.

```shell
* TCP_NODELAY set
* Connected to 54.165.128.189 (54.165.128.189) port 32281 (#0)
> GET /backend/get-quote/ HTTP/1.1
> Host: 54.165.128.189:32281
> User-Agent: curl/7.63.0
> Accept: */*
> 
< HTTP/1.1 401 Unauthorized
< www-authenticate: Basic realm="Ambassador Realm"
< content-length: 0
< date: Thu, 23 May 2019 15:24:55 GMT
< server: envoy
< 
* Connection #0 to host 54.165.128.189 left intact
```

If we authenticate to the service, we will get a quote successfully:

```shell
$ curl -v -u username:password $AMBASSADORURL/backend/get-quote/

* TCP_NODELAY set
* Connected to 54.165.128.189 (54.165.128.189) port 32281 (#0)
* Server auth using Basic with user 'username'
> GET /backend/get-quote/ HTTP/1.1
> Host: 54.165.128.189:32281
> Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
> User-Agent: curl/7.63.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< content-type: application/json
< date: Thu, 23 May 2019 15:25:06 GMT
< content-length: 172
< x-envoy-upstream-service-time: 0
< server: envoy
< 
{
    "server": "humble-blueberry-o2v493st",
    "quote": "Nihilism gambles with lives, happiness, and even destiny itself!",
    "time": "2019-05-23T15:25:06.544417902Z"
* Connection #0 to host 54.165.128.189 left intact
}
```

## More

For more details about configuring authentication, read the documentation on [external authentication](/reference/services/auth-service).

## Legacy v0 API
If using Ambassador Edge Stack v0.40.2 or earlier, use the deprecated v0 `AuthService` API
```yaml
      ---
      apiVersion: getambassador.io/v2
      kind:  AuthService
      name:  authentication
      auth_service: "example-auth:3000"
      path_prefix: "/extauth"
      allowed_headers:
      - "x-qotm-session"
```


