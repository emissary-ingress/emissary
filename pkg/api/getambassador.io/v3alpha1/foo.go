package v3alpha1

import (
	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FooSpec defines the desired state of Foo
type FooSpec struct {
	AmbassadorID v2.AmbassadorID `json:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Required
	Foo string `json:"auth_service,omitempty"`
}

// Foo is the Schema for the authservices API
//
// +kubebuilder:object:root=true
type Foo struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FooSpec `json:"spec,omitempty"`
}

// FooList contains a list of Foos.
//
// +kubebuilder:object:root=true
type FooList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Foo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Foo{}, &FooList{})
}
