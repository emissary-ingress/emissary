package snapshot

import (
	"context"
	"fmt"
	"strings"

	crdAll "github.com/datawire/ambassador/v2/pkg/api/getambassador.io"
	crdCurrent "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/derror"
)

func annotationKey(obj kates.Object) string {
	return fmt.Sprintf("%s/%s.%s",
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace())
}

var (
	scheme    = crdAll.BuildScheme()
	validator = crdAll.NewValidator()
)

func (s *KubernetesSnapshot) PopulateAnnotations(ctx context.Context) error {
	var annotatable []kates.Object
	for _, svc := range s.Services {
		annotatable = append(annotatable, svc)
	}
	for _, ing := range s.Ingresses {
		annotatable = append(annotatable, ing)
	}

	s.Annotations = make(map[string]AnnotationList)
	var errs derror.MultiError
	for _, r := range annotatable {
		key := annotationKey(r)
		objs, err := ParseAnnotationResources(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			continue
		}
		annotations := make(AnnotationList, len(objs))
		for i, untypedObj := range objs {
			typedObj, err := ValidateAndConvertObject(ctx, untypedObj)
			if err != nil {
				untypedObj.Object["errors"] = err.Error()
				annotations[i] = untypedObj
			} else {
				annotations[i] = typedObj
			}
		}
		if len(annotations) > 0 {
			s.Annotations[key] = annotations
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateAndConvertObject validates an apiGroup=getambassador.io resource, and converts it to the
// preferred version.
//
// This is meant for use on objects that come from annotations.  You should probably not be calling
// this directly; the only reason it's public is for use by tests.
func ValidateAndConvertObject(
	ctx context.Context,
	in *kates.Unstructured,
) (out kates.Object, err error) {
	// Validate it
	gvk := in.GetObjectKind().GroupVersionKind()
	if !scheme.Recognizes(gvk) {
		return nil, fmt.Errorf("unsupported GroupVersionKind %q, ignoring", gvk)
	}
	if err := validator.Validate(ctx, in); err != nil {
		return nil, err
	}

	// Convert it to the correct type+version.
	out, err = convertAnnotationObject(in)
	if err != nil {
		return nil, err
	}

	// Validate it again (after conversion) just to be safe
	if err := validator.Validate(ctx, out); err != nil {
		return nil, err
	}

	return out, nil
}

// convertAnnotationObject converts a valid kates.Object to the correct type+version.
func convertAnnotationObject(srcUnstruct *kates.Unstructured) (kates.Object, error) {
	// Convert from an 'Unstructured' to the appropriate Go type, without actually converting
	// versions.
	srcGVK := srcUnstruct.GetObjectKind().GroupVersionKind()
	_src, err := scheme.ConvertToVersion(srcUnstruct, srcGVK.GroupVersion())
	if err != nil {
		return nil, fmt.Errorf("1: %w", err)
	}
	src, ok := _src.(kates.Object)
	if !ok {
		return nil, fmt.Errorf("type %T doesn't implement kates.Object", _src)
	}

	// Create the Go type of the output version.
	dstGVK := crdCurrent.GroupVersion.WithKind(srcGVK.Kind)
	if dstGVK == srcGVK {
		// Optimize!  Plus, kates.ConvertObject doesn't like doing no-op conversions.
		return src, nil
	}
	_dst, err := scheme.New(dstGVK)
	if err != nil {
		return nil, fmt.Errorf("2: %w", err)
	}
	dst, ok := _dst.(kates.Object)
	if !ok {
		return nil, fmt.Errorf("type %T doesn't implement kates.Object", _dst)
	}
	dst.GetObjectKind().SetGroupVersionKind(dstGVK)

	// Convert versions.  This is based around Go types, which is why we need to convert from
	// 'Unstructured' first.
	if err := kates.ConvertObject(scheme, src, dst); err != nil {
		return nil, fmt.Errorf("3: %w", err)
	}
	return dst, nil
}

// ParseAnnotationResources parses the annotations on an object, and munges them to be
// Kubernetes-structured objects.  It does not do any validation or version conversion.
//
// You should probably not be calling this directly; the only reason it's public is for use by
// tests.
func ParseAnnotationResources(resource kates.Object) ([]*kates.Unstructured, error) {
	annotationStr, annotationStrOK := resource.GetAnnotations()["getambassador.io/config"]
	if !annotationStrOK {
		return nil, nil
	}
	// Parse in to a scratch _annotationResources list instead of the final annotationResources, so that we can more
	// easily prune invalid entries out before returning it.
	_annotationResources, err := kates.ParseManifestsToUnstructured(annotationStr)
	if err != nil {
		return nil, fmt.Errorf("annotation getambassador.io/config: could not parse YAML: %w", err)
	}
	annotationResources := make([]*kates.Unstructured, 0, len(_annotationResources))
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

		// Add it to the snapshot
		annotationResources = append(annotationResources, &kates.Unstructured{Object: annotationResource})
	}
	return annotationResources, nil
}
