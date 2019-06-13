# Ambassador Pro Authentication

## Overview

Ambassador Pro supports the Authorization Code Flow authentication flow. On an incoming request, Ambassador will check for a cookie called `ambassador_session`. If the cookie is not present, Ambassador Pro will redirect the request to an IDP for user authentication. Upon a successful authentication by the IDP, Ambassador Pro will set the cookie. Subsequent requests will be validated directly by Ambassador Pro without requiring an additional query to the IDP.

## OAuth protocol

The Ambassador Pro OAuth2 filter does two things:

* It is an OAuth Client, which feteches resources from the Resource Server on the user's behalf.
* It is half of a Resource Server, validating the Access Token before allowing the request through to the upstream service, which implements the other half of the Resource Server.

This is different from most OAuth implementations where the Authorization Server and the Resource Server are in the same security domain. With Ambassador Pro, the Client and the Resource Server are in the same security domain, and there is an independent Authorization Server.

## XSRF protection

The `ambassador_session` is an opaque string that should be used as an XSRF token. This token is used in `/callback` to prevent XSRF attacks.

## Redis

Ambassador Pro relies on Redis to store short-lived authentication credentials and rate limiting information. If the Redis data store is lost, users will need to log back in and all existing rate limits would be reset.
