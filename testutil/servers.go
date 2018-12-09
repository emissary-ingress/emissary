package testutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/util"
)

// NewIDPTestServer returns an instance of the identity provider server.
func NewIDPTestServer() *httptest.Server {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(gsagula): This should return `discovery.JWK` object with out app public keys so
		// we can call the authorization server with a valid access token.
		// if r.URL.Path == "/.well-known" {
		//
		// }

		if r.URL.Path == "/oauth/token" {
			authREQ := &client.AuthorizationRequest{}
			body, err := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				log.Fatal(err)
			}

			if err = json.Unmarshal(body, authREQ); err != nil {
				log.Fatal(err)
			}

			if authREQ.Code == "authorize" {
				util.ToJSONResponse(w, http.StatusOK, &client.AuthorizationResponse{
					AccessToken:  "mocked_token_123",
					IDToken:      "mocked_id_token_123",
					TokenType:    "Bearer",
					RefreshToken: "mocked_refresh_token_123",
					ExpiresIn:    time.Now().Add(time.Minute * 2).Unix(),
				})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	})

	return httptest.NewServer(h)
}

// NewAPPTestServer returns an instance of the authorization server.
func NewAPPTestServer(idpURL string) (*httptest.Server, *app.App) {
	u, err := url.Parse(idpURL)
	if err != nil {
		panic(err)
	}

	os.Setenv("AUTH_AUDIENCE", "friends")
	os.Setenv("AUTH_DOMAIN", fmt.Sprintf("%s:%s", u.Hostname(), u.Port()))
	os.Setenv("AUTH_CALLBACK_URL", fmt.Sprintf("%s/callback", idpURL))
	os.Setenv("AUTH_CLIENT_ID", "foo")

	c := config.New()

	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)

	ct := &controller.Controller{
		Config: c,
		Logger: l,
	}
	ct.Rules.Store(make([]controller.Rule, 0))

	cl := client.NewRestClient(u)

	app := &app.App{
		Config:     c,
		Logger:     l,
		Secret:     s,
		Discovery:  d,
		Controller: ct,
		Rest:       cl,
	}

	return httptest.NewServer(app.Handler()), app
}
