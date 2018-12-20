package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	// RedirectURLFmt is a template string for redirect url.
	RedirectURLFmt = "%s://%s/authorize?audience=%s&response_type=code&redirect_uri=%s&client_id=%s&state=%s&scope=%s"
)

// Authorize is the last handler in the chain of the authorization server.
type Authorize struct {
	Config *config.Config
	Logger *logrus.Entry
	Ctrl   *controller.Controller
	Secret *secret.Secret
}

// Check is a handler function that inspects the request by looking for the presence of
// a token and for any insvalid scope. If these validations pass, an authorization
// header is set in a 200 OK response.
func (h *Authorize) Check(w http.ResponseWriter, r *http.Request) {
	tenant := controller.GetTenantFromContext(r.Context())
	if tenant == nil {
		h.Logger.Errorf("authorization handler: app request context cannot be nil")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	u, err := url.Parse(util.ToRawURL(r))
	if err != nil {
		h.Logger.Errorf("error parsing request url: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	redirect := fmt.Sprintf(
		RedirectURLFmt,
		u.Scheme,
		h.Config.Domain,
		tenant.Audience,
		tenant.CallbackURL,
		tenant.ClientID,
		h.signState(r),
		tenant.Scopes,
	)

	h.Logger.Debug("redirecting to the authorization endpoint")
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h *Authorize) signState(r *http.Request) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Config.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),    // a unique identifier for the token
		"iat":          time.Now().Unix(),                        // when the token was issued/created (now)
		"nbf":          0,                                        // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": util.ToRawURL(r),                         // original request url
	}

	k, err := t.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Errorf("failed to sign state: %v", err)
	}

	return k
}
