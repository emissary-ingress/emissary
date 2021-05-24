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
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TCPRoute is the Schema for the TCPRoute resource.
type TCPRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of TCPRoute.
	Spec TCPRouteSpec `json:"spec,omitempty"`

	// Status defines the current state of TCPRoute.
	Status TCPRouteStatus `json:"status,omitempty"`
}

// TCPRouteSpec defines the desired state of TCPRoute
type TCPRouteSpec struct {
	// Rules are a list of TCP matchers and actions.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Rules []TCPRouteRule `json:"rules"`

	// Gateways defines which Gateways can use this Route.
	//
	// +optional
	// +kubebuilder:default={allow: "SameNamespace"}
	Gateways RouteGateways `json:"gateways,omitempty"`
}

// TCPRouteStatus defines the observed state of TCPRoute
type TCPRouteStatus struct {
	RouteStatus `json:",inline"`
}

// TCPRouteRule is the configuration for a given rule.
type TCPRouteRule struct {
	// Matches define conditions used for matching the rule against
	// incoming TCP connections. Each match is independent, i.e. this
	// rule will be matched if **any** one of the matches is satisfied.
	// If unspecified, all requests from the associated gateway TCP
	// listener will match.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=8
	Matches []TCPRouteMatch `json:"matches,omitempty"`

	// ForwardTo defines the backend(s) where matching requests should
	// be sent.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	ForwardTo []RouteForwardTo `json:"forwardTo"`
}

// TCPRouteMatch defines the predicate used to match connections to a
// given action.
type TCPRouteMatch struct {
	// ExtensionRef is an optional, implementation-specific extension to the
	// "match" behavior.  For example, resource "mytcproutematcher" in group
	// "networking.acme.io". If the referent cannot be found, the rule is not
	// included in the route. The controller should raise the "ResolvedRefs"
	// condition on the Gateway with the "DegradedRoutes" reason. The gateway
	// status for this route should be updated with a condition that describes
	// the error more specifically.
	//
	// Support: Custom
	//
	// +optional
	ExtensionRef *LocalObjectReference `json:"extensionRef,omitempty"`
}

// +kubebuilder:object:root=true

// TCPRouteList contains a list of TCPRoute
type TCPRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TCPRoute `json:"items"`
}
