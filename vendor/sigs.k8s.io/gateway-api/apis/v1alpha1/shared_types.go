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

// GatewayAllowType specifies which Gateways should be allowed to use a Route.
type GatewayAllowType string

const (
	// Any Gateway will be able to use this route.
	GatewayAllowAll GatewayAllowType = "All"
	// Only Gateways that have been  specified in GatewayRefs will be able to use this route.
	GatewayAllowFromList GatewayAllowType = "FromList"
	// Only Gateways within the same namespace as the route will be able to use this route.
	GatewayAllowSameNamespace GatewayAllowType = "SameNamespace"
)

const (
	// AnnotationAppProtocol defines the protocol a Gateway should use for
	// communication with a Kubernetes Service. This annotation must be present
	// on the BackendPolicy resource and the protocol will apply to all Service
	// ports that are selected by BackendPolicy.Spec.BackendRefs. If the
	// AppProtocol field is available, this annotation should not be used. The
	// AppProtocol field, when populated, takes precedence over this annotation.
	// The value of this annotation must be also be a valid value for the
	// AppProtocol field.
	//
	// Examples:
	//
	// - `networking.x-k8s.io/app-protocol: https`
	// - `networking.x-k8s.io/app-protocol: tls`
	AnnotationAppProtocol = "networking.x-k8s.io/app-protocol"
)

// RouteGateways defines which Gateways will be able to use a route. If this
// field results in preventing the selection of a Route by a Gateway, an
// "Admitted" condition with a status of false must be set for the Gateway on
// that Route.
type RouteGateways struct {
	// Allow indicates which Gateways will be allowed to use this route.
	// Possible values are:
	// * All: Gateways in any namespace can use this route.
	// * FromList: Only Gateways specified in GatewayRefs may use this route.
	// * SameNamespace: Only Gateways in the same namespace may use this route.
	//
	// +optional
	// +kubebuilder:validation:Enum=All;FromList;SameNamespace
	// +kubebuilder:default=SameNamespace
	Allow GatewayAllowType `json:"allow,omitempty"`

	// GatewayRefs must be specified when Allow is set to "FromList". In that
	// case, only Gateways referenced in this list will be allowed to use this
	// route. This field is ignored for other values of "Allow".
	//
	// +optional
	GatewayRefs []GatewayReference `json:"gatewayRefs,omitempty"`
}

// PortNumber defines a network port.
//
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=65535
type PortNumber int32

// GatewayReference identifies a Gateway in a specified namespace.
type GatewayReference struct {
	// Name is the name of the referent.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Namespace is the namespace of the referent.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Namespace string `json:"namespace"`
}

// RouteForwardTo defines how a Route should forward a request.
type RouteForwardTo struct {
	// ServiceName refers to the name of the Service to forward matched requests
	// to. When specified, this takes the place of BackendRef. If both
	// BackendRef and ServiceName are specified, ServiceName will be given
	// precedence.
	//
	// If the referent cannot be found, the rule is not included in the route.
	// The controller should raise the "ResolvedRefs" condition on the Gateway
	// with the "DegradedRoutes" reason. The gateway status for this route should
	// be updated with a condition that describes the error more specifically.
	//
	// The protocol to use is defined using AppProtocol field (introduced in
	// Kubernetes 1.18) in the Service resource. In the absence of the
	// AppProtocol field a `networking.x-k8s.io/app-protocol` annotation on the
	// BackendPolicy resource may be used to define the protocol. If the
	// AppProtocol field is available, this annotation should not be used. The
	// AppProtocol field, when populated, takes precedence over the annotation
	// in the BackendPolicy resource. For custom backends, it is encouraged to
	// add a semantically-equivalent field in the Custom Resource Definition.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxLength=253
	ServiceName *string `json:"serviceName,omitempty"`

	// BackendRef is a reference to a backend to forward matched requests to. If
	// both BackendRef and ServiceName are specified, ServiceName will be given
	// precedence.
	//
	// If the referent cannot be found, the rule is not included in the route.
	// The controller should raise the "ResolvedRefs" condition on the Gateway
	// with the "DegradedRoutes" reason. The gateway status for this route should
	// be updated with a condition that describes the error more specifically.
	//
	// Support: Custom
	//
	// +optional
	BackendRef *LocalObjectReference `json:"backendRef,omitempty"`

	// Port specifies the destination port number to use for the
	// backend referenced by the ServiceName or BackendRef field.
	// If unspecified, the destination port in the request is used
	// when forwarding to a backendRef or serviceName.
	//
	// Support: Core
	//
	// +optional
	Port *PortNumber `json:"port,omitempty"`

	// Weight specifies the proportion of HTTP requests forwarded to the backend
	// referenced by the ServiceName or BackendRef field. This is computed as
	// weight/(sum of all weights in this ForwardTo list). For non-zero values,
	// there may be some epsilon from the exact proportion defined here
	// depending on the precision an implementation supports. Weight is not a
	// percentage and the sum of weights does not need to equal 100.
	//
	// If only one backend is specified and it has a weight greater than 0, 100%
	// of the traffic is forwarded to that backend. If weight is set to 0, no
	// traffic should be forwarded for this entry. If unspecified, weight
	// defaults to 1.
	//
	// Support: Extended
	//
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000
	Weight int32 `json:"weight,omitempty"`
}

// RouteConditionType is a type of condition for a route.
type RouteConditionType string

const (
	// This condition indicates whether the route has been admitted
	// or rejected by a Gateway, and why.
	ConditionRouteAdmitted RouteConditionType = "Admitted"
)

// RouteGatewayStatus describes the status of a route with respect to an
// associated Gateway.
type RouteGatewayStatus struct {
	// GatewayRef is a reference to a Gateway object that is associated with
	// the route.
	GatewayRef GatewayReference `json:"gatewayRef"`

	// Conditions describes the status of the route with respect to the
	// Gateway. The "Admitted" condition must always be specified by controllers
	// to indicate whether the route has been admitted or rejected by the Gateway,
	// and why. Note that the route's availability is also subject to the Gateway's
	// own status conditions and listener status.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// RouteStatus defines the observed state that is required across
// all route types.
type RouteStatus struct {
	// Gateways is a list of Gateways that are associated with the route,
	// and the status of the route with respect to each Gateway. When a
	// Gateway selects this route, the controller that manages the Gateway
	// must add an entry to this list when the controller first sees the
	// route and should update the entry as appropriate when the route is
	// modified.
	//
	// A maximum of 100 Gateways will be represented in this list. If this list
	// is full, there may be additional Gateways using this Route that are not
	// included in the list. An empty list means the route has not been admitted
	// by any Gateway.
	//
	// +kubebuilder:validation:MaxItems=100
	Gateways []RouteGatewayStatus `json:"gateways"`
}

// Hostname is used to specify a hostname that should be matched.
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
type Hostname string
