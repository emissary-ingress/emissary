package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/util"
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

// Authorize is the last handler in the chain of the authorization server.
type Authorize struct {
	Config *config.Config
	Logger *logrus.Logger
	Ctrl   *controller.Controller
	Secret *secret.Secret
}

// Check is a handler function that inspects the request by looking for the presence of
// a token and for any insvalid scope. If these validations pass, an authorization
// header is set in a 200 OK response.
func (h *Authorize) Check(w http.ResponseWriter, r *http.Request) {
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

func (h *Authorize) checkRequest(w http.ResponseWriter, r *http.Request) {
	if h.Config.DenyOnFailure {
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	u := fmt.Sprintf(
		RedirectURLFmt,
		h.Config.Domain,
		h.Config.Audience,
		h.Config.CallbackURL,
		h.Config.ClientID,
		h.signState(r, h.Config.StateTTL),
		DefaultScopes,
	)

	h.Logger.Debugf("redirecting to authorize endpoint: %s", u)
	http.Redirect(w, r, u, http.StatusSeeOther)
	return
}

func (h *Authorize) signState(r *http.Request, exp time.Duration) string {
	var buf bytes.Buffer
	if h.Config.Secure {
		buf.WriteString("https://")
	} else {
		buf.WriteString("http://")
	}
	buf.WriteString(r.Host)
	buf.WriteString(r.RequestURI)

	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(exp).Unix(),            // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(), // a unique identifier for the token
		"iat":          time.Now().Unix(),                     // when the token was issued/created (now)
		"nbf":          0,                                     // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": buf.String(),                          // original request url
	}

	key, err := token.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Fatalf("failed to sign state: %v", err)
	}

	return key
}

// CustomClaims TODO(gsagula): comment
type CustomClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
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
func (h *Authorize) policy(method, host, path string) (bool, []string) {
	h.Logger.Debugf("checking policy for request: %s  %s %s", method, host, path)
	for _, rule := range h.Ctrl.Rules.Load().([]controller.Rule) {
		h.Logger.Debugf("checking %v against %v, %v", rule, host, path)
		if rule.Match(host, path) {
			return rule.Public, strings.Fields(rule.Scopes)
		}
	}
	return false, []string{}
}
