package middleware

import (
	"context"
	"net/http"
	"net/url"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/util"
	"github.com/sirupsen/logrus"
)

// DomainCheck verifies that a given request has a correspondent application. Applications are
// registered via CRD, therefore this middleware should be called at the very top of the chain,
// but after config_check. If an app is found, its configuration will be written to the request
// context. See controller.App for more details.
type DomainCheck struct {
	Config *config.Config
	Logger *logrus.Entry
	Ctrl   *controller.Controller
}

func (c *DomainCheck) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	u, err := url.Parse(util.ToRawURL(r))
	if err != nil {
		c.Logger.Errorf("error parsing request url: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	a := c.Ctrl.FindTenant(u.Hostname())
	if a == nil {
		c.Logger.Debugf("not a registered domain: %s", u.Hostname())
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	next(w, r.WithContext(context.WithValue(r.Context(), controller.TenantCTXKey, a)))
}
