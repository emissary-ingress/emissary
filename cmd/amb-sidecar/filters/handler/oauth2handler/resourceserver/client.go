package resourceserver

import (
	"net/http"

	"github.com/pkg/errors"

	rfc6750client "github.com/datawire/liboauth2/client/rfc6750"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func (rs *OAuth2ResourceServer) validateAccessTokenUserinfo(token string, discovered *discovery.Discovered, httpClient *http.Client, logger types.Logger) error {
	// This method is a little funny, since it has the Resource
	// Server acting like a Client to a different Resource server.

	req, err := http.NewRequest("GET", discovered.UserInfoEndpoint.String(), nil)
	if err != nil {
		return err
	}
	rfc6750client.AddToHeader(token, req.Header)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.Errorf("token validation through userinfo endpoint failed: HTTP %d", res.StatusCode)
	}
	return nil
}
