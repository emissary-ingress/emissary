// Package rfc6749 implements the "Client" role of the OAuth 2.0 Framework.
//
// A Client application (that is, an application that receives an Access Token from an Authorization
// Server, and uses that Access Token to access resources on a Resource Server) will make use of
// this package by creating an instance of one of the 4 Client types:
//
//  1. AuthorizationCodeClient
//  2. ImplicitClient
//  3. ResourceOwnerPasswordCredentialsClient
//  4. ClientCredentialsClient
//
// After creating the Client object, but before using it, applications will need to register the
// OAuth 2.0 protocol extensions that they will be using, by calling `.RegisterProtocolExtensions()`
// on the Client object.
//
// Once the Client object has been initialized, the application will call the (client-type-specific)
// method to initiate the authorization flow, which will return a struct containing session data.
// Once the authorization flow has been completed (which may be a multi-step process, depending on
// the client-type), individual requests to the Resource Server can be authorized by calling
// `.AuthorizationForResourceRequest()`, and responses from the Resource Server can be checked for
// authorization errors with `.ErrorFromResourceResponse()`:
//
//   Note: These are pseudo-code examples, to illustrate the high-level flow for
//   each client type.  The full function signatures are not reflected in what
//   is shown below.
//
//     client := NewAuthorizationCodeClient()
//         client.RegisterProtocolExtensions(ext1, ext2)
//         // client is now initialized
//         session, request := client.AuthorizationRequest()
//         response := do(request)
//         authcode := client.ParseAuthorizationResponse(session, response)
//         client.AccessToken(session, authcode)
//         // authorization flow is now completed
//         authorization := client.AuthorizationForResourceRequest(session)
//         response := do(resourceRequest(authorization))
//         err := client.ErrorFromResourceResponse(response)
//
//     client := NewImplicitClient()
//         client.RegisterProtocolExtensions(ext1, ext2)
//         // client is now initialized
//         session, request := client.AuthorizationRequest()
//         response := do(request)
//         client.ParseAuthorizationResponse(session, response)
//         // authorization flow is now completed
//         authorization := client.AuthorizationForResourceRequest(session)
//         response := do(resourceRequest(authorization))
//         err := client.ErrorFromResourceResponse(response)
//
//     client := NewResourceOwnerPAsswordCredentialsClient()
//         client.RegisterProtocolExtensions(ext1, ext2)
//         // client is now initialized
//         session := client.AuthorizationRequest()
//         // authorization flow is now completed
//         authorization := client.AuthorizationForResourceRequest(session)
//         response := do(resourceRequest(authorization))
//         err := client.ErrorFromResourceResponse(response)
//
//     client := NewClientCredentialsClient()
//         client.RegisterProtocolExtensions(ext1, ext2)
//         // client is now initialized
//         session := client.AuthorizationRequest()
//         // authorization flow is now completed
//         authorization := client.AuthorizationForResourceRequest(session)
//         response := do(resourceRequest(authorization))
//         err := client.ErrorFromResourceResponse(response)
//
//
// type TokenErrorResponse struct{ ... }
// type TokenResponse struct{ ... }
//
// type ClientAuthenticationMethod func(header http.Header, body url.Values) // used as an argument to all client constructors except NewImplicitClient
// type Scope map[string]struct{}                                            // used all over the place
// type UnsupportedTokenTypeError struct{ ... }                              // used all over the place
//
// type AuthorizationCodeClient struct{ ... }
//
//     func (client *AuthorizationCodeClient) AuthorizationRequest(*url.URL, Scope, string) =>                   type AuthorizationCodeClientSessionData struct{ ... }
//     func (client *AuthorizationCodeClient) ParseAuthorizationResponse(...)               => error might be of type AuthorizationCodeGrantErrorResponse struct{ ... }
//     func (client *AuthorizationCodeClient) AccessToken(...)                              => error might be of type TokenErrorResponse
//     func (client *AuthorizationCodeClient) ErrorFromResourceResponse()                   =>                   type ReifiedResourceAccessErrorResponse struct{ ... }
//
//     func (client *AuthorizationCodeClient) Refresh(...)                                  => error might be of type TokenErrorResponse
//
// type ClientCredentialsClient struct{ ... }
//
//     func (client *ClientCredentialsClient) AuthorizationRequest(*url.URL, Scope, string) =>                   type ClientCredentialsClientSessionData struct{ ... }
//                                                                                          => error might be of type TokenErrorResponse
//     func (client *ClientCredentialsClient) ParseAuthorizationResponse(...)               => error might be of type ImplicitGrantErrorResponse struct{ ... }
//     func (client *ClientCredentialsClient) ErrorFromResourceResponse()                   =>                   type ReifiedResourceAccessErrorResponse struct{ ... }
//
//     func (client *ClientCredentialsClient) Refresh(...)                                  => error might be of type TokenErrorResponse
//
// type ImplicitClient struct{ ... }
//
//     func (client *ImplicitClient) AuthorizationRequest()                                 =>                   type ImplicitClientSessionData struct{ ... }
//     func (client *ImplicitClient) ErrorFromResourceResponse()                            =>                   type ReifiedResourceAccessErrorResponse struct{ ... }
//
// type ResourceOwnerPasswordCredentialsClient struct{ ... }
//
//     func (client *ResourceOwnerPasswordCredentialsClient) AuthorizationRequest()         =>                   type ResourceOwnerPasswordCredentialsClientSessionData struct{ ... }
//                                                                                          => error might be of type TokenErrorResponse
//     func (client *ResourceOwnerPasswordCredentialsClient) ErrorFromResourceResponse()    =>                   type ReifiedResourceAccessErrorResponse struct{ ... }
//
//     func (client *ResourceOwnerPasswordCredentialsClient) Refresh(...)                   => error might be of type TokenErrorResponse
//
// type ProtocolExtension struct{ ... }
//     type AccessTokenType struct{ ... }
//         type ResourceAccessErrorResponse interface{ ... }
//     type ExtensionError struct{ ... }
//         type ErrorUsageLocation interface{ ... }
package rfc6749
