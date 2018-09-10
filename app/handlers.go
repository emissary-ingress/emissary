package app

// TODO(gsagula): WIP Lots to be cleaned in this file.

import (
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	jwtm "github.com/auth0/go-jwt-middleware"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

const (
	// AuthzKEY header key.
	AuthzKEY = "Authorization"
	// ClientSECKey header key
	ClientSECKey = "Client-Secret"
	// StateSignature secret is used to sign the authorization state value.
	StateSignature = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3"
	// RedirectURLFmt is a template string for redirect url.
	RedirectURLFmt = "https://%s/authorize?audience=%s&response_type=code&redirect_uri=%s&client_id=%s&state=%s&scope=%s"
	// ContentTYPE HTTP header key
	ContentTYPE = "Content-Type"
	// ApplicationJSON HTTP header value
	ApplicationJSON = "Application/Json"
	// EmptyString ...
	EmptyString = ""
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
)

// Map is used to to track state and the initial request url, so
// it the call can be redirected after acquiring the access token.
var stateURLKv map[string]string

// Handler ..
type Handler struct {
	Config *config.Config
	Logger *logrus.Logger
	Ctrl   *Controller
	Jwt    *jwtm.JWTMiddleware
	// TODO(gsagula): Make this atomic and remove expired keys.
	StateKV *map[string]string
}

// Authorize ..
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	public, scopes := h.policy(r.Method, r.Host, r.URL.Path)
	if !public {
		token, _ := r.Context().Value(h.Jwt.Options.UserProperty).(*jwt.Token)
		if token == nil {
			h.Logger.Info("Token nil")
			h.verify(w, r)
			return
		}

		if err := token.Claims.Valid(); err != nil {
			h.Logger.Info("Claim invalid")
			h.verify(w, r)
			return
		}

		// TODO(gsagula): consider redirecting to consent uri and logging the error.
		for _, scope := range scopes {
			if !checkScope(scope, token.Raw) {
				h.Logger.Info("Scope invalid")
				h.verify(w, r)
				return
			}
		}
	}

	h.Logger.Info("Success")
	w.Header().Set(AuthzKEY, r.Header.Get(AuthzKEY))
	w.Header().Del(ClientSECKey)
	w.WriteHeader(http.StatusOK)
}

// CustomClaims TODO(gsagula): comment
type CustomClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
}

// Response TODO(gsagula): comment
type Response struct {
	Message string `json:"message"`
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if h.Config.DenyOnFailure {
		jsonResponse, err := json.Marshal(&Response{"unauthorized"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set(ContentTYPE, ApplicationJSON)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(jsonResponse)
	} else {
		var buf bytes.Buffer
		buf.WriteString(r.URL.Path)
		if len(r.URL.RawQuery) > 0 {
			buf.WriteString("?")
			buf.WriteString(r.URL.RawQuery)
		}

		stateKEY := h.signState(buf.String())
		//TODO(gsagula): avoid copying..
		storage := *h.StateKV
		storage[stateKEY] = buf.String()

		redirectURL := fmt.Sprintf(
			RedirectURLFmt,
			h.Config.Domain,
			h.Config.Audience,
			h.Config.CallbackURL,
			h.Config.ClientID,
			stateKEY,
			"offline_access openid profile", // TODO(gsagula): get the scopes from the list
		)

		h.Logger.Info("authorizing with the identity provider.")
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func (h *Handler) signState(url string) string {
	token := jwt.New(&jwt.SigningMethodHMAC{Name: "HS256", Hash: crypto.SHA256})
	key, err := token.SignedString([]byte(StateSignature))
	if err != nil {
		log.Fatal(err)
	}
	//TODO(gsagula): avoid copying..
	storage := *h.StateKV
	storage[key] = url
	return key
}

func checkScope(scope string, tokenString string) bool {
	token, _ := jwt.ParseWithClaims(tokenString, &CustomClaims{}, nil)
	claims, _ := token.Claims.(*CustomClaims)
	hasScope := false
	result := strings.Split(claims.Scope, " ")
	for i := range result {
		if result[i] == scope {
			hasScope = true
		}
	}

	return hasScope
}

// The first return result specifies whether authentication is
// required, the second return result specifies which scopes are
// required for access.
func (h *Handler) policy(method, host, path string) (bool, []string) {
	for _, rule := range h.Ctrl.Rules.Load().([]Rule) {
		h.Logger.Infof("checking %v against %v, %v", rule, host, path)
		if rule.match(host, path) {
			return rule.Public, strings.Fields(rule.Scopes)
		}
	}
	return false, []string{}
}
