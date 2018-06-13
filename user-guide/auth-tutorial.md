# Authentication

Ambassador can authenticate incoming requests before routing them to a backing service. In this tutorial, we'll configure Ambassador to use an external third party authentication service.

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started.html) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started.html), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding authentication to this setup.

## 1. Deploy the authentication service

Ambassador delegates the actual authentication logic to a third party authentication service. We've written a [simple authentication service](https://github.com/datawire/ambassador-auth-service) that:

- listens for requests on port 3000;
- expects all URLs to begin with `/extauth/`;
- performs HTTP Basic Auth for all URLs starting with `/qotm/quote/` (other URLs are always permitted);
- accepts only user `username`, password `password`; and
- makes sure that the `x-qotm-session` header is present, generating a new one if needed.

Ambassador routes _all_ requests through the authentication service: it relies on the auth service to distinguish between requests that need authentication and those that do not. If Ambassador cannot contact the auth service, it will return a 503 for the request; as such, **it is very important to have the auth service running before configuring Ambassador to use it.**

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
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-auth
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: example-auth
    spec:
      containers:
      - name: example-auth
        image: datawire/ambassador-auth-service:1.1.1
        imagePullPolicy: Always
        ports:
        - name: http-api
          containerPort: 3000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
```

Note that the service does _not_ yet contain any Ambassador annotations. This is intentional: we want the service running before we tell Ambassador about it.

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

## 2. Configure Ambassador authentication

Once the auth service is running, we need to tell Ambassador about it. The easiest way to do that is to annotate the `example-auth` service. While we could use `kubectl patch` for this, it's simpler to just modify the service definition and re-apply. Here's the new YAML:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: example-auth
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  AuthService
      name:  authentication
      auth_service: "example-auth:3000"
      path_prefix: "/extauth"
      allowed_headers:
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

This configuration tells Ambassador about the auth service, notably that it
needs the `/extauth` prefix, and that it's OK for it to pass back the
`x-qotm-session` header. Note that `path_prefix` and `allowed_headers` are
optional.

If the auth service uses a framework like
[Gorilla Toolkit](http://www.gorillatoolkit.org) which enforces strict slashes
as HTTP path separators, it is possible to end up with an infinite redirect
where the auth service's framework redirects any request with non-conformant
slashing. This would arise if the above example had
```path_prefix: "/extauth/"```, the auth service would see a request for
```/extauth//qotm/quote/1``` which would then be redirected to
```/extauth/quotm/quote/1``` rather than actually be handled by the
authentication handler. For this reason, remember that the full path of the
incoming request including the leading slash, will be appended to
```path_prefix``` regardless of non-conformant slashing.

You can apply this file from getambassador.io with

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-auth-enable.yaml
```

or, again, apply it from a local file if you prefer.

Ambassador will see the annotations and reconfigure itself within a few seconds.

## 3. Test authentication

If we `curl` to a protected URL:

```shell
$ curl -v $AMBASSADORURL/qotm/quote/1
```

We get a 401, since we haven't authenticated.

```shell
HTTP/1.1 401 Unauthorized
x-powered-by: Express
x-request-id: 9793dec9-323c-4edf-bc30-352141b0a5e5
www-authenticate: Basic realm=\"Ambassador Realm\"
content-type: text/html; charset=utf-8
content-length: 0
etag: W/\"0-2jmj7l5rSw0yVb/vlWAYkK/YBwk\"
date: Fri, 15 Sep 2017 15:22:09 GMT
x-envoy-upstream-service-time: 2
server: envoy
```

If we authenticate to the service, we will get a quote successfully:

```shell
$ curl -v -u username:password $AMBASSADORURL/qotm/quote/1

TCP_NODELAY set
* Connected to 35.196.173.175 (35.196.173.175) port 80 (#0)
* Server auth using Basic with user \'username\'
> GET /qotm/quote/1 HTTP/1.1
> Host: 35.196.173.175
> Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< content-type: application/json
< x-qotm-session: 069da5de-5433-46c0-a8de-d266e327d451
< content-length: 172
< server: envoy
< date: Wed, 27 Sep 2017 18:53:38 GMT
< x-envoy-upstream-service-time: 25
<
{
 \"hostname\": \"qotm-1827164760-gf534\",
 \"ok\": true,
 \"quote\": \"A late night does not make any sense.\",
 \"time\": \"2017-09-27T18:53:39.376073\",
 \"version\": \"1.1\"
}
```

## More

For more details about configuring authentication, read the documentation on [external authentication](/reference/services/auth-service).
