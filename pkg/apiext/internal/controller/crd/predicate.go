package crd

import (
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getAmbassadorioPredicate will only match CRD's for the getambassado.io group and will ignore the others
func getAmbassadorioPredicate() func(client.Object) bool {
	return func(obj client.Object) bool {
		crDef, ok := obj.(*apiextv1.CustomResourceDefinition)
		if !ok || crDef == nil {
			return false
		}

		if crDef.Spec.Group == "getambassador.io" {
			return true
		}

		return false
	}
}
