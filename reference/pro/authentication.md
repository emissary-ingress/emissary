# Ambassador Pro Authentication

## Overview

Ambassador Pro supports the Authorization Code Flow authentication flow.  On an incoming request, Ambassador Pro will look up session information based on a cookie called `ambassador_session.NAME.NAMESPACE`, where `NAME` and `NAMESPACE` describe the [`Filter` resource](../filter-reference#filter-type-oauth2) being used.  If the cookie is not present, refers to an expired session, or refers to a not-yet-authorized session, then Ambassador Pro will set the cookie and redirect the request to an IDP for user authentication.  Upon a successful authentication by the IDP, Ambassador Pro will mark the session as authorized, and redirect to the originally requested resource.  Depending on the [`accessTokenValidation` Filter setting](../filter-reference#oauth2-global-arguments) subsequent requests may be validated directly by Ambassador Pro without requiring an additional query to the IDP, or may be validated by making requests to the IDP.

## OAuth 2.0 protocol

The Ambassador Pro OAuth2 filter does two things:

* It is an OAuth Client, which fetches resources from the Resource Server on the user's behalf.
* It is half of a Resource Server, validating the Access Token before allowing the request through to the upstream service, which implements the other half of the Resource Server.

This is different from most OAuth implementations where the Authorization Server and the Resource Server are in the same security domain. With Ambassador Pro, the Client and the Resource Server are in the same security domain, and there is an independent Authorization Server.

## XSRF protection

The `ambassador_session.NAME.NAMESPACE` cookie is an opaque string that should be used as an XSRF token.  Applications wishing to leverage Ambassador Pro in their XSRF attack protection should take two extra steps:

 1. When generating an HTML form, the server should read the cookie, and include a `<input type="hidden" name="_xsrf" value="COOKIE_VALUE" />` element in the form.
 2. When handling submitted form data should verify that the form value and the cookie value match.  If they do not match, it should refuse to handle the request, and return an HTTP 4XX response.

Applications using request submission formats other than HTML forms should perform analogous steps of ensuring that the value is duplicated in the cookie and in the request body.

## Redis

Ambassador Pro relies on Redis to store short-lived authentication credentials and rate limiting information. If the Redis data store is lost, users will need to log back in and all existing rate limits would be reset.
