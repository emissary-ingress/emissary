package runner

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/jwks"
	"github.com/datawire/apro/lib/util"
)

// NewIDP returns an instance of the identity provider server.
func NewIDP() *httptest.Server {
	var serverURL *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			authREQ := oauth2handler.AuthorizationRequest{}

			ct := r.Header.Get("Content-Type")
			if ct == "" {
				log.Fatal("test IDP server: No Content-Type header")
			}
			mt, _, err := mime.ParseMediaType(ct)
			if err != nil {
				log.Fatalf("test IDP server: Could not parse Content-Type header: %q", ct)
			}
			switch mt {
			case "application/x-www-form-urlencoded", "multipart/form-data":
				authREQ = oauth2handler.AuthorizationRequest{
					GrantType:    r.PostFormValue("grant_type"),
					ClientID:     r.PostFormValue("client_id"),
					Code:         r.PostFormValue("code"),
					RedirectURL:  r.PostFormValue("redirect_uri"),
					ClientSecret: r.PostFormValue("client_secret,omitempty"),
					Audience:     r.PostFormValue("audience,omitempty"),
				}
			case "application/json":
				decoder := json.NewDecoder(r.Body)
				if err = decoder.Decode(&authREQ); err != nil {
					log.Fatal(errors.Wrapf(err, "test IDP server: malformed JSON in POST"))
				}
			default:
				log.Fatalf("test IDP server: Unsupported media type: %q", mt)
			}

			if authREQ.Code == "authorize" {
				util.ToJSONResponse(w, http.StatusOK, &oauth2handler.AuthorizationResponse{
					AccessToken:  "mocked_token_123",
					IDToken:      "mocked_id_token_123",
					TokenType:    "Bearer",
					RefreshToken: "mocked_refresh_token_123",
					ExpiresIn:    time.Now().Add(time.Minute * 2).Unix(),
				})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		case "/.well-known/openid-configuration":
			util.ToJSONResponse(w, http.StatusOK, &oauth2handler.OpenIDConfig{
				Issuer:                fmt.Sprintf("%s://%s/", serverURL.Scheme, serverURL.Host),
				AuthorizationEndpoint: "TODO://AuthorizationEndpoint",
				TokenEndpoint:         fmt.Sprintf("%s://%s/oauth/token", serverURL.Scheme, serverURL.Host),
				JSONWebKeySetURI:      fmt.Sprintf("%s://%s/.well-known/jwks.json", serverURL.Scheme, serverURL.Host),
			})
		case "/.well-known/jwks.json":
			util.ToJSONResponse(w, http.StatusOK, map[string]interface{}{
				"keys": []jwks.JWK{
					// TODO
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL, _ = url.Parse(server.URL)
	return server
}

// NewAPP returns an instance of the authorization server.
func NewAPP(idpURL string, tb testing.TB) (*httptest.Server, http.Handler) {
	os.Setenv("RLS_RUNTIME_DIR", "/bogus")

	c, warn, fatal := types.ConfigFromEnv()
	if len(fatal) > 0 {
		tb.Fatal(fatal[len(fatal)-1])
	}
	if len(warn) > 0 {
		tb.Fatal(warn[len(warn)-1])
	}

	_l := logrus.New()
	_l.SetLevel(logrus.DebugLevel)
	l := types.WrapLogrus(_l)

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithField("test", "unit"),
	}

	httpHandler, err := app.NewFilterMux(c, l, ct)
	if err != nil {
		tb.Fatal(err)
	}
	server := httptest.NewServer(httpHandler)

	filters := map[string]interface{}{
		"dummy.default": crd.FilterOAuth2{
			RawAuthorizationURL: idpURL,
			RawClientURL:        "http://dummy-host.net",
			Audience:            "foo",
			ClientID:            "bar",
		},
		"app.default": crd.FilterOAuth2{
			RawAuthorizationURL: idpURL,
			RawClientURL:        server.URL,
			Audience:            "friends",
			ClientID:            "foo",
		},
	}
	for k, _v := range filters {
		v := _v.(crd.FilterOAuth2)
		v.Validate()
		filters[k] = v
	}
	ct.Filters.Store(filters)
	ct.Rules.Store([]crd.Rule{
		{
			Host: "*",
			Path: "*",
			Filters: []crd.FilterReference{
				{
					Name:      "app",
					Namespace: "default",
					Arguments: crd.FilterOAuth2Arguments{
						Scopes: []string{},
					},
				},
			},
		},
	})

	return server, httpHandler
}
