# Implementing JWT Validation

JWTs are validated using public keys supplied in a JWKS file. For the purposes of this demo, we're supplying a Datawire JWKS file (and giving you a JWT that we've generated ourselves). You can change the JWKS file by modifying the `jwt-filter.yaml` manifest and changing the `jwksURI` value.

1. Install Ambassador Pro

2. Configure the JWT filter and JWT-authenticated `httpbin` test service:

   ```
   make apply-jwt
   ```

3. Send a valid JWT to the `jwt-httpbin` URL:

   ```
   curl -i --header "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." $AMBASSADOR_IP/jwt-httpbin/ip
   ```

4. Send an invalid JWT, and get a 401:

   ```
   curl -i $AMBASSADOR_IP/jwt-httpbin/ip
   HTTP/1.1 401 Unauthorized
   content-length: 58
   content-type: text/plain
   date: Thu, 28 Feb 2019 01:07:10 GMT
   server: envoy
   ```

5. Note that we've configured the `jwt-httpbin` URL to require JWTs, but the `httpbin` URL does not:

   ```
   curl -v http://$AMBASSADOR_IP/httpbin/ip
   {
      "origin": "108.20.119.124, 35.184.242.212, 108.20.119.124"
   }
   ```

   This policy is set in the `FilterPolicy` object in `jwt-filter.yaml`.

6. We're sending a short, unsigned JWT (hence the only `validAlgorithms` type is `none`). For real-world applications, you'll want to delete the `validAlgorithms` section and supply signed JWTs.
