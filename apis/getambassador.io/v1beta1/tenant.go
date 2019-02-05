package v1

import (
	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type Tenant struct {
	Metadata *metav1.ObjectMeta `json:"metadata"`
	Spec     *TenantSpec        `json:"spec"`
}

type TenantSpec struct {
	Tenants []TenantObject `json:"tenants"`
}

type TenantObject struct {
	CallbackURL string `json:"-"` // is calculated from TenantURL
	TenantURL   string `json:"tenantUrl"`
	TLS         bool   `json:"tls"`
	Domain      string `json:"-"` // is calculated from TenantURL
	Audience    string `json:"audience"`
	ClientID    string `json:"clientId"`
	Secret      string `json:"secret"`
}

// register //////////////////////////////////////////////////////////

// Required to implement k8s.Resource
func (crd *Tenant) GetMetadata() *metav1.ObjectMeta {
	return crd.Metadata
}

type TenantList struct {
	Metadata *metav1.ListMeta `json:"metadata"`
	Items    []*Tenant        `json:"items"`
}

// Require for TenantList to implement k8s.ResourceList
func (crdl *TenantList) GetMetadata() *metav1.ListMeta {
	return crdl.Metadata
}

func init() {
	k8s.Register("getambassador.io", "v1beta1", "tenants", true, &Tenant{})
	k8s.RegisterList("getambassador.io", "v1beta1", "tenants", true, &TenantList{})
}
