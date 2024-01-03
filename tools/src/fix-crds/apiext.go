package main

import (
	_ "embed"
	"io"
	"text/template"

	apiextdefaults "github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
)

//go:embed apiext-deployment.yaml
var apiextDeploymentTmplStr string

//go:embed apiext-rbac.yaml
var apiextRBACTmplStr string

//go:embed apiext-namespace.yaml
var apiextNSTmplStr string

const (
	apiextSvcName = "emissary-apiext"
)

var globalLabels = map[string]string{
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	"app.kubernetes.io/name":       "emissary-apiext",
	"app.kubernetes.io/instance":   "emissary-apiext",
	"app.kubernetes.io/part-of":    "emissary-apiext",
	"app.kubernetes.io/managed-by": "kubectl_apply_-f_emissary-apiext.yaml",
}

func buildTemplateData(args Args, crdNames []string) map[string]interface{} {
	labelSelectors := map[string]string{
		"app.kubernetes.io/name":     "emissary-apiext",
		"app.kubernetes.io/instance": "emissary-apiext",
		"app.kubernetes.io/part-of":  "emissary-apiext",
	}

	var image string
	switch args.Target {
	case TargetAPIServerKAT:
		image = "{images[emissary]}"
	case TargetAPIServerKubectl:
		image = "$imageRepo$:$version$"
	case TargetAPIExtDeployment:
		image = args.Image
	}

	return map[string]interface{}{
		// Constants that are easier to edit here than in YAML.
		"Labels":         globalLabels,
		"LabelSelectors": labelSelectors,
		"Names": map[string]string{
			// These are *just* the names that appear in multiple places, and therefore
			// need to be consistent.  Names that are on-offs can just go straight in
			// the YAML.
			"Namespace":      apiextdefaults.WebhookCASecretNamespace,
			"Service":        apiextSvcName,
			"ServiceAccount": "emissary-apiext",
			"ClusterRole":    "emissary-apiext",
			"Role":           "emissary-apiext",
		},
		// CLI Args
		"Target": args.Target,
		"Image":  image,
		// Input files
		"CRDNames": crdNames,
	}
}

func writeAPIExtNamespace(output io.Writer, data map[string]interface{}) error {
	tmpl := template.Must(template.New("apiext-namespace.yaml").
		Option("missingkey=error").
		Parse(apiextNSTmplStr))

	return tmpl.Execute(output, data)
}

func writeAPIExtRBAC(output io.Writer, data map[string]interface{}) error {
	apiextRBACTmpl := template.Must(template.New("apiext-rbac.yaml").
		Option("missingkey=error").
		Parse(apiextRBACTmplStr))

	return apiextRBACTmpl.Execute(output, data)
}

func writeAPIExtDeployment(output io.Writer, data map[string]interface{}) error {
	apiextDeploymentTmpl := template.Must(template.New("apiext-deployment.yaml").
		Option("missingkey=error").
		Parse(apiextDeploymentTmplStr))

	return apiextDeploymentTmpl.Execute(output, data)
}
