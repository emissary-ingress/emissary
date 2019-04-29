package runner

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"

	envoyCoreV2 "github.com/datawire/kat-backend/xds/envoy/api/v2/core"
	envoyAuthV2 "github.com/datawire/kat-backend/xds/envoy/service/auth/v2alpha"
	pbTypes "github.com/gogo/protobuf/types"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
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
func NewAPP(idpURL string, tb testing.TB) (*httptest.Server, filterapi.FilterClient) {
	os.Setenv("RLS_RUNTIME_DIR", "/bogus")

	c, warn, fatal := types.ConfigFromEnv()
	if len(fatal) > 0 {
		tb.Fatal(fatal[len(fatal)-1])
	}
	if len(warn) > 0 {
		tb.Fatal(warn[len(warn)-1])
	}

	l := types.WrapTB(tb)

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
			RawClientURL:        "http://lukeshu.com/",
			Audience:            "friends",
			ClientID:            "foo",
		},
	}
	for k, _v := range filters {
		v := _v.(crd.FilterOAuth2)
		v.Validate("default", nil)
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

	grpcClientConn, err := grpc.Dial(strings.TrimPrefix(server.URL, "http://"), grpc.WithInsecure())
	if err != nil {
		tb.Fatal(err)
	}
	client := filterapi.NewFilterClient(grpcClientConn)

	return server, client
}

// picks a random "remote client port"; this doesn't have to really be
// available on this system.  Returns a random port in the range
// (1024, UINT32_MAX].
func pickAPort() uint32 {
	bigPort, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint32-1024))
	return uint32(bigPort.Int64()) + 1024
}

func newFilterRequest(method string, path string, header http.Header) *filterapi.FilterRequest {
	ret := &filterapi.FilterRequest{
		Source: &envoyAuthV2.AttributeContext_Peer{
			Address: &envoyCoreV2.Address{
				Address: &envoyCoreV2.Address_SocketAddress{
					SocketAddress: &envoyCoreV2.SocketAddress{
						Protocol: envoyCoreV2.TCP,
						Address:  "73.168.135.4", // A "user IP", c-73-168-135-4.hsd1.in.comcast.net, LukeShu's current IP
						PortSpecifier: &envoyCoreV2.SocketAddress_PortValue{
							PortValue: pickAPort(),
						},
					},
				},
			},
		},
		Destination: &envoyAuthV2.AttributeContext_Peer{
			Address: &envoyCoreV2.Address{
				Address: &envoyCoreV2.Address_SocketAddress{
					SocketAddress: &envoyCoreV2.SocketAddress{
						Protocol: envoyCoreV2.TCP,
						Address:  "45.76.26.79", // A "server IP", lukeshu.com
						PortSpecifier: &envoyCoreV2.SocketAddress_PortValue{
							PortValue: 80,
						},
					},
				},
			},
		},
		Request: &envoyAuthV2.AttributeContext_Request{
			Time: pbTypes.TimestampNow(),
			Http: &envoyAuthV2.AttributeContext_HttpRequest{
				Id:       uuid.NewV4().String(),
				Method:   method,
				Headers:  map[string]string{},
				Path:     path,
				Host:     "lukeshu.com",
				Scheme:   "http",
				Query:    "",
				Fragment: "",
				Size_:    -1,
				Protocol: "HTTP/1.1",
				Body:     nil,
			},
		},
	}
	if header != nil {
		for k, vs := range header {
			ret.Request.Http.Headers[k] = strings.Join(vs, ",")
		}
	}
	return ret
}
