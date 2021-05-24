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
// +kubebuilder:printcolumn:name="Hostnames",type=string,JSONPath=`.spec.hostnames`

// HTTPRoute is the Schema for the HTTPRoute resource.
type HTTPRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of HTTPRoute.
	Spec HTTPRouteSpec `json:"spec,omitempty"`

	// Status defines the current state of HTTPRoute.
	Status HTTPRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HTTPRouteList contains a list of HTTPRoute.
type HTTPRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPRoute `json:"items"`
}

// HTTPRouteSpec defines the desired state of HTTPRoute
type HTTPRouteSpec struct {
	// Gateways defines which Gateways can use this Route.
	//
	// +optional
	// +kubebuilder:default={allow: "SameNamespace"}
	Gateways RouteGateways `json:"gateways,omitempty"`

	// Hostnames defines a set of hostname that should match against
	// the HTTP Host header to select a HTTPRoute to process the request.
	// Hostname is the fully qualified domain name of a network host,
	// as defined by RFC 3986. Note the following deviations from the
	// "host" part of the URI as defined in the RFC:
	//
	// 1. IPs are not allowed.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//
	// Incoming requests are matched against the hostnames before the
	// HTTPRoute rules. If no hostname is specified, traffic is routed
	// based on the HTTPRouteRules.
	//
	// Hostname can be "precise" which is a domain name without the terminating
	// dot of a network host (e.g. "foo.example.com") or "wildcard", which is
	// a domain name prefixed with a single wildcard label (e.g. `*.example.com`).
	// The wildcard character `*` must appear by itself as the first DNS
	// label and matches only a single label.
	// You cannot have a wildcard label by itself (e.g. Host == `*`).
	// Requests will be matched against the Host field in the following order:
	//
	// 1. If Host is precise, the request matches this rule if
	//    the HTTP Host header is equal to Host.
	// 2. If Host is a wildcard, then the request matches this rule if
	//    the HTTP Host header is to equal to the suffix
	//    (removing the first label) of the wildcard rule.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Hostnames []Hostname `json:"hostnames,omitempty"`

	// TLS defines the TLS certificate to use for Hostnames defined in this
	// Route. This configuration only takes effect if the AllowRouteOverride
	// field is set to true in the associated Gateway resource.
	//
	// Collisions can happen if multiple HTTPRoutes define a TLS certificate
	// for the same hostname. In such a case, conflict resolution guiding
	// principles apply, specifically, if hostnames are same and two different
	// certificates are specified then the certificate in the
	// oldest resource wins.
	//
	// Please note that HTTP Route-selection takes place after the
	// TLS Handshake (ClientHello). Due to this, TLS certificate defined
	// here will take precedence even if the request has the potential to
	// match multiple routes (in case multiple HTTPRoutes share the same
	// hostname).
	//
	// Support: Core
	//
	// +optional
	TLS *RouteTLSConfig `json:"tls,omitempty"`

	// Rules are a list of HTTP matchers, filters and actions.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:default={{matches: {{path: {type: "Prefix", value: "/"}}}}}
	Rules []HTTPRouteRule `json:"rules,omitempty"`
}

// RouteTLSConfig describes a TLS configuration defined at the Route level.
type RouteTLSConfig struct {
	// CertificateRef refers to a Kubernetes object that
	// contains a TLS certificate and private key.
	// This certificate MUST be used for TLS handshakes for the domain
	// this RouteTLSConfig is associated with.
	// If an entry in this list omits or specifies the empty
	// string for both the group and kind, the resource defaults to "secrets".
	// An implementation may support other resources (for example, resource
	// "mycertificates" in group "networking.acme.io").
	//
	// Support: Core (Kubernetes Secrets)
	//
	// Support: Implementation-specific (Other resource types)
	//
	CertificateRef LocalObjectReference `json:"certificateRef"`
}

// HTTPRouteRule defines semantics for matching an HTTP request based on
// conditions, optionally executing additional processing steps, and forwarding
// the request to an API object.
type HTTPRouteRule struct {
	// Matches define conditions used for matching the rule against
	// incoming HTTP requests.
	// Each match is independent, i.e. this rule will be matched
	// if **any** one of the matches is satisfied.
	//
	// For example, take the following matches configuration:
	//
	// ```
	// matches:
	// - path:
	//     value: "/foo"
	//   headers:
	//     values:
	//       version: "2"
	// - path:
	//     value: "/v2/foo"
	// ```
	//
	// For a request to match against this rule, a request should satisfy
	// EITHER of the two conditions:
	//
	// - path prefixed with `/foo` AND contains the header `version: "2"`
	// - path prefix of `/v2/foo`
	//
	// See the documentation for HTTPRouteMatch on how to specify multiple
	// match conditions that should be ANDed together.
	//
	// If no matches are specified, the default is a prefix
	// path match on "/", which has the effect of matching every
	// HTTP request.
	//
	//
	// A client request may match multiple HTTP route rules. Matching precedence
	// MUST be determined in order of the following criteria, continuing on ties:
	//
	// * The longest matching hostname.
	// * The longest matching path.
	// * The largest number of header matches
	// * The oldest Route based on creation timestamp. For example, a Route with
	//   a creation timestamp of "2020-09-08 01:02:03" is given precedence over
	//   a Route with a creation timestamp of "2020-09-08 01:02:04".
	// * The Route appearing first in alphabetical order (namespace/name) for
	//   example, foo/bar is given precedence over foo/baz.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:default={{path:{ type: "Prefix", value: "/"}}}
	Matches []HTTPRouteMatch `json:"matches,omitempty"`

	// Filters define the filters that are applied to requests that match
	// this rule.
	//
	// The effects of ordering of multiple behaviors are currently unspecified.
	// This can change in the future based on feedback during the alpha stage.
	//
	// Conformance-levels at this level are defined based on the type of filter:
	//
	// - ALL core filters MUST be supported by all implementations.
	// - Implementers are encouraged to support extended filters.
	// - Implementation-specific custom filters have no API guarantees across
	//   implementations.
	//
	// Specifying a core filter multiple times has unspecified or custom conformance.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Filters []HTTPRouteFilter `json:"filters,omitempty"`

	// ForwardTo defines the backend(s) where matching requests should be sent.
	// If unspecified, the rule performs no forwarding. If unspecified and no
	// filters are specified that would result in a response being sent, a 503
	// error code is returned.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	ForwardTo []HTTPRouteForwardTo `json:"forwardTo,omitempty"`
}

// PathMatchType specifies the semantics of how HTTP paths should be compared.
// Valid PathMatchType values are:
//
// * "Exact"
// * "Prefix"
// * "RegularExpression"
// * "ImplementationSpecific"
//
// Prefix and Exact paths must be syntactically valid:
//
// - Must begin with the '/' character
// - Must not contain consecutive '/' characters (e.g. /foo///, //).
// - For prefix paths, a trailing '/' character in the Path is ignored,
// e.g. /abc and /abc/ specify the same match.
//
// +kubebuilder:validation:Enum=Exact;Prefix;RegularExpression;ImplementationSpecific
type PathMatchType string

// PathMatchType constants.
const (
	PathMatchExact                  PathMatchType = "Exact"
	PathMatchPrefix                 PathMatchType = "Prefix"
	PathMatchRegularExpression      PathMatchType = "RegularExpression"
	PathMatchImplementationSpecific PathMatchType = "ImplementationSpecific"
)

// HeaderMatchType specifies the semantics of how HTTP header values should be
// compared. Valid HeaderMatchType values are:
//
// * "Exact"
// * "RegularExpression"
// * "ImplementationSpecific"
//
// +kubebuilder:validation:Enum=Exact;RegularExpression;ImplementationSpecific
type HeaderMatchType string

// HeaderMatchType constants.
const (
	HeaderMatchExact                  HeaderMatchType = "Exact"
	HeaderMatchRegularExpression      HeaderMatchType = "RegularExpression"
	HeaderMatchImplementationSpecific HeaderMatchType = "ImplementationSpecific"
)

// HTTPPathMatch describes how to select a HTTP route by matching the HTTP request path.
type HTTPPathMatch struct {
	// Type specifies how to match against the path Value.
	//
	// Support: Core (Exact, Prefix)
	//
	// Support: Custom (RegularExpression, ImplementationSpecific)
	//
	// Since RegularExpression PathType has custom conformance, implementations
	// can support POSIX, PCRE or any other dialects of regular expressions.
	// Please read the implementation's documentation to determine the supported
	// dialect.
	//
	// +optional
	// +kubebuilder:default=Prefix
	Type PathMatchType `json:"type,omitempty"`

	// Value of the HTTP path to match against.
	//
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value"`
}

// HTTPHeaderMatch describes how to select a HTTP route by matching HTTP request
// headers.
type HTTPHeaderMatch struct {
	// Type specifies how to match against the value of the header.
	//
	// Support: Core (Exact)
	//
	// Support: Custom (RegularExpression, ImplementationSpecific)
	//
	// Since RegularExpression PathType has custom conformance, implementations
	// can support POSIX, PCRE or any other dialects of regular expressions.
	// Please read the implementation's documentation to determine the supported
	// dialect.
	//
	// HTTP Header name matching MUST be case-insensitive (RFC 2616 - section 4.2).
	//
	// +optional
	// +kubebuilder:default=Exact
	Type HeaderMatchType `json:"type,omitempty"`

	// Values is a map of HTTP Headers to be matched.
	// It MUST contain at least one entry.
	//
	// The HTTP header field name to match is the map key, and the
	// value of the HTTP header is the map value. HTTP header field name matching
	// MUST be case-insensitive.
	//
	// Multiple match values are ANDed together, meaning, a request
	// must match all the specified headers to select the route.
	Values map[string]string `json:"values"`
}

// HTTPRouteMatch defines the predicate used to match requests to a given
// action. Multiple match types are ANDed together, i.e. the match will
// evaluate to true only if all conditions are satisfied.
//
// For example, the match below will match a HTTP request only if its path
// starts with `/foo` AND it contains the `version: "1"` header:
//
// ```
// match:
//   path:
//     value: "/foo"
//   headers:
//     values:
//       version: "1"
// ```
type HTTPRouteMatch struct {
	// Path specifies a HTTP request path matcher. If this field is not
	// specified, a default prefix match on the "/" path is provided.
	//
	// +optional
	// +kubebuilder:default={type: "Prefix", value: "/"}
	Path HTTPPathMatch `json:"path,omitempty"`

	// Headers specifies a HTTP request header matcher.
	//
	// +optional
	Headers *HTTPHeaderMatch `json:"headers,omitempty"`

	// ExtensionRef is an optional, implementation-specific extension to the
	// "match" behavior. For example, resource "myroutematcher" in group
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

// HTTPRouteFilter defines additional processing steps that must be completed
// during the request or response lifecycle. HTTPRouteFilters are meant as an
// extension point to express additional processing that may be done in Gateway
// implementations. Some examples include request or response modification,
// implementing authentication strategies, rate-limiting, and traffic shaping.
// API guarantee/conformance is defined based on the type of the filter.
// TODO(hbagdi): re-render CRDs once controller-tools supports union tags:
// - https://github.com/kubernetes-sigs/controller-tools/pull/298
// - https://github.com/kubernetes-sigs/controller-tools/issues/461
// +union
type HTTPRouteFilter struct {
	// Type identifies the type of filter to apply. As with other API fields,
	// types are classified into three conformance levels:
	//
	// - Core: Filter types and their corresponding configuration defined by
	//   "Support: Core" in this package, e.g. "RequestHeaderModifier". All
	//   implementations must support core filters.
	//
	// - Extended: Filter types and their corresponding configuration defined by
	//   "Support: Extended" in this package, e.g. "RequestMirror". Implementers
	//   are encouraged to support extended filters.
	//
	// - Custom: Filters that are defined and supported by specific vendors.
	//   In the future, filters showing convergence in behavior across multiple
	//   implementations will be considered for inclusion in extended or core
	//   conformance levels. Filter-specific configuration for such filters
	//   is specified using the ExtensionRef field. `Type` should be set to
	//   "ExtensionRef" for custom filters.
	//
	// Implementers are encouraged to define custom implementation types to
	// extend the core API with implementation-specific behavior.
	//
	// +unionDiscriminator
	Type HTTPRouteFilterType `json:"type"`

	// RequestHeaderModifier defines a schema for a filter that modifies request
	// headers.
	//
	// Support: Core
	//
	// +optional
	RequestHeaderModifier *HTTPRequestHeaderFilter `json:"requestHeaderModifier,omitempty"`

	// RequestMirror defines a schema for a filter that mirrors requests.
	//
	// Support: Extended
	//
	// +optional
	RequestMirror *HTTPRequestMirrorFilter `json:"requestMirror,omitempty"`

	// ExtensionRef is an optional, implementation-specific extension to the
	// "filter" behavior.  For example, resource "myroutefilter" in group
	// "networking.acme.io"). ExtensionRef MUST NOT be used for core and
	// extended filters.
	//
	// Support: Implementation-specific
	//
	// +optional
	ExtensionRef *LocalObjectReference `json:"extensionRef,omitempty"`
}

// HTTPRouteFilterType identifies a type of HTTPRoute filter.
// +kubebuilder:validation:Enum=RequestHeaderModifier;RequestMirror;ExtensionRef
type HTTPRouteFilterType string

const (
	// HTTPRouteFilterRequestHeaderModifier can be used to add or remove an HTTP
	// header from an HTTP request before it is sent to the upstream target.
	//
	// Support in HTTPRouteRule: Core
	//
	// Support in HTTPRouteForwardTo: Extended
	HTTPRouteFilterRequestHeaderModifier HTTPRouteFilterType = "RequestHeaderModifier"

	// HTTPRouteFilterRequestMirror can be used to mirror HTTP requests to a
	// different backend. The responses from this backend MUST be ignored by
	// the Gateway.
	//
	// Support in HTTPRouteRule: Extended
	//
	// Support in HTTPRouteForwardTo: Extended
	HTTPRouteFilterRequestMirror HTTPRouteFilterType = "RequestMirror"

	// HTTPRouteFilterExtensionRef should be used for configuring custom
	// HTTP filters.
	//
	// Support in HTTPRouteRule: Custom
	//
	// Support in HTTPRouteForwardTo: Custom
	HTTPRouteFilterExtensionRef HTTPRouteFilterType = "ExtensionRef"
)

// HTTPRequestHeaderFilter defines configuration for the RequestHeaderModifier
// filter.
type HTTPRequestHeaderFilter struct {
	// Set overwrites the request with the given header (name, value)
	// before the action.
	//
	// Input:
	//   GET /foo HTTP/1.1
	//   my-header: foo
	//
	// Config:
	//   set: {"my-header": "bar"}
	//
	// Output:
	//   GET /foo HTTP/1.1
	//   my-header: bar
	//
	// Support: Extended
	//
	// +optional
	Set map[string]string `json:"set,omitempty"`

	// Add adds the given header (name, value) to the request
	// before the action. It appends to any existing values associated
	// with the header name.
	//
	// Input:
	//   GET /foo HTTP/1.1
	//   my-header: foo
	//
	// Config:
	//   add: {"my-header": "bar"}
	//
	// Output:
	//   GET /foo HTTP/1.1
	//   my-header: foo
	//   my-header: bar
	//
	// Support: Extended
	//
	// +optional
	Add map[string]string `json:"add,omitempty"`

	// Remove the given header(s) from the HTTP request before the
	// action. The value of RemoveHeader is a list of HTTP header
	// names. Note that the header names are case-insensitive
	// [RFC-2616 4.2].
	//
	// Input:
	//   GET /foo HTTP/1.1
	//   my-header1: foo
	//   my-header2: bar
	//   my-header3: baz
	//
	// Config:
	//   remove: ["my-header1", "my-header3"]
	//
	// Output:
	//   GET /foo HTTP/1.1
	//   my-header2: bar
	//
	// Support: Extended
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Remove []string `json:"remove,omitempty"`
}

// HTTPRequestMirrorFilter defines configuration for the RequestMirror filter.
type HTTPRequestMirrorFilter struct {
	// ServiceName refers to the name of the Service to mirror matched requests
	// to. When specified, this takes the place of BackendRef. If both
	// BackendRef and ServiceName are specified, ServiceName will be given
	// precedence.
	//
	// If the referent cannot be found, the rule is not included in the route.
	// The controller should raise the "ResolvedRefs" condition on the Gateway
	// with the "DegradedRoutes" reason. The gateway status for this route should
	// be updated with a condition that describes the error more specifically.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxLength=253
	ServiceName *string `json:"serviceName,omitempty"`

	// BackendRef is a local object reference to mirror matched requests to. If
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
	//
	// If unspecified, the destination port in the request is used
	// when forwarding to a backendRef or serviceName.
	//
	// +optional
	Port *PortNumber `json:"port,omitempty"`
}

// HTTPRouteForwardTo defines how a HTTPRoute should forward a request.
type HTTPRouteForwardTo struct {
	// ServiceName refers to the name of the Service to forward matched requests
	// to. When specified, this takes the place of BackendRef. If both
	// BackendRef and ServiceName are specified, ServiceName will be given
	// precedence.
	//
	// If the referent cannot be found, the route must be dropped
	// from the Gateway. The controller should raise the "ResolvedRefs"
	// condition on the Gateway with the "DegradedRoutes" reason.
	// The gateway status for this route should be updated with a
	// condition that describes the error more specifically.
	//
	// The protocol to use should be specified with the AppProtocol field on Service
	// resources. This field was introduced in Kubernetes 1.18. If using an earlier version
	// of Kubernetes, a `networking.x-k8s.io/app-protocol` annotation on the
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
	// If the referent cannot be found, the route must be dropped
	// from the Gateway. The controller should raise the "ResolvedRefs"
	// condition on the Gateway with the "DegradedRoutes" reason.
	// The gateway status for this route should be updated with a
	// condition that describes the error more specifically.
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
	// Support: Core
	//
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000
	Weight int32 `json:"weight,omitempty"`

	// Filters defined at this-level should be executed if and only if the
	// request is being forwarded to the backend defined here.
	//
	// Support: Custom (For broader support of filters, use the Filters field
	// in HTTPRouteRule.)
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Filters []HTTPRouteFilter `json:"filters,omitempty"`
}

// HTTPRouteStatus defines the observed state of HTTPRoute.
type HTTPRouteStatus struct {
	RouteStatus `json:",inline"`
}
