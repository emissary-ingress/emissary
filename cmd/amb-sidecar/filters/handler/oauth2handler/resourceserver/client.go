package resourceserver

import (
	"net/http"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/pkg/errors"

	rfc6750client "github.com/datawire/apro/client/rfc6750"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
)

func (rs *OAuth2ResourceServer) validateAccessTokenUserinfo(token string, discovered *discovery.Discovered, httpClient *http.Client, logger dlog.Logger) (tokenErr, serverErr error) {
	// This method is a little funny, since it has the Resource
	// Server acting like a Client to a different Resource server.

	req, err := http.NewRequest("GET", discovered.UserInfoEndpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	rfc6750client.AddToHeader(token, req.Header)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err := errors.Errorf("token validation through userinfo endpoint failed: HTTP %d", res.StatusCode)
		if res.StatusCode/100 == 4 {
			return err, nil
		}
		return nil, err
	}
	return nil, nil
}
