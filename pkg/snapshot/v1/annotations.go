package snapshot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/datawire/ambassador/v2/pkg/kates"
)

//nolint:unused // This will be used by a later commit.
type wattError struct {
	Source kates.Object
	Err    error
}

//nolint:unused // This will be used by a later commit.
func (e *wattError) Error() string {
	return fmt.Sprintf("%s/%s.%s: %v",
		e.Source.GetObjectKind().GroupVersionKind().Kind,
		e.Source.GetName(), e.Source.GetNamespace(),
		e.Err)
}

//nolint:unused // This will be used by a later commit.
func parseAnnotationResources(resource kates.Object) (annotationResources []kates.Object, annotationErrs []*wattError) {
	annotationStr, annotationStrOK := resource.GetAnnotations()["getambassador.io/config"]
	if !annotationStrOK {
		return
	}
	// Parse in to a scratch _annotationResources list instead of the final annotationResources, so that we can more
	// easily prune invalid entries out before returning it.
	_annotationResources, err := kates.ParseManifestsToUnstructured(annotationStr)
	if err != nil {
		annotationErrs = append(annotationErrs, &wattError{
			Source: resource,
			Err:    fmt.Errorf("could not read YAML in annotation getambassador.io/config: %w", err),
		})
		return
	}
	for _, _annotationResource := range _annotationResources {
		annotationResource := _annotationResource.(*kates.Unstructured).Object
		// Un-fold annotations with collapsed metadata/spec
		if _, ok := annotationResource["apiVersion"].(string); !ok {
			annotationResource["apiVersion"] = ""
		}
		if dat, ok := annotationResource["metadata"].(map[string]interface{}); !ok || dat == nil {
			annotationResource["metadata"] = map[string]interface{}{}
		}
		if dat, ok := annotationResource["spec"].(map[string]interface{}); !ok || dat == nil {
			annotationResource["spec"] = map[string]interface{}{}
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
		if ns, ok := annotationResource["metadata"].(map[string]interface{})["namespace"].(string); !ok || ns == "" {
			annotationResource["metadata"].(map[string]interface{})["namespace"] = resource.GetNamespace()
		}
		if annotationResource["metadata"].(map[string]interface{})["labels"] == nil && resource.GetLabels() != nil {
			annotationResource["metadata"].(map[string]interface{})["labels"] = resource.GetLabels()
		}

		// The Canonical API Version for our resources always starts with "getambassador.io/",
		// but it used to always start with "ambassador/". Translate as needed for backward
		// compatibility.
		if apiVersion := annotationResource["apiVersion"].(string); strings.HasPrefix(apiVersion, "ambassador/") {
			annotationResource["apiVersion"] = "getambassador.io/" + strings.TrimPrefix(apiVersion, "ambassador/")
		}

		// Make sure it's in the right API group
		if !strings.HasPrefix(annotationResource["apiVersion"].(string), "getambassador.io/") {
			annotationErrs = append(annotationErrs, &wattError{
				Source: resource,
				Err: fmt.Errorf("annotation getambassador.io/config: %w", &wattError{
					Source: &kates.Unstructured{Object: annotationResource},
					Err:    errors.New("not in the getambassador.io apiGroup, ignoring"),
				}),
			})
			continue
		}

		// Add it to the snapshot
		annotationResources = append(annotationResources, &kates.Unstructured{Object: annotationResource})
	}
	return
}
