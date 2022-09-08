package ir

import "github.com/emissary-ingress/emissary/v3/internal/ir/types"

// MapToNamespacedName is a helper function for converting unstructured json
// into a strongly type k8s NamespacedName.
//
// It is infalliable and will always return a value even if both the Name
// and Namespace are set to zeroth value of string ("") so if additional validation
// must be done by the caller based on their needs
func MapToNamespacedName(input map[string]interface{}) types.NamespacedName {

	var name string
	var namespace string

	if val, ok := input["name"].(string); ok {
		name = val
	}

	if val, ok := input["namespace"].(string); ok {
		namespace = val
	}

	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

}
