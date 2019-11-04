package banner

import (
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/mediocregopher/radix.v2/pool"
	"html/template"
	"net/http"
)

type banner struct {
	t             *template.Template
	licenseClaims **licensekeys.LicenseClaimsLatest
	redisPool     *pool.Pool
}

func NewBanner(licenseClaims **licensekeys.LicenseClaimsLatest, redisPool *pool.Pool) http.Handler {
	// TODO(alexgervais): Display a banner inviting "unregistered" license users to enter their email
	t := template.New("banner")
	t, _ = t.Parse(`
{{- if not .hasRedis -}}
<div style="color:red; font-weight: bold">Authentication and Rate Limiting are disabled as Ambassador Edge Stack is not configured to use Redis. Please follow the <a href="https://www.getambassador.io/user-guide/install">Ambassador Edge Stack installation guide</a> to complete your setup.</div>
{{- else if .features_over_limit -}}
<div style="color:red; font-weight: bold">You've reached the usage limits for your license. If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.</div>
{{- end -}}
`)
	return &banner{t: t, licenseClaims: licenseClaims, redisPool: redisPool}
}

func (b *banner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	licensedFeaturesOverLimit := []string{}
	for _, feature := range licensekeys.ListKnownFeatures() {
		// TODO(alexgervais): Filter on actual `usage` >= `limit`
		licensedFeaturesOverLimit = append(licensedFeaturesOverLimit, feature)
	}
	data := map[string]interface{}{
		"features_over_limit": licensedFeaturesOverLimit,
		"unregistered":        (*b.licenseClaims).CustomerID == "unregistered",
		"hasRedis":            b.redisPool != nil,
	}
	b.t.Execute(w, data)
}
