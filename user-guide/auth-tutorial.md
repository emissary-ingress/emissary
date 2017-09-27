# Authentication

Ambassador can authenticate incoming requests before routing them to a backing service. In this tutorial, we'll configure Ambassador to use an external third party authentication service.

## Before you get started

This tutorial assumes you have already followed the [Ambassador Getting Started](https://www.getambassador.io/user-guide/getting-started). If you haven't done that already, go do that now.

## 1. Deploy third party authentication service

Ambassador delegates the actual authentication logic to a third party authentication service. We've written a [simple authentication service](https://github.com/datawire/ambassador-auth-service) that:

- listens for requests on port 3000;
- expects all URLs to begin with `/extauth/`;
- performs HTTP Basic Auth for all URLs starting with `/qotm/quote/` (other URLs are always permitted);
- accepts only user `username`, password `password`; and
- makes sure that the `x-qotm-session` header is present, generating a new one if needed.

Deploy the auth service in Kubernetes:

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-auth.yaml
```

## 2. Configure Ambassador authentication

Now, we configure Ambassador to use the authentication service. Once the auth service is running, add the following to the end of your existing `mapping-qotm.yaml` file:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  authentication
config:
  auth_service: "example-auth:3000"
  path_prefix: "/extauth"
  allowed_headers:
  - "x-qotm-session"
```

This configuration tells Ambassador about the auth service, notably that it needs the `/extauth` prefix, and that it's OK for it to pass back the `x-qotm-session` header. Note that `path_prefix` and `allowed_headers` are optional.

## 3. Update the ConfigMap

We need to update the `ConfigMap` with the new configuration.

```shell
$ kubectl create configmap ambassador-config --from-file mapping-qotm.yaml -o yaml --dry-run | \
    kubectl replace -f -
```

## 4. Update Ambassador

We use Kubernetes to do a rolling update (zero downtime) of Ambassador. We do this by updating an annotation on the deployment:

```shell
$ kubectl patch deployment ambassador -p \
  "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"
```

## 5. Test authentication

If we `curl` to a protected URL:

```shell
$ curl -v $AMBASSADORURL/qotm/quote/1
```

We get a 401, since we haven't authenticated.

```shell
HTTP/1.1 401 Unauthorized
x-powered-by: Express
x-request-id: 9793dec9-323c-4edf-bc30-352141b0a5e5
www-authenticate: Basic realm="Ambassador Realm"
content-type: text/html; charset=utf-8
content-length: 0
etag: W/"0-2jmj7l5rSw0yVb/vlWAYkK/YBwk"
date: Fri, 15 Sep 2017 15:22:09 GMT
x-envoy-upstream-service-time: 2
server: envoy
```

If we authenticate to the service, we will get a quote successfully:

```shell
$ curl -v -u username:password $AMBASSADORURL/qotm/quote/1

TCP_NODELAY set
* Connected to 35.196.173.175 (35.196.173.175) port 80 (#0)
* Server auth using Basic with user 'username'
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
 "hostname": "qotm-1827164760-gf534",
 "ok": true,
 "quote": "A late night does not make any sense.",
 "time": "2017-09-27T18:53:39.376073",
 "version": "1.1"
}
```
