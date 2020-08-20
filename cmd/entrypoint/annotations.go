package entrypoint

import (
	"github.com/datawire/ambassador/pkg/kates"
)

// Annotations require some post processing because they are weird. The older ones use "ambassador"
// as the Group rather than "getambassador.io", and also don't put their fields underneath
// "spec". These end up parsing as unstructured resources, this function converts them to the
// appropriate structured resource.
func convertAnnotation(namespace string, kobj kates.Object) kates.Object {
	un, ok := kobj.(*kates.Unstructured)
	if !ok {
		return kobj
	}

	var tm kates.TypeMeta
	err := convert(un, &tm)
	if err != nil {
		return un
	}

	gvk := tm.GroupVersionKind()

	if gvk.Group == "ambassador" {
		gvk.Group = "getambassador.io"
	}

	if gvk.Group != "getambassador.io" {
		return un
	}

	if gvk.Version == "v0" || gvk.Version == "v1" {
		gvk.Version = "v2"
	}

	apiVersion := gvk.GroupVersion().String()
	result, err := kates.NewObject(gvk.Kind, apiVersion)
	if err != nil {
		return un
	}

	name, ok := un.Object["name"]
	if !ok {
		return un
	}

	nsi, ok := un.Object["namespace"]
	if ok {
		namespace = nsi.(string)
	}

	obj := make(map[string]interface{})

	obj["kind"] = gvk.Kind
	obj["apiVersion"] = apiVersion

	metadata := make(map[string]interface{})
	obj["metadata"] = metadata
	metadata["name"] = name
	if namespace != "" {
		metadata["namespace"] = namespace
	}

	spec := make(map[string]interface{})
	obj["spec"] = spec

	for k, v := range un.Object {
		switch k {
		case "kind":
		case "apiVersion":
		case "name":
		default:
			spec[k] = v
		}
	}

	err = convert(obj, result)
	if err != nil {
		return un
	}

	return result
}
