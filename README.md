# liboauth2: A Go library for OAuth 2.0 and related specifications

The notable feature of this library compared to other OAuth libraries
is that its structure closely mimics the structure of the
corresponding specifications.  Want the bare OAuth 2.0 framework, to
build your own protocol?  Just use the `rfc6749` package by itself.
Want to opt-in to also using bearer tokens?  Also import `rfc6750`.
Every function or type clearly indicates which section of which spec
it is based on; there should be no questions about what optional
specified behavior is included or not.

Additionally, the APIs are designed to cleanly distinguish between
which things are meant to be used by the Authorization Server, the
Client, or the Resource Server.


### Client overview

Set up:

 1. Create a new Client object with `client := rfc6749.New{FLOW}Client()` where
    `{FLOW}` is the appropriate client type for your application.  (See RFC 6749
    for a discussion of client types).
 2. Add desired protocol extensions from other packages to the client with
    `client.RegisterProtocolExtensions(...)`.

Use:

 1. Obtain a Session object by complete the authorization flow for the specific
    Client type; see the docs for the
    `github.com/datawire/liboauth2/client/rfc6749` package.
 2. Inject the headers from `client.AuthorizationForResourceRequest(session)` in
    to any requests you make to the resource server.

### Resource Server overview

The OAuth 2.0 specification itself (RFC 6749) provides very little useful
information for resource servers, which is a shame.

Resource servers will likely want to extract RFC 6750 Bearer tokens from the
HTTP header (`rfc6750.GetFromHeader()`) and validate that token as a JWT.

### Authorization Server overview

Authorization server functionality is not implemented at this time.

## Package listing

### Primary specifications

| Specification | Name                    | Client package                                  | Resource Server package                                | Authorization Server package |
|---------------|-------------------------|-------------------------------------------------|--------------------------------------------------------|------------------------------|
| RFC 6749      | OAuth 2.0               | `github.com/datawire/liboauth2/client/rfc6749`  | not implemented                                        | not implemented              |
| RFC 6750      | OAuth 2.0 Bearer tokens | `github.com/datawire/liboauth2/client/rfc6750`  | `github.com/datawire/liboauth2/resourceserver/rfc6750` | N/A                          |
| OIDC Core     | OIDC Core               | `github.com/datawire/liboauth2/client/oidccore` | not implemented                                        | not implemented              |

### Dependency specifications

This should include all normative references of the primary
specifications.

#### RFC 6749
| Specification                                                    | Name                                          | Package                              | Referenced by                                                       |
|------------------------------------------------------------------|-----------------------------------------------|--------------------------------------|---------------------------------------------------------------------|
| RFC 2119                                                         | Key words for RFCs                            | N/A                                  | RFC 6749                                                            |
| RFC 2246                                                         | TLS 1.0                                       | `net/http`                           | RFC 6749                                                            |
| RFC 2616                                                         | HTTP/1.1                                      | obsoleted by RFC 7230 - RFC 7235     | RFC 6749                                                            |
| RFC 2617 (Basic)                                                 | HTTP Basic Authentication                     | obsoleted by RFC 7617                | RFC 6749                                                            |
| RFC 2617 (framework)                                             | HTTP Authentication                           | obsoleted by RFC 7235                | RFC 6749                                                            |
| RFC 2618                                                         | HTTP over TLS                                 | obsoleted by RFC 7230                | RFC 6749                                                            |
| RFC 3629                                                         | UTF-8                                         | built-in                             | RFC 6749                                                            |
| RFC 3986                                                         | URI                                           | `net/uri`                            | RFC 6749                                                            |
| RFC 4627                                                         | JSON                                          | `net/json`                           | RFC 6749                                                            |
| RFC 4949                                                         | Internet Security Glossary v2                 | N/A                                  | RFC 6749                                                            |
| RFC 5226                                                         | Guidlines for Writing an RFC                  | N/A                                  | RFC 6749                                                            |
| RFC 5234                                                         | ABNF                                          | N/A                                  | RFC 6749                                                            |
| RFC 5246                                                         | TLS 1.2                                       | `net/http`                           | RFC 6749                                                            |
| RFC 6125                                                         | PKIX with TLS                                 | `net/http`                           | RFC 6749                                                            |
| USASCII                                                          | ASCII                                         | N/A?                                 | RFC 6749                                                            |
| [W3C.REC-html401-19991224][] (application/x-www-form-urlencoded) | HTML 4.01 (application/x-www-form-urlencoded) | obsoleted by W3C.REC-html52-20171214 | RFC 6749                                                            |
| [W3C.REC-xml-20081126][] (Unicode handling)                      | XML 1.0 (Unicode handling)                    | N/A?                                 | RFC 6749                                                            |
|------------------------------------------------------------------|-----------------------------------------------|--------------------------------------|---------------------------------------------------------------------|
| RFC 7230                                                         | HTTP/1.1 Syntax                               | `net/http`                           | RFC 6749 (via RFC 2616)                                             |
| RFC 7231                                                         | HTTP/1.1 Semantics                            | `net/http`                           | RFC 6749 (via RFC 2616)                                             |
| RFC 7232                                                         | HTTP/1.1 Conditional Requests                 | N/A?                                 | RFC 6749 (via RFC 2616)                                             |
| RFC 7233                                                         | HTTP/1.1 Range Requests                       | N/A?                                 | RFC 6749 (via RFC 2616)                                             |
| RFC 7234                                                         | HTTP/1.1 Caching                              | not implemented                      | RFC 6749 (via RFC 2616)                                             |
| RFC 7235                                                         | HTTP/1.1 Authentication                       | TODO?                                | RFC 6749 (via RFC 2616, RFC 2617)                                   |
| RFC 7616                                                         | HTTP Basic Authentication                     | `net/http`                           | RFC 6749 (via RFC 2617)                                             |
| [W3C.REC-html52-20171214][] (application/x-www-form-urlencoded)  | HTML 5.2 (application/x-www-form-urlencoded)  | defers to WHATWG.URL                 | RFC 6749 (via W3C.REC-html401-19991224)                             |
| [WHATWG.URL][]                                                   | URL                                           | `net/url`                            | RFC 6749 (via W3C.REC-html52-20171214 via W3C.REC-html401-19991224) |

#### RFC 6750
| Specification                                                    | Name                                          | Package                              | Referenced by                                                       |
|------------------------------------------------------------------|-----------------------------------------------|--------------------------------------|---------------------------------------------------------------------|
| RFC 2119                                                         | Key words for RFCs                            | N/A                                  | RFC 6750                                                            |
| RFC 2246                                                         | TLS 1.0                                       | `net/http`                           | RFC 6750                                                            |
| RFC 2616                                                         | HTTP/1.1                                      | obsoleted by RFC 7230 - RFC 7235     | RFC 6750                                                            |
| RFC 2617 (framework)                                             | HTTP Authentication                           | obsoleted by RFC 7235                | RFC 6750                                                            |
| RFC 2618                                                         | HTTP over TLS                                 | obsoleted by RFC 7230                | RFC 6750                                                            |
| RFC 3986                                                         | URI                                           | `net/uri`                            | RFC 6750                                                            |
| RFC 5234                                                         | ABNF                                          | N/A                                  | RFC 6750                                                            |
| RFC 5246                                                         | TLS 1.2                                       | `net/http`                           | RFC 6750                                                            |
| RFC 5280                                                         | PKIX CRL                                      | `net/http`                           | RFC 6750                                                            |
| RFC 6265                                                         | HTTP Cookies                                  | `net/http`                           | RFC 6750                                                            |
| RFC 6749                                                         | OAuth 2.0                                     | see above                            | RFC 6750                                                            |
| USASCII                                                          | ASCII                                         | N/A?                                 | RFC 6750                                                            |
| [W3C.REC-html401-19991224][] (application/x-www-form-urlencoded) | HTML 4.01 (application/x-www-form-urlencoded) | obsoleted by W3C.REC-html52-20171214 | RFC 6750                                                            |
| [W3C.REC-webarch-20041215][]                                     | WWW Architecture                              | N/A                                  | RFC 6750                                                            |
|------------------------------------------------------------------|-----------------------------------------------|--------------------------------------|---------------------------------------------------------------------|
| RFC 7230                                                         | HTTP/1.1 Syntax                               | `net/http`                           | RFC 6750 (via RFC 2616)                                             |
| RFC 7231                                                         | HTTP/1.1 Semantics                            | `net/http`                           | RFC 6750 (via RFC 2616)                                             |
| RFC 7232                                                         | HTTP/1.1 Conditional Requests                 | N/A?                                 | RFC 6750 (via RFC 2616)                                             |
| RFC 7233                                                         | HTTP/1.1 Range Requests                       | N/A?                                 | RFC 6750 (via RFC 2616)                                             |
| RFC 7234                                                         | HTTP/1.1 Caching                              | not implemented                      | RFC 6750 (via RFC 2616)                                             |
| RFC 7235                                                         | HTTP/1.1 Authentication                       | TODO?                                | RFC 6750 (via RFC 2616, RFC 2617)                                   |
| [W3C.REC-html52-20171214][] (application/x-www-form-urlencoded)  | HTML 5.2 (application/x-www-form-urlencoded)  | defers to WHATWG.URL                 | RFC 6750 (via W3C.REC-html401-19991224)                             |
| [WHATWG.URL][]                                                   | URL                                           | `net/url`                            | RFC 6750 (via W3C.REC-html52-20171214 via W3C.REC-html401-19991224) |

[W3C.REC-html401-19991224]: http://www.w3.org/TR/1999/REC-html401-19991224
[W3C.REC-html52-20171214]: https://www.w3.org/TR/2017/REC-html52-20171214
[W3C.REC-xml-20081126]: http://www.w3.org/TR/2008/REC-xml-20081126
[WHATWG.URL]: https://url.spec.whatwg.org/
[W3C.REC-webarch-20041215]: http://www.w3.org/TR/2004/REC-webarch-20041215
