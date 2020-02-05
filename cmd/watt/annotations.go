package main

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/watt"
)

func parseAnnotationResources(resource k8s.Resource) (annotationResources []k8s.Resource, annotationErrs []watt.Error) {
	annotationStr, annotationStrOK := resource.Metadata().Annotations()["getambassador.io/config"].(string)
	if !annotationStrOK {
		return
	}
	annotationDecoder := yaml.NewDecoder(strings.NewReader(annotationStr))
	// Parse in to a scratch _annotationResources list instead of the final annotationResources, so that we can more
	// easily prune invalid entries out before returning it.
	var _annotationResources []k8s.Resource
	for {
		var annotationResource k8s.Resource
		err := annotationDecoder.Decode(&annotationResource)
		if err == io.EOF {
			break
		}
		if err != nil {
			annotationErrs = append(annotationErrs, watt.NewError(
				fmt.Sprintf("%s/%s", resource.QKind(), resource.QName()),
				"could not read YAML in annotation getambassador.io/config"))
			return
		}
		_annotationResources = append(_annotationResources, annotationResource)
	}
	for _, annotationResource := range _annotationResources {
		// The Canonical API Version for our resources always starts with "getambassador.io/",
		// but it used to always start with "ambassador/". Translate as needed for backward
		// compatibility.
		if apiVersion := k8s.Map(annotationResource).GetString("apiVersion"); strings.HasPrefix(apiVersion, "ambassador/") {
			annotationResource["apiVersion"] = "getambassador.io/" + strings.TrimPrefix(apiVersion, "ambassador/")
		}

		// Make sure it's in the right API group
		if !strings.HasPrefix(k8s.Map(annotationResource).GetString("apiVersion"), "getambassador.io/") {
			annotationErrs = append(annotationErrs, watt.NewError(
				fmt.Sprintf("%s/%s: annotation getambassador.io/config: %s/%s",
					resource.QKind(), resource.QName(),
					annotationResource.QKind(), annotationResource.QName()),
				"not in the getambassador.io apiGroup, ignoring"))
			continue
		}

		// Un-fold annotations with collapsed metadata/spec
		if dat, ok := annotationResource["metadata"].(map[string]interface{}); !ok || dat == nil {
			annotationResource["metadata"] = k8s.Metadata{}
		}
		if dat, ok := annotationResource["spec"].(map[string]interface{}); !ok || dat == nil {
			annotationResource["spec"] = k8s.Map{}
		}
		for k, v := range annotationResource {
			switch k {
			case "apiVersion", "kind", "metadata", "spec", "status":
				// do nothing
			case "name", "namespace", "generation":
				annotationResource["metadata"].(map[string]interface{})[k] = v
				delete(annotationResource, k)
			case "metadata_labels":
				annotationResource["metadata"].(map[string]interface{})["labels"] = v
				delete(annotationResource, k)
			default:
				annotationResource["spec"].(map[string]interface{})[k] = v
				delete(annotationResource, k)
			}
		}

		// Default attributes from the parent
		if annotationResource.Namespace() == "" {
			annotationResource.Metadata()["namespace"] = resource.Namespace()
		}
		if annotationResource.Metadata()["labels"] == nil && resource.Metadata()["labels"] != nil {
			annotationResource.Metadata()["labels"] = resource.Metadata()["labels"]
		}

		// Add it to the snapshot
		annotationResources = append(annotationResources, annotationResource)
	}
	return
}
