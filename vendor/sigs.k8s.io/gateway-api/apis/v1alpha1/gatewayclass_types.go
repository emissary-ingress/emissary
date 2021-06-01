/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=gc
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Controller",type=string,JSONPath=`.spec.controller`

// GatewayClass describes a class of Gateways available to the user
// for creating Gateway resources.
//
// GatewayClass is a Cluster level resource.
type GatewayClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of GatewayClass.
	Spec GatewayClassSpec `json:"spec,omitempty"`

	// Status defines the current state of GatewayClass.
	//
	// +kubebuilder:default={conditions: {{type: "Admitted", status: "False", message: "Waiting for controller", reason: "Waiting", lastTransitionTime: "1970-01-01T00:00:00Z"}}}
	Status GatewayClassStatus `json:"status,omitempty"`
}

// GatewayClassSpec reflects the configuration of a class of Gateways.
type GatewayClassSpec struct {
	// Controller is a domain/path string that indicates the
	// controller that is managing Gateways of this class.
	//
	// Example: "acme.io/gateway-controller".
	//
	// This field is not mutable and cannot be empty.
	//
	// The format of this field is DOMAIN "/" PATH, where DOMAIN
	// and PATH are valid Kubernetes names
	// (https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names).
	//
	// Support: Core
	//
	// +kubebuilder:validation:MaxLength=253
	Controller string `json:"controller"`

	// ParametersRef is a reference to a resource that contains the configuration
	// parameters corresponding to the GatewayClass. This is optional if the
	// controller does not require any additional configuration.
	//
	// ParametersRef can reference a standard Kubernetes resource, i.e. ConfigMap,
	// or an implementation-specific custom resource. The resource can be
	// cluster-scoped or namespace-scoped.
	//
	// If the referent cannot be found, the GatewayClass's "InvalidParameters"
	// status condition will be true.
	//
	// Support: Custom
	//
	// +optional
	ParametersRef *ParametersReference `json:"parametersRef,omitempty"`
}

// ParametersReference identifies an API object containing controller-specific
// configuration resource within the cluster.
type ParametersReference struct {
	// Group is the group of the referent.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Group string `json:"group"`

	// Kind is kind of the referent.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Kind string `json:"kind"`

	// Name is the name of the referent.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Scope represents if the referent is a Cluster or Namespace scoped resource.
	// This may be set to "Cluster" or "Namespace".
	// +kubebuilder:validation:Enum=Cluster;Namespace
	// +kubebuilder:default=Cluster
	// +optional
	Scope string `json:"scope,omitempty"`

	// Namespace is the namespace of the referent.
	// This field is required when scope is set to "Namespace" and ignored when
	// scope is set to "Cluster".
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// GatewayClassConditionType is the type for status conditions on
// Gateway resources. This type should be used with the
// GatewayClassStatus.Conditions field.
type GatewayClassConditionType string

// GatewayClassConditionReason defines the set of reasons that explain why
// a particular GatewayClass condition type has been raised.
type GatewayClassConditionReason string

const (
	// This condition indicates whether the GatewayClass has been
	// admitted by the controller requested in the `spec.controller`
	// field.
	//
	// This condition defaults to False, and MUST be set by a controller when it sees
	// a GatewayClass using its controller string.
	// The status of this condition MUST be set to true if the controller will support
	// provisioning Gateways using this class. Otherwise, this status MUST be set to false.
	// If the status is set to false, the controller SHOULD set a Message and Reason as an
	// explanation.
	//
	// Controllers should prefer to use the values of GatewayClassConditionReason
	// for the corresponding Reason, where appropriate.
	GatewayClassConditionStatusAdmitted GatewayClassConditionType = "Admitted"

	// This reason is used with the "Admitted" condition when the
	// GatewayClass was not admitted because the parametersRef field
	// was invalid, with more detail in the message.
	GatewayClassNotAdmittedInvalidParameters GatewayClassConditionReason = "InvalidParameters"

	// This reason is used with the "Admitted" condition when the
	// requested controller has not yet made a decision about whether
	// to admit the GatewayClass. It is the default Reason on a new
	// GatewayClass. It indicates
	GatewayClassNotAdmittedWaiting GatewayClassConditionReason = "Waiting"

	// GatewayClassFinalizerGatewaysExist should be added as a finalizer to the
	// GatewayClass whenever there are provisioned Gateways using a GatewayClass.
	GatewayClassFinalizerGatewaysExist = "gateway-exists-finalizer.networking.x-k8s.io"
)

// GatewayClassStatus is the current status for the GatewayClass.
type GatewayClassStatus struct {
	// Conditions is the current status from the controller for
	// this GatewayClass.
	//
	// Controllers should prefer to publish conditions using values
	// of GatewayClassConditionType for the type of each Condition.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:default={{type: "Admitted", status: "False", message: "Waiting for controller", reason: "Waiting", lastTransitionTime: "1970-01-01T00:00:00Z"}}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayClassList contains a list of GatewayClass
type GatewayClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayClass `json:"items"`
}
