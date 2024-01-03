package crd

import (
	"context"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// registerGetAmbassadorioGroupIndexer indexes the group so we can get only the getambassador.io CRDs
func registerGetAmbassadorioGroupIndexer(mgr manager.Manager) error {
	return mgr.GetFieldIndexer().IndexField(context.Background(), &apiextv1.CustomResourceDefinition{},
		crdGroupIndexField,
		func(obj client.Object) []string {
			crDef := obj.(*apiextv1.CustomResourceDefinition)
			if crDef.Spec.Group == getambassadorioCRDGroup {
				return []string{obj.GetName()}
			}

			return nil
		},
	)
}
