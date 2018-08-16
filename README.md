
## Configuring the service:

The service depends on two environment variables: `AUTH0_DOMAIN`, and
`AUTH0_AUDIENCE`. You an either set these directly or put them in a
`.env` file.

XXX: the function named "policy" in main.go will become another source
of configuration, right now it is hardcoded. Read it and its comment
for more info.

## Running the service:

1. Install Ambassador.
1. First build a docker image: `docker build . -t <blah>:<bleh>`
2. Write a k8s deployment manifests that either sets the environment variables or
mounts a .env file appropriately.
3. Create an Auth0 account. Then, click on the API and create a new API.
   * AUTH0_DOMAIN example is foo.auth0.com.
   * AUTH0_AUDIENCE is listed on the API page https://manage.auth0.com/#/apis
4. Write a service with ambassador annotation that looks something like this. Note the `allowed_headers` stuff, that is
   important:

   ```
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: auth
     annotations:
      getambassador.io/config: |
        ---
        apiVersion: ambassador/v0
        kind:  AuthService
        name:  authentication
        auth_service: auth
        allowed_headers:
         - "Authorization"
         - "Client-Id"
         - "Client-Secret"
   spec:
     selector:
       deployment: auth
     ports:
     - protocol: TCP
       port: 80
       targetPort: 8080
```

   XXX: In practice we probably want to deploy this as a sidecar, but I'm
   not 100% sure how we would tell ambassador to route to that. I'm
   guessing maybe setting auth_service to `localhost:8080` would work but
   we'd have to make sure that doesn't conflict with anything else on the
   pod.


Step 5: Deploy the test backend service: `kubectl apply -f backend.yaml`

Step 6: Do some curls with different kinds of credentials (or no
credentials) against `<host>/backend/public/`,
`<host>/backend/private/`, and `<host>/backend/private-scoped/` to see
different behaviors.

You have two options for authentication. You can acquire a jwt on the
client side somehow (either manually for use with curl or
programatically) and supply it directly in the `Authorization` header,
e.g.: `curl -H "Authorization: Bearer ${JWT}" <url>`

Your other option is to create a `Client-Id` and `Client-Secret` (this
is auth0's terminology for API keys). You can then supply them in the
headers of the same name, e.g.: `curl -H "Client-Id: ${ID}" -H "Client-Secret: ${SECRET}"`

Note that the JWTs will expire quickly, so the first method is not
suitable for scripts, that's what the second method is for.

In the second case the auth service will transform the client-id and
client-secret into a jwt, validate it, and make it available to the
backend, so the backend service shouldn't see any difference between
these two methods.
