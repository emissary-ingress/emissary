# Auth with an External Auth Service

As an alternative to Ambassador's built-in authentication service, you can direct Ambassador to use an external authentication service. If set up this way, Ambassador will query the auth service on every request. It is up to the service to decide whether to accept the request or how to reject it.

## Ambassador External Auth API

When Ambassador is configured to use the external auth service, the method and headers of every incoming request are forwarded to the auth service, with two changes:

1. The `Host` header is overwritten with the host information of the external auth service.
2. The body is removed.

So, for example, if the incoming request is 

```
PUT /path/to/service HTTP/1.1
Host: myservice.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 27

{ "greeting": "hello world!", "spiders": "OMG no "}
```

then the request Ambassador will make of the auth service is:

```
PUT /path/to/service HTTP/1.1
Host: extauth.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 0
```

**ALL** request methods will be proxied; the auth service should be able to handle any request that any client could make. If desired, Ambassador can add a prefix to the path before forwarding it to the auth service; see the example below.

If Ambassador cannot reach the auth service, it returns 503 to the client. If Ambassador receives any response from the auth service other than 200, it returns that full response (header and body) to the client. Ambassador assumes the auth service will return an appropriate response, such as 401.

If the auth service response code is 200, then Ambassador allows the client request to resume being processed by the normal Ambassador Envoy flow. This typically means that the client will receive the expected response to its request.

Additionally, Ambassador can be configured to allow headers from the auth service to be passed back to the client when the auth service returns 200; see the example below.

## Example

See [the Getting Started document](#../user-guide/getting-started.md) for an example.
