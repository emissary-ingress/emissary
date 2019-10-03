package oauth2handler

import (
	"net/http"

	"github.com/pkg/errors"

	rfc6750client "github.com/datawire/liboauth2/client/rfc6750"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func (j *OAuth2Filter) validateAccessTokenUserinfo(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
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
