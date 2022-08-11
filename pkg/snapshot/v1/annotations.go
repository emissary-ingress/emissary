package snapshot

import (
	"context"
	"fmt"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/dlib/derror"
)

func (a *KubernetesSnapshot) PopulateAnnotations(ctx context.Context) error {
	var annotatable []kates.Object

	for _, s := range a.Services {
		annotatable = append(annotatable, s)
	}

	for _, i := range a.Ingresses {
		annotatable = append(annotatable, i)
	}

	var errs derror.MultiError
	a.Annotations, errs = getAnnotations(ctx, annotatable...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// getAnnotations extracts and converts any parseable annotations from the supplied resource. It
// omits any malformed annotations and does not report the errors. This is ok for now because the
// python code will catch and report any errors.
func getAnnotations(ctx context.Context, resources ...kates.Object) ([]kates.Object, derror.MultiError) {
	var result []kates.Object
	var errs derror.MultiError
	for _, r := range resources {
		ann, ok := r.GetAnnotations()["getambassador.io/config"]
		if ok {
			objs, err := kates.ParseManifestsToUnstructured(ann)
			if err != nil {
				errs = append(errs, err)
			} else {
				for _, untypedObj := range objs {
					typedObj, err := ConvertAnnotation(ctx, r, untypedObj.(*kates.Unstructured))
					if err != nil {
						errs = append(errs, err)
						result = append(result, untypedObj)
					} else {
						result = append(result, typedObj)
					}
				}
			}
		}
	}

	return result, errs
}

// Annotations require some post processing because they are weird. The older ones use "ambassador"
// as the Group rather than "getambassador.io", and also don't put their fields underneath
// "spec". These end up parsing as unstructured resources, this function converts them to the
// appropriate structured resource.
//
// NOTE: Right now this is only guaranteed to preserve enough fidelity to find secrets, this may
// work well enough for other purposes, but some careful review is required before such use.
func ConvertAnnotation(ctx context.Context, parent kates.Object, un *kates.Unstructured) (kates.Object, error) {
	// XXX: steal luke's code
	var tm kates.TypeMeta
	err := convert(un, &tm)
	if err != nil {
		return nil, err
	}

	gvk := tm.GroupVersionKind()

	// The Canonical Group for our resources is "getambassador.io", but it used to be
	// "ambassador". Translate as needed for backward compatibility.
	if gvk.Group == "ambassador" {
		gvk.Group = "getambassador.io"
	}

	if gvk.Group != "getambassador.io" {
		return nil, fmt.Errorf("annotation has unsupported GroupVersionKind %q, ignoring", gvk)
	}

	// This version munging is only kosher because right now we only care about preserving enough
	// fidelty to find secrets. (The v2 schema is a superset of the v1 schema for reasons, but the
	// semantics may not be the same non-secret fields, and who knows about v0.)
	if gvk.Version == "v0" || gvk.Version == "v1" {
		gvk.Version = "v2"
	}

	// Try to create a new typed object with the massaged group/version, if it doesn't work, bail
	// and return the original resource.
	apiVersion := gvk.GroupVersion().String()
	result, err := kates.NewObject(gvk.Kind, apiVersion)
	if err != nil {
		return nil, err
	}

	// create our converted object with the massaged apiVersion
	obj := make(map[string]interface{})
	obj["kind"] = gvk.Kind
	obj["apiVersion"] = apiVersion

	// create our converted metadata
	metadata := make(map[string]interface{})
	obj["metadata"] = metadata

	// default the namespace and labels based on the parent resource
	metadata["namespace"] = parent.GetNamespace()
	metadata["labels"] = parent.GetLabels()

	// copy everything into our converted metadata
	if orig, ok := un.Object["metadata"]; ok {
		for k, v := range orig.(map[string]interface{}) {
			metadata[k] = v
		}
	}

	// create our converted spec
	spec := make(map[string]interface{})
	obj["spec"] = spec

	// copy everything into our converted spec
	if orig, ok := un.Object["spec"]; ok {
		for k, v := range orig.(map[string]interface{}) {
			spec[k] = v
		}
	}

	// copy top level entries into the right places underneath metadata and spec
	for k, v := range un.Object {
		switch k {
		case "apiVersion", "kind", "metadata", "spec", "status":
			// do nothing
		case "name", "namespace", "generation":
			metadata[k] = v
		case "metadata_labels":
			metadata["labels"] = v
		default:
			spec[k] = v
		}
	}

	// now convert our unstructured annotation into the correct golang struct
	err = convert(obj, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
