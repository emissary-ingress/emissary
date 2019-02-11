package tpl

import (
	"text/template"

	shared "github.com/lyft/protoc-gen-validate/templates/goshared"
)

func Register(tpl *template.Template) {
	shared.Register(tpl)
	template.Must(tpl.Parse(fileTpl))
	template.Must(tpl.New("required").Parse(requiredTpl))
	template.Must(tpl.New("timestamp").Parse(timestampTpl))
	template.Must(tpl.New("duration").Parse(durationTpl))
}
