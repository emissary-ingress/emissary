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

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
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
func NewAPP(idpURL string) (*httptest.Server, *app.App, error) {
	os.Setenv("AUTH_PROVIDER_URL", idpURL)

	flags := pflag.NewFlagSet("newapp", pflag.PanicOnError)
	afterParse := types.InitializeFlags(flags)
	_ = flags.Parse([]string{})

	c := afterParse()
	l := logrus.New()

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithField("test", "unit"),
	}

	tenants := make([]crd.TenantObject, 2)
	tenants[0] = crd.TenantObject{
		CallbackURL: "dummy-host.net/callback",
		Domain:      "dummy-host.net",
		Audience:    "foo",
		ClientID:    "bar",
	}
	tenants[1] = crd.TenantObject{
		CallbackURL: fmt.Sprintf("%s/callback", idpURL),
		Domain:      c.BaseURL.Hostname(),
		Audience:    "friends",
		ClientID:    "foo",
	}

	ct.Tenants.Store(tenants)
	ct.Rules.Store(make([]crd.Rule, 0))

	app := &app.App{
		Config:     c,
		Logger:     l,
		Controller: ct,
	}
	httpHandler, err := app.Handler()
	if err != nil {
		return nil, nil, err
	}

	return httptest.NewServer(httpHandler), app, nil
}
