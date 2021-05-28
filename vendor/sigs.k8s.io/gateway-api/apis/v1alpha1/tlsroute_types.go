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

// The TLSRoute resource is similar to TCPRoute, but can be configured
// to match against TLS-specific metadata. This allows more flexibility
// in matching streams for a given TLS listener.
//
// If you need to forward traffic to a single target for a TLS listener, you
// could choose to use a TCPRoute with a TLS listener.
type TLSRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of TLSRoute.
	Spec TLSRouteSpec `json:"spec,omitempty"`

	// Status defines the current state of TLSRoute.
	Status TLSRouteStatus `json:"status,omitempty"`
}

// TLSRouteSpec defines the desired state of a TLSRoute resource.
type TLSRouteSpec struct {
	// Rules are a list of TLS matchers and actions.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Rules []TLSRouteRule `json:"rules"`

	// Gateways defines which Gateways can use this Route.
	//
	// +optional
	// +kubebuilder:default={allow: "SameNamespace"}
	Gateways RouteGateways `json:"gateways,omitempty"`
}

// TLSRouteStatus defines the observed state of TLSRoute
type TLSRouteStatus struct {
	RouteStatus `json:",inline"`
}

// TLSRouteRule is the configuration for a given rule.
type TLSRouteRule struct {
	// Matches define conditions used for matching the rule against an
	// incoming TLS handshake. Each match is independent, i.e. this
	// rule will be matched if **any** one of the matches is satisfied.
	// If unspecified, all requests from the associated gateway TLS
	// listener will match.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=8
	Matches []TLSRouteMatch `json:"matches,omitempty"`

	// ForwardTo defines the backend(s) where matching requests should be
	// sent.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	ForwardTo []RouteForwardTo `json:"forwardTo"`
}

// TLSRouteMatch defines the predicate used to match connections to a
// given action.
type TLSRouteMatch struct {
	// SNIs defines a set of SNI names that should match against the
	// SNI attribute of TLS ClientHello message in TLS handshake.
	//
	// SNI can be "precise" which is a domain name without the terminating
	// dot of a network host (e.g. "foo.example.com") or "wildcard", which is
	// a domain name prefixed with a single wildcard label (e.g. `*.example.com`).
	// The wildcard character `*` must appear by itself as the first DNS label
	// and matches only a single label. You cannot have a wildcard label by
	// itself (e.g. Host == `*`).
	//
	// Requests will be matched against the Host field in the following order:
	//
	// 1. If SNI is precise, the request matches this rule if the SNI in
	//    ClientHello is equal to one of the defined SNIs.
	// 2. If SNI is a wildcard, then the request matches this rule if the
	//    SNI is to equal to the suffix (removing the first label) of the
	//    wildcard rule.
	// 3. If SNIs is unspecified, all requests associated with the gateway TLS
	//    listener will match. This can be used to define a default backend
	//    for a TLS listener.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	SNIs []Hostname `json:"snis,omitempty"`

	// ExtensionRef is an optional, implementation-specific extension to the
	// "match" behavior.  For example, resource "mytlsroutematcher" in group
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

// TLSRouteList contains a list of TLSRoute
type TLSRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TLSRoute `json:"items"`
}
