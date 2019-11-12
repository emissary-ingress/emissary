package banner

import (
	"html/template"
	"net/http"

	"github.com/mediocregopher/radix.v2/pool"

	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
)

type banner struct {
	t         *template.Template
	limit     limiter.Limiter
	redisPool *pool.Pool
}

func NewBanner(limit limiter.Limiter, redisPool *pool.Pool) http.Handler {
	// TODO(alexgervais): Display a banner inviting "unregistered" license users to enter their email
	t := template.New("banner")
	t, _ = t.Parse(`
{{- if not .hasRedis -}}
<div style="color:red; font-weight: bold">Authentication and Rate Limiting are disabled as Ambassador Edge Stack is not configured to use Redis. Please follow the <a href="https://www.getambassador.io/user-guide/install">Ambassador Edge Stack installation guide</a> to complete your setup.</div>
{{- else if .features_over_limit -}}
<div style="color:red; font-weight: bold">You've reached the usage limits for your license. If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.</div>
{{- end -}}
`)
	return &banner{t: t, limit: limit, redisPool: redisPool}
}

func (b *banner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	licensedFeaturesOverLimit := []string{}
	for _, limitName := range licensekeys.ListKnownLimits() {
		limit, ok := licensekeys.ParseLimit(limitName)
		if ok {
			limitValue := b.limit.GetLimitValueAtPointInTime(&limit)
			usageValue := b.limit.GetFeatureUsageValueAtPointInTime(&limit)
			if usageValue >= limitValue {
				licensedFeaturesOverLimit = append(licensedFeaturesOverLimit, limitName)
			}
		}
	}
	data := map[string]interface{}{
		"features_over_limit": licensedFeaturesOverLimit,
		"unregistered":        b.limit.GetClaims().CustomerID == "unregistered",
		"hasRedis":            b.redisPool != nil,
	}
	b.t.Execute(w, data)
}
