package testutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/logger"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/secret"
	"github.com/datawire/apro/lib/util"
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
	os.Setenv("AUTH_PROVIDER_URL", idpURL)

	c := config.New()
	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithField("test", "unit"),
	}

	tenants := make([]controller.Tenant, 2)
	tenants[0] = controller.Tenant{
		CallbackURL: "dummy-host.net/callback",
		Domain:      "dummy-host.net",
		Audience:    "foo",
		ClientID:    "bar",
	}
	tenants[1] = controller.Tenant{
		CallbackURL: fmt.Sprintf("%s/callback", idpURL),
		Domain:      c.BaseURL.Hostname(),
		Audience:    "friends",
		ClientID:    "foo",
	}

	ct.Tenants.Store(tenants)
	ct.Rules.Store(make([]controller.Rule, 0))

	cl := client.NewRestClient(c.BaseURL)

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
