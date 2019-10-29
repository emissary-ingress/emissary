package banner

import (
	"github.com/datawire/apro/lib/licensekeys"
	"html/template"
	"net/http"
	"strings"
)

type banner struct {
	t *template.Template
}

func NewBanner() http.Handler {
	t := template.New("banner")
	t = t.Funcs(template.FuncMap{"StringsJoin": strings.Join})
	t, _ = t.Parse(`
{{- if .features_over_limit -}}
<div style="color:red; font-weight: bold">Contact Datawire for a license key to remove limits on features: {{ StringsJoin .features_over_limit ", " }}</div>
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
