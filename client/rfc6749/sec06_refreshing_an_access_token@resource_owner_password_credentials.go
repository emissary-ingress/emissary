package rfc6749

// Refresh talks to the Authorization Server to exchange a Refresh Token for an Access Token (and
// maybe a new Refresh Token); per §6.
//
// If the authorization flow for the session has not yet been completed, ErrNoAccessToken is
// returned.  If the Authorization Server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the Authorization Server sent a semantically valid error
// response, an error of type TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
//
// It is not normally nescessary to call this method; it is called automatically by
// `.AuthorizationForResourceRequest()`.  It should only be nescessary to call this method if the
// Client has a reason to believe that that the AccessToken has prematurely expired before the
// `expires_in` timestamp returned by the Authorization Server.
func (client *ResourceOwnerPasswordCredentialsClient) Refresh(
	session *ResourceOwnerPasswordCredentialsClientSessionData,
	scope Scope,
) error {
	return client.refresh(session, scope)
}
