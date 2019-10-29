package banner

import (
	"github.com/datawire/apro/lib/licensekeys"
	"html/template"
	"net/http"
)

type banner struct {
	t *template.Template
}

func NewBanner() http.Handler {
	t := template.New("banner")
	t, _ = t.Parse(`
{{- if .features_over_limit -}}
<div style="color:red; font-weight: bold">You've reached the usage limits for your license. If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.</div>
{{- end -}}
`)
	return &banner{t: t}
}

func (b *banner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	licensedFeaturesOverLimit := []string{}
	for _, feature := range licensekeys.ListKnownFeatures() {
		// TODO: Filter on actual `usage` > `limit`
		licensedFeaturesOverLimit = append(licensedFeaturesOverLimit, feature)
	}
	data := map[string]interface{}{
		"features_over_limit": licensedFeaturesOverLimit,
	}
	b.t.Execute(w, data)
}
