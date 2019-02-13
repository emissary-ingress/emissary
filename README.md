# Ambassador Pro example middleware

This is an example middleware for Ambassador Pro that injects an
`X-Wikipedia` header with the URL for a random Wikipedia article
before passing the request off to the backend service.  It uses
https://en.wikipedia.org/wiki/Special:Random to obtain the URL; as an
example of a middleware needing to talk to an external service.

## Compiling

Just run

	$ make

It will generate a Docker image named

	localhost:31000/amb-sidecar-plugin:VERSION
	
where version is `git describe --tags --always --dirty`.

 > Note: You may need to edit the `FROM` line in `Dockerfile`.

## Deploying

Use that `amb-sidecar-plugin` Docker image instead of
`quay.io/datawire/ambassador_pro:amb-sidecar` when deploying
Ambassador Pro.


## How a plugin works

In `package main`, define a functions named `PluginMain` with this
signature:

	func PluginMain(w http.ResponseWriter, r *http.Request) { … }

The `*http.Request` is the incoming HTTP request that you have the
opportunity to mutate or intercept, which you do via the
`http.ResponseWriter`.

You may mutate the headers by calling `w.Header().Set(HEADERNAME,
VALUE)`.  Finalize this by calling `w.WriteHeader(http.StatusOK)`

If you call `w.WriteHeader()` with any value other than 200
(`http.StatusOK`) instead of modifying the request, the middleware has
taken over the request.  You can call `w.Write()` to write the body of
an error page.

This is all very similar to writing an Envoy ext_authz authorization
service.
