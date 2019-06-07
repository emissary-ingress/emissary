package rfc6749

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrNoAccessToken indicates that `.AuthorizationForResourceRequest()` or `.Refresh()` was
	// called with session data that has not had Access Token data added to it.  This probably
	// indicates a programming error; trying to call one of those functions before the
	// authorization flow has been completed.
	ErrNoAccessToken = errors.New("no Access Token data")

	// ErrNoRefreshToken indicates that `.Refresh()` was called but the Authorization Server did
	// not give us a Refresh Token to use.
	ErrNoRefreshToken = errors.New("no Refresh Token")

	// ErrAccessTokenExpired indicates that the Access Token is expired, and could not be
	// refreshed.
	ErrAccessTokenExpired = errors.New("Access Token expired")
)

type accessTokenData struct {
	AccessToken  string
	TokenType    string
	ExpiresAt    time.Time
	RefreshToken *string
	Scope        Scope
}

type clientSessionData interface {
	currentAccessToken() *accessTokenData
	setDirty()
	IsDirty() bool
}

type explicitClient struct {
	tokenEndpoint        *url.URL
	clientAuthentication ClientAuthenticationMethod
	httpClient           *http.Client
}

// postForm is the common bits of request/response handling per §4.1.3/§4.1.4, §4.3.2/§4.3.3,
// §4.4.2/§4.4.3, and §6.  I'm not a huge fan of it being factored out here, instead of being
// duplicated in sec4_{1,3,4}_*.go and sec6_*.go.  But that's the only sane way I could figure to
// structure it such that the refresh API is sane.
func (client *explicitClient) postForm(form url.Values) (TokenResponse, error) {
	header := make(http.Header)

	if client.clientAuthentication != nil {
		client.clientAuthentication(header, form)
	}

	req, err := http.NewRequest("POST", client.tokenEndpoint.String(),
		strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header = header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer res.Body.Close()

	return parseTokenResponse(res)
}
