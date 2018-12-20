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

// NewIDP returns an instance of the identity provider server.
func NewIDP() *httptest.Server {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// NewAPP returns an instance of the authorization server.
func NewAPP(idpURL string) (*httptest.Server, *app.App) {
	u, err := url.Parse(idpURL)
	if err != nil {
		panic(err)
	}

	os.Setenv("AUTH_DOMAIN", u.Hostname())

	c := config.New()
	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithField("test", "unit"),
	}

	apps := make([]controller.Tenant, 2)
	apps[0] = controller.Tenant{
		CallbackURL: "dummy-host.net/callback",
		Domain:      "dummy-host.net",
		Audience:    "foo",
		ClientID:    "bar",
	}
	apps[1] = controller.Tenant{
		CallbackURL: fmt.Sprintf("%s/callback", idpURL),
		Domain:      u.Hostname(),
		Audience:    "friends",
		ClientID:    "foo",
	}

	ct.Apps.Store(apps)
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
