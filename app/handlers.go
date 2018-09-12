package app

// TODO(gsagula): WIP Lots to be cleaned in this file.

import (
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwtm "github.com/auth0/go-jwt-middleware"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	// AuthzKEY header key.
	AuthzKEY = "Authorization"
	// ClientSECKey header key
	ClientSECKey = "Client-Secret"
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
	// DefaultScopes for 3rd party providers.
	DefaultScopes = "offline_access openid profile" // TODO(gsagula): get the scopes from the list
)

// Handler ..
type Handler struct {
	Config     *config.Config
	Logger     *logrus.Logger
	Ctrl       *Controller
	Jwt        *jwtm.JWTMiddleware
	PrivateKey string
}

// Authorize ..
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	public, scopes := h.policy(r.Method, r.Host, r.URL.Path)
	if !public {
		token, _ := r.Context().Value(h.Jwt.Options.UserProperty).(*jwt.Token)
		if token == nil {
			h.Logger.Info("authorization token not present")
			h.verify(w, r)
			return
		}

		if err := token.Claims.Valid(); err != nil {
			h.Logger.Info("invalid authorization token")
			h.verify(w, r)
			return
		}

		// TODO(gsagula): consider redirecting to consent uri and logging the error.
		for _, scope := range scopes {
			if !checkScope(scope, token.Raw) {
				h.Logger.Infof("invalid scope: %s", scope)
				h.verify(w, r)
				return
			}
		}
	}

	w.Header().Set(AuthzKEY, r.Header.Get(AuthzKEY))
	w.Header().Del(ClientSECKey)
	w.WriteHeader(http.StatusOK)
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
	// TODO(gsagula): add to an util lib.
	var buf bytes.Buffer
	buf.WriteString(r.URL.Path)
	if len(r.URL.RawQuery) > 0 {
		buf.WriteString("?")
		buf.WriteString(r.URL.RawQuery)
	}

	token := jwt.New(&jwt.SigningMethodHMAC{Name: "HS256", Hash: crypto.SHA256})
	token.Claims = jwt.MapClaims{
		"exp":  time.Now().Add(exp).Unix(),            // time when the token will expire (10 minutes from now)
		"jti":  uuid.Must(uuid.NewV4(), nil).String(), // a unique identifier for the token
		"iat":  time.Now().Unix(),                     // when the token was issued/created (now)
		"nbf":  0,                                     // time before which the token is not yet valid (2 minutes ago)
		"path": buf.String(),                          // the subject/principal is whom the token is about
	}

	key, err := token.SignedString([]byte(h.PrivateKey))
	if err != nil {
		h.Logger.Fatalf("failed to sign state")
	}

	h.Logger.Debugf("state key: %s", key)
	return key
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
	h.Logger.Infof("checking policy for request: %s  %s %s", method, host, path)
	for _, rule := range h.Ctrl.Rules.Load().([]Rule) {
		h.Logger.Debugf("checking %v against %v, %v", rule, host, path)
		if rule.match(host, path) {
			return rule.Public, strings.Fields(rule.Scopes)
		}
	}
	return false, []string{}
}
