package main

import (
	_ "embed"
	"fmt"
	"io"
	"sort"
	"text/template"
)

//go:embed apiext.yaml
var apiextTmplStr string

const (
	namespace     = "emissary-system"
	apiextSvcName = "emissary-apiext"
)

var globalLabels = map[string]string{
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	"app.kubernetes.io/name":       "emissary-apiext",
	"app.kubernetes.io/instance":   "emissary-apiext",
	"app.kubernetes.io/part-of":    "emissary-apiext",
	"app.kubernetes.io/managed-by": "kubectl_apply_-f_emissary-apiext.yaml",
}

func writeAPIExt(output io.Writer, args Args, crdNames []string) error {
	if args.Target == TargetInternalValidator {
		return nil
	}

	sort.Strings(crdNames)
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
	default:
		return fmt.Errorf("unsure which image to use for args.Target=%q", args.Target)
	}

	data := map[string]interface{}{
		// Constants that are easier to edit here than in YAML.
		"Labels":         globalLabels,
		"LabelSelectors": labelSelectors,
		"Names": map[string]string{
			// These are *just* the names that appear in multiple places, and therefore
			// need to be consistent.  Names that are on-offs can just go straight in
			// the YAML.
			"Namespace":      namespace,
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

	apiextTmpl := template.Must(template.New("apiext.yaml").
		Option("missingkey=error").
		Parse(apiextTmplStr))

	return apiextTmpl.Execute(output, data)
}
