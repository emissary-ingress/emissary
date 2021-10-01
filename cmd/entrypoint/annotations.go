package entrypoint

import (
	"context"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kates/k8sresourcetypes"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

func parseAnnotations(ctx context.Context, a *snapshotTypes.KubernetesSnapshot) {
	var annotatable []kates.Object

	for _, s := range a.Services {
		annotatable = append(annotatable, s)
	}

	for _, i := range a.Ingresses {
		annotatable = append(annotatable, i)
	}

	a.Annotations = GetAnnotations(ctx, annotatable...)
}

// GetAnnotations extracts and converts any parseable annotations from the supplied resource. It
// omits any malformed annotations and does not report the errors. This is ok for now because the
// python code will catch and report any errors.
func GetAnnotations(ctx context.Context, resources ...kates.Object) (result []kates.Object) {
	for _, r := range resources {
		ann, ok := r.GetAnnotations()["getambassador.io/config"]
		if ok {
			objs, err := kates.ParseManifestsToUnstructured(ann)
			if err != nil {
				dlog.Errorf(ctx, "error parsing annotations: %v", err)
			} else {
				for _, o := range objs {
					result = append(result, convertAnnotation(ctx, r, o))
				}
			}
		}
	}

	return result
}

// Annotations require some post processing because they are weird. The older ones use "ambassador"
// as the Group rather than "getambassador.io", and also don't put their fields underneath
// "spec". These end up parsing as unstructured resources, this function converts them to the
// appropriate structured resource.
//
// NOTE: Right now this is only guaranteed to preserve enough fidelity to find secrets, this may
// work well enough for other purposes, but some careful review is required before such use.
func convertAnnotation(ctx context.Context, parent kates.Object, kobj kates.Object) kates.Object {
	un, ok := kobj.(*k8sresourcetypes.Unstructured)
	if !ok {
		return kobj
	}

	// XXX: steal luke's code
	var tm kates.TypeMeta
	err := convert(un, &tm)
	if err != nil {
		dlog.Debugf(ctx, "Error parsing type meta for annotation")
		return un
	}

	gvk := tm.GroupVersionKind()

	// The Canonical Group for our resources is "getambassador.io", but it used to be
	// "ambassador". Translate as needed for backward compatibility.
	if gvk.Group == "ambassador" {
		gvk.Group = "getambassador.io"
	}

	if gvk.Group != "getambassador.io" {
		dlog.Debugf(ctx, "Annotation does not have group getambassador.io")
		return un
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
		return un
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
		return un
	}

	return result
}
