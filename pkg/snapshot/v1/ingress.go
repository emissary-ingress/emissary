package snapshot

import (
	"encoding/json"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kates/k8s_resource_types"
)

type Ingress struct {
	k8s_resource_types.Ingress
}

func (ingress *Ingress) UnmarshalJSON(bs []byte) error {
	var untyped kates.Unstructured
	if err := json.Unmarshal(bs, &untyped); err != nil {
		return err
	}
	typed, err := k8s_resource_types.NewIngress(&untyped)
	if err != nil {
		return err
	}
	ingress.Ingress = *typed
	return nil
}

type IngressClass struct {
	k8s_resource_types.IngressClass
}

func (ingressclass *IngressClass) UnmarshalJSON(bs []byte) error {
	var untyped kates.Unstructured
	if err := json.Unmarshal(bs, &untyped); err != nil {
		return err
	}
	typed, err := k8s_resource_types.NewIngressClass(&untyped)
	if err != nil {
		return err
	}
	ingressclass.IngressClass = *typed
	return nil
}
