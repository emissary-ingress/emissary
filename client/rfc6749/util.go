package rfc6749client

import (
	"net/http"
	"net/url"
	"strings"
)

type explicitClient struct {
	tokenEndpoint        *url.URL
	clientAuthentication ClientAuthenticationMethod
}

// postForm is the common bits of request/response handling per
// §4.1.3/§4.1.4, §4.3.2/§4.3.3, §4.4.2/§4.4.3, and §6.  I'm not a
// huge fan of it being factored out here, instead of being duplicated
// in sec4_{1,3,4}_*.go and sec6_*.go.  But that's the only sane way I
// could figure to structure it such that the refresh API is sane.
func (client *explicitClient) postForm(httpClient *http.Client, form url.Values) (TokenResponse, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	header := make(http.Header)

	if client.clientAuthentication != nil {
		client.clientAuthentication(header, form)
	}

	req, err := http.NewRequest("POST", client.tokenEndpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header = header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return parseTokenResponse(res)
}
