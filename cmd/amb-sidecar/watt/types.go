package watt

import (
	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
)

type Snapshot struct {
	Consul     ResourceCollection
	Kubernetes ResourceCollection
}

type ResourceCollection struct {
	Host   []*ambassadorTypesV2.Host `json:"Host"`   // yes, uppercase
	Secret []*k8sTypesCoreV1.Secret  `json:"secret"` // yes, lowercase
}
