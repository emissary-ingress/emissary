package e2e

import (
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

func CustomResourceDefinitionConditions() func(object k8s.Object) bool {
	return func(obj k8s.Object) bool {
		status := obj.(*apiextv1.CustomResourceDefinition).Status
		conditionsMap := map[apiextv1.CustomResourceDefinitionConditionType]apiextv1.ConditionStatus{
			apiextv1.NamesAccepted: apiextv1.ConditionTrue,
			apiextv1.Established:   apiextv1.ConditionTrue,
		}

		for _, cond := range status.Conditions {
			if status, ok := conditionsMap[cond.Type]; !ok || cond.Status != status {
				return false
			}
		}

		return true
	}
}
