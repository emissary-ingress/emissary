package watt

import (
	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Snapshot struct {
	Consul     ResourceCollection
	Kubernetes ResourceCollection
}

type ResourceCollection struct {
	Host []Host
}

type Host struct {
	k8sTypesMetaV1.TypeMeta   `json:",inline"`
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      ambassadorTypesV2.HostSpec `json:"spec"`
}
