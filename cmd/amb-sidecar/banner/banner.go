package banner

import (
	"html/template"
	"net/http"

	"github.com/mediocregopher/radix.v2/pool"

	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
)

type banner struct {
	t         *template.Template
	limit     limiter.Limiter
	redisPool *pool.Pool
}

func NewBanner(limit limiter.Limiter, redisPool *pool.Pool) http.Handler {
	t := template.New("banner")
	t, _ = t.Parse(`
{{- if not .hasRedis -}}
<div style="color:red; font-weight: bold">Authentication and Rate Limiting are disabled as Ambassador Edge Stack is not configured to use Redis. Please follow the <a href="https://www.getambassador.io/user-guide/install">Ambassador Edge Stack installation guide</a> to complete your setup.</div>
{{- else if .features_over_limit -}}
<div style="color:red; font-weight: bold">You've reached the <a href="https://www.getambassador.io/editions/">usage limits</a> for your license. If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.</div>
{{- end -}}
`)
	return &banner{t: t, limit: limit, redisPool: redisPool}
}

func (b *banner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"features_over_limit": b.limit.GetFeaturesOverLimitAtPointInTime(),
		"unregistered":        b.limit.GetClaims().CustomerID == "unregistered",
		"hasRedis":            b.redisPool != nil,
	}
	b.t.Execute(w, data)
}
