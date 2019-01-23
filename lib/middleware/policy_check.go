package middleware

import (
	"context"
	"net/http"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/sirupsen/logrus"
)

// PolicyCheck does an initial check on Path and Host matches. If policy matches to
// a public resource this midleware will return immediately, otherwise a reference to Rules
// will be passed to the request context for further checking.
type PolicyCheck struct {
	Logger      *logrus.Entry
	Ctrl        *controller.Controller
	DefaultRule *controller.Rule
}

func (p *PolicyCheck) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	ok, rule := p.policy(r.Host, r.URL.Path)
	if ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	next(w, r.WithContext(context.WithValue(r.Context(), controller.RuleCTXKey, rule)))
}

// The first return result specifies whether authentication is
// required, the second return result specifies which scopes are
// required for access.
func (p *PolicyCheck) policy(host, path string) (bool, *controller.Rule) {
	rules := p.Ctrl.Rules.Load()

	if rules != nil {
		for _, rule := range rules.([]controller.Rule) {
			// if any rule matches, continue..
			if !rule.MatchHTTPHeaders(host, path) {
				continue
			}

			p.Logger.Debugf("host=%s, path=%s, public=%v", rule.Host, rule.Path, rule.Public)

			// if rule matches and it's public, return 200 OK.
			if rule.Public {
				p.Logger.Debugf("%s %s is public", host, path)
				return true, nil
			}

			// if rule matches, but not public move through the chain and pass the rule
			// in the context.
			return false, &rule
		}
	}

	p.Logger.Debug("no matched rule")
	return false, p.DefaultRule
}
