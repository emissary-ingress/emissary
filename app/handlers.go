package app

// TODO(gsagula): WIP Lots to be cleaned in this file.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	// RedirectURLFmt is a template string for redirect url.
	RedirectURLFmt = "https://%s/authorize?audience=%s&response_type=code&redirect_uri=%s&client_id=%s&state=%s&scope=%s"
	// DefaultScopes for 3rd party providers.
	DefaultScopes = "offline_access openid profile"
)

// CustomClaims TODO(gsagula): comment
type CustomClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
}

// Response TODO(gsagula): comment
type Response struct {
	Message string `json:"message"`
}

// Handler TODO(gsagula): comment
type Handler struct {
	Config *config.Config
	Logger *logrus.Logger
	Ctrl   *Controller
	Secret *secret.Secret
}

// Authorize TODO(gsagula): comment
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	public, scopes := h.policy(r.Method, r.Host, r.URL.Path)
	if !public {
		token, _ := r.Context().Value("user").(*jwt.Token)

		if token == nil {
			h.Logger.Debug("authorization token not present")
			h.checkRequest(w, r)
			return
		}

		for _, scope := range scopes {
			if !checkScope(scope, token.Raw) {
				h.Logger.Debugf("invalid scope: %s", scope)
				h.checkRequest(w, r)
				return
			}
		}
	}

	w.Header().Set("Authorization", r.Header.Get("Authorization"))
	w.Header().Del("Client-Secret")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) checkRequest(w http.ResponseWriter, r *http.Request) {
	if h.Config.DenyOnFailure {
		jsonResponse, err := json.Marshal(&Response{"unauthorized"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "Application/Json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(jsonResponse)
	} else {
		// TODO(gsagula): make duration configurable.
		authURL := fmt.Sprintf(
			RedirectURLFmt,
			h.Config.Domain,
			h.Config.Audience,
			h.Config.CallbackURL,
			h.Config.ClientID,
			h.signState(r, h.Config.StateTTL),
			DefaultScopes,
		)

		h.Logger.Debugf("redirecting to authorize endpoint: %s", authURL)
		http.Redirect(w, r, authURL, http.StatusSeeOther)
	}
}

func (h *Handler) signState(r *http.Request, exp time.Duration) string {
	var buf bytes.Buffer
	buf.WriteString(r.URL.Path)
	if len(r.URL.RawQuery) > 0 {
		buf.WriteString("?")
		buf.WriteString(r.URL.RawQuery)
	}

	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{
		"exp":  time.Now().Add(exp).Unix(),            // time when the token will expire (10 minutes from now)
		"jti":  uuid.Must(uuid.NewV4(), nil).String(), // a unique identifier for the token
		"iat":  time.Now().Unix(),                     // when the token was issued/created (now)
		"nbf":  0,                                     // time before which the token is not yet valid (2 minutes ago)
		"path": buf.String(),                          // the subject/principal is whom the token is about
	}

	key, err := token.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Fatalf("failed to sign state: %v", err)
	}

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
	h.Logger.Debugf("checking policy for request: %s  %s %s", method, host, path)
	for _, rule := range h.Ctrl.Rules.Load().([]Rule) {
		h.Logger.Debugf("checking %v against %v, %v", rule, host, path)
		if rule.match(host, path) {
			return rule.Public, strings.Fields(rule.Scopes)
		}
	}
	return false, []string{}
}
