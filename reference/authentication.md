# Ambassador Edge Stack Authentication

The Ambassador Edge Stack can operate as an Identity Aware Proxy as part of a Zero Trust security architecture in Kubernetes. In the traditional network-oriented security model, trust is assumed by your network location:  if you’re inside the network (e.g., behind the firewall), you’re trusted. If you’re outside the network, you’re not trusted until you join the network (e.g., by connecting via VPN). This trust model does not extend well to modern networks:

* There is no “defense in depth.” Once an attacker breaches the (network) perimeter, an attacker in this model has full access to all internal resources.
* Modern infrastructure is fully distributed: cloud providers, on-premises data centers, laptops, mobile devices. Creating a strong perimeter across this diffuse edge is impractical.

In the zero trust model, every request to a resource is verified, regardless of where that request originates. Google was one of the first companies to deploy a complete zero trust architecture, as detailed in their [BeyondCorp security architecture whitepaper](https://ai.google/research/pubs/pub43231).

## Identity-Aware Proxy

One of the key components of a zero trust architecture is the Identity-Aware Proxy. The Ambassador Edge Stack can be deployed in front of an application or microservices, and authenticate users, check authorization, and enforce other types of security policies. Critically, the Ambassador Edge Stack operates at the application level, which means it can take advantage of domain knowledge of users to improve security. Ambassador interfaces with the Identity Provider (IdP), which is the trusted canonical source for authentication and authorization information.

![IAP](../../doc-images/pro-iap.png)

## Integrating with IdPs

The Ambassador integrates with Identity Providers using OpenID Connect and OAuth2. In particular, the Ambassador Edge Stack supports the Authorization Code Flow authentication flow.  On an incoming request, the Ambassador Edge Stack will look up session information based on a cookie called `ambassador_session.NAME.NAMESPACE`, where `NAME` and `NAMESPACE` describe the [`Filter` resource](../../reference/filter-reference#filter-type-oauth2) being used. If the cookie is not present, refers to an expired session, or refers to a not-yet-authorized session, then the Ambassador Edge Stack will set the cookie and redirect the request to an IDP for user authentication.  Upon a successful authentication by the IDP, the Ambassador Edge Stack will mark the session as authorized, and redirect to the originally requested resource.  Depending on the [`accessTokenValidation` Filter setting](../../reference/filter-reference#oauth2-global-arguments) subsequent requests may be validated directly by the Ambassador Edge Stack without requiring an additional query to the IDP, or may be validated by making requests to the IDP.

## OAuth 2.0 protocol

The Ambassador Edge Stack OAuth2 filter does two things:

* It is an OAuth Client, which fetches resources from the Resource Server on the user's behalf.
* It is half of a Resource Server, validating the Access Token before allowing the request through to the upstream service, which implements the other half of the Resource Server.

This is different from most OAuth implementations where the Authorization Server and the Resource Server are in the same security domain. With the Ambassador Edge Stack, the Client and the Resource Server are in the same security domain, and there is an independent Authorization Server.

## XSRF protection

The `ambassador_xsrf.NAME.NAMESPACE` cookie is an opaque string that should be used as an XSRF token.  Applications wishing to leverage the Ambassador Edge Stack in their XSRF attack protection should take two extra steps:

 1. When generating an HTML form, the server should read the cookie, and include a `<input type="hidden" name="_xsrf" value="COOKIE_VALUE" />` element in the form.
 2. When handling submitted form data should verify that the form value and the cookie value match.  If they do not match, it should refuse to handle the request, and return an HTTP 4XX response.

Applications using request submission formats other than HTML forms
should perform analogous steps of ensuring that the value is present
in the request duplicated in the cookie and in either the request body
or secure header field.  A secure header field is one that is not
`Cookie`, is not "[simple][simple-header]", and is not explicitly
allowed by the CORS policy.

[simple-header]: https://www.w3.org/TR/cors/#simple-header

**Note**: Prior versions of the Ambassador Edge Stack did not have a
`ambassador_xsrf.NAME.NAMESPACE` cookie, and instead required you to
use the `ambassador_session.NAME.NAMESPACE` cookie.  The
`ambassador_session.NAME.NAMESPACE` cookie should no longer be used
for XSRF-protection purposes

## RP-initiated logout

When a logout occurs, it is often not enough to delete the Ambassador
Edge Stack's session cookie or session data; after this happens, and the web
browser is redirected to the Identity Provider to re-log-in, the
Identity Provider may remember the previous login, and immediately
re-authorize the user; it would be like the logout never even
happened.

To solve this, the Ambassador Edge Stack can use [OpenID Connect Session
Management][oidc-session] to perform an "RP-Initiated Logout", where the
Ambassador Edge Stack (the OpenID Connect "Relying Party" or "RP")
communicates directly with Identity Providers that support OpenID
Connect Session Management, to properly log out the user.
Unfortunately, many Identity Providers do not support OpenID Connect
Session Management.

[oidc-session]: https://openid.net/specs/openid-connect-session-1_0.html

This is done by having your application direct the web browser `POST`
*and navigate* to `/.ambassador/oauth2/logout`.  There are 2
form-encoded values that you need to include:

 1. `realm`: The `name.namespace` of the `Filter` that you want to log
    out of.  This may be submitted as part of the POST body, or may be set as a URL query parameter.
 2. `_xsrf`: The value of the `ambassador_xsrf.{{realm}}` cookie
    (where `{{realm}}` is as described above).  This must be set in the POST body, the URL query part will not be checked.

For example:

```html
<form method="POST" action="/.ambassador/oauth2/logout" target="_blank">
  <input type="hidden" name="realm" value="myfilter.mynamespace" />
  <input type="hidden" name="_xsrf" value="{{ .Cookie.Value }}" />
  <input type="submit" value="Log out" />
</form>
```

or

```html
<form method="POST" action="/.ambassador/oauth2/logout?realm=myfilter.mynamespace" target="_blank">
  <input type="hidden" name="_xsrf" value="{{ .Cookie.Value }}" />
  <input type="submit" value="Log out" />
</form>
```

or from JavaScript

```js
function getCookie(name) {
    var prefix = name + "=";
    var cookies = document.cookie.split(';');
    for (var i = 0; i < cookies.length; i++) {
        var cookie = cookies[i].trimStart();
        if (cookie.indexOf(prefix) == 0) {
            return cookie.slice(prefix.length);
        }
    }
    return "";
}

function logout(realm) {
    var form = document.createElement('form');
    form.method = 'post';
    form.action = '/.ambassador/oauth2/logout?realm='+realm;
    //form.target = '_blank'; // uncomment to open the identity provider's page in a new tab

    var xsrfInput = document.createElement('input');
    xsrfInput.type = 'hidden';
    xsrfInput.name = '_xsrf';
    xsrfInput.value = getCookie("ambassador_xsrf."+realm);
    form.appendChild(xsrfInput);

    document.body.appendChild(form);
    form.submit();
}
```

## Redis

The Ambassador Edge Stack relies on Redis to store short-lived authentication credentials and rate limiting information. If the Redis data store is lost, users will need to log back in and all existing rate limits would be reset.
