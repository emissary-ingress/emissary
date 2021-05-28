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
// +kubebuilder:resource:shortName=gtw
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Class",type=string,JSONPath=`.spec.gatewayClassName`

// Gateway represents an instantiation of a service-traffic handling
// infrastructure by binding Listeners to a set of IP addresses.
//
// Implementations should add the `gateway-exists-finalizer.networking.x-k8s.io`
// finalizer on the associated GatewayClass whenever Gateway(s) is running.
// This ensures that a GatewayClass associated with a Gateway(s) is not
// deleted while in use.
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Gateway.
	Spec GatewaySpec `json:"spec,omitempty"`

	// Status defines the current state of Gateway.
	//
	// +kubebuilder:default={conditions: {{type: "Scheduled", status: "False", reason:"NotReconciled", message:"Waiting for controller", lastTransitionTime: "1970-01-01T00:00:00Z"}}}
	Status GatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayList contains a list of Gateway.
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

// GatewaySpec defines the desired state of Gateway.
//
// Not all possible combinations of options specified in the Spec are
// valid. Some invalid configurations can be caught synchronously via a
// webhook, but there are many cases that will require asynchronous
// signaling via the GatewayStatus block.
type GatewaySpec struct {
	// GatewayClassName used for this Gateway. This is the name of a
	// GatewayClass resource.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	GatewayClassName string `json:"gatewayClassName"`

	// Listeners associated with this Gateway. Listeners define
	// logical endpoints that are bound on this Gateway's addresses.
	// At least one Listener MUST be specified.
	//
	// An implementation MAY group Listeners by Port and then collapse each
	// group of Listeners into a single Listener if the implementation
	// determines that the Listeners in the group are "compatible". An
	// implementation MAY also group together and collapse compatible
	// Listeners belonging to different Gateways.
	//
	// For example, an implementation might consider Listeners to be
	// compatible with each other if all of the following conditions are
	// met:
	//
	// 1. Either each Listener within the group specifies the "HTTP"
	//    Protocol or each Listener within the group specifies either
	//    the "HTTPS" or "TLS" Protocol.
	//
	// 2. Each Listener within the group specifies a Hostname that is unique
	//    within the group.
	//
	// 3. As a special case, one Listener within a group may omit Hostname,
	//    in which case this Listener matches when no other Listener
	//    matches.
	//
	// If the implementation does collapse compatible Listeners, the
	// hostname provided in the incoming client request MUST be
	// matched to a Listener to find the correct set of Routes.
	// The incoming hostname MUST be matched using the Hostname
	// field for each Listener in order of most to least specific.
	// That is, exact matches must be processed before wildcard
	// matches.
	//
	// If this field specifies multiple Listeners that have the same
	// Port value but are not compatible, the implementation must raise
	// a "Conflicted" condition in the Listener status.
	//
	// Support: Core
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	Listeners []Listener `json:"listeners"`

	// Addresses requested for this gateway. This is optional and
	// behavior can depend on the GatewayClass. If a value is set
	// in the spec and the requested address is invalid, the
	// GatewayClass MUST indicate this in the associated entry in
	// GatewayStatus.Addresses.
	//
	// If no Addresses are specified, the GatewayClass may
	// schedule the Gateway in an implementation-defined manner,
	// assigning an appropriate set of Addresses.
	//
	// The GatewayClass MUST bind all Listeners to every
	// GatewayAddress that it assigns to the Gateway.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Addresses []GatewayAddress `json:"addresses,omitempty"`
}

// Listener embodies the concept of a logical endpoint where a Gateway can
// accept network connections. Each listener in a Gateway must have a unique
// combination of Hostname, Port, and Protocol. This will be enforced by a
// validating webhook.
type Listener struct {
	// Hostname specifies the virtual hostname to match for protocol types that
	// define this concept. When unspecified, "", or `*`, all hostnames are
	// matched. This field can be omitted for protocols that don't require
	// hostname based matching.
	//
	// Hostname is the fully qualified domain name of a network host, as defined
	// by RFC 3986. Note the following deviations from the "host" part of the
	// URI as defined in the RFC:
	//
	// 1. IP literals are not allowed.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//
	// Hostname can be "precise" which is a domain name without the terminating
	// dot of a network host (e.g. "foo.example.com") or "wildcard", which is a
	// domain name prefixed with a single wildcard label (e.g. `*.example.com`).
	// The wildcard character `*` must appear by itself as the first DNS label
	// and matches only a single label.
	//
	// Support: Core
	//
	// +optional
	Hostname *Hostname `json:"hostname,omitempty"`

	// Port is the network port. Multiple listeners may use the
	// same port, subject to the Listener compatibility rules.
	//
	// Support: Core
	Port PortNumber `json:"port"`

	// Protocol specifies the network protocol this listener expects to receive.
	// The GatewayClass MUST apply the Hostname match appropriately for each
	// protocol:
	//
	// * For the "TLS" protocol, the Hostname match MUST be
	//   applied to the [SNI](https://tools.ietf.org/html/rfc6066#section-3)
	//   server name offered by the client.
	// * For the "HTTP" protocol, the Hostname match MUST be
	//   applied to the host portion of the
	//   [effective request URI](https://tools.ietf.org/html/rfc7230#section-5.5)
	//   or the [:authority pseudo-header](https://tools.ietf.org/html/rfc7540#section-8.1.2.3)
	// * For the "HTTPS" protocol, the Hostname match MUST be
	//   applied at both the TLS and HTTP protocol layers.
	//
	// Support: Core
	Protocol ProtocolType `json:"protocol"`

	// TLS is the TLS configuration for the Listener. This field
	// is required if the Protocol field is "HTTPS" or "TLS" and
	// ignored otherwise.
	//
	// The association of SNIs to Certificate defined in GatewayTLSConfig is
	// defined based on the Hostname field for this listener.
	//
	// The GatewayClass MUST use the longest matching SNI out of all
	// available certificates for any TLS handshake.
	//
	// Support: Core
	//
	// +optional
	TLS *GatewayTLSConfig `json:"tls,omitempty"`

	// Routes specifies a schema for associating routes with the
	// Listener using selectors. A Route is a resource capable of
	// servicing a request and allows a cluster operator to expose
	// a cluster resource (i.e. Service) by externally-reachable
	// URL, load-balance traffic and terminate SSL/TLS.  Typically,
	// a route is a "HTTPRoute" or "TCPRoute" in group
	// "networking.x-k8s.io", however, an implementation may support
	// other types of resources.
	//
	// The Routes selector MUST select a set of objects that
	// are compatible with the application protocol specified in
	// the Protocol field.
	//
	// Although a client request may technically match multiple route rules,
	// only one rule may ultimately receive the request. Matching precedence
	// MUST be determined in order of the following criteria:
	//
	// * The most specific match. For example, the most specific HTTPRoute match
	//   is determined by the longest matching combination of hostname and path.
	// * The oldest Route based on creation timestamp. For example, a Route with
	//   a creation timestamp of "2020-09-08 01:02:03" is given precedence over
	//   a Route with a creation timestamp of "2020-09-08 01:02:04".
	// * If everything else is equivalent, the Route appearing first in
	//   alphabetical order (namespace/name) should be given precedence. For
	//   example, foo/bar is given precedence over foo/baz.
	//
	// All valid portions of a Route selected by this field should be supported.
	// Invalid portions of a Route can be ignored (sometimes that will mean the
	// full Route). If a portion of a Route transitions from valid to invalid,
	// support for that portion of the Route should be dropped to ensure
	// consistency. For example, even if a filter specified by a Route is
	// invalid, the rest of the Route should still be supported.
	//
	// Support: Core
	Routes RouteBindingSelector `json:"routes"`
}

// ProtocolType defines the application protocol accepted by a Listener.
// Implementations are not required to accept all the defined protocols.
// If an implementation does not support a specified protocol, it
// should raise a "Detached" condition for the affected Listener with
// a reason of "UnsupportedProtocol".
//
// Core ProtocolType values are listed in the table below.
//
// Implementations can define their own protocols if a core ProtocolType does not
// exist. Such definitions must use prefixed name, such as
// `mycompany.com/my-custom-protocol`. Un-prefixed names are reserved for core
// protocols. Any protocol defined by implementations will fall under custom
// conformance.
type ProtocolType string

const (
	// Accepts cleartext HTTP/1.1 sessions over TCP.
	HTTPProtocolType ProtocolType = "HTTP"

	// Accepts HTTP/1.1 or HTTP/2 sessions over TLS.
	HTTPSProtocolType ProtocolType = "HTTPS"

	// Accepts TLS sessions over TCP.
	TLSProtocolType ProtocolType = "TLS"

	// Accepts TCP sessions.
	TCPProtocolType ProtocolType = "TCP"

	// Accepts UDP packets.
	UDPProtocolType ProtocolType = "UDP"
)

// TLSRouteOverrideType type defines the level of allowance for Routes
// to override a specific TLS setting.
// +kubebuilder:validation:Enum=Allow;Deny
// +kubebuilder:default=Deny
type TLSRouteOverrideType string

const (
	// Allows the parameter to be configured from all routes.
	TLSROuteOVerrideAllow TLSRouteOverrideType = "Allow"

	// Prohibits the parameter from being configured from any route.
	TLSRouteOverrideDeny TLSRouteOverrideType = "Deny"
)

// TLSOverridePolicy defines a schema for overriding TLS settings at the Route
// level.
type TLSOverridePolicy struct {
	// Certificate dictates if TLS certificates can be configured
	// via Routes. If set to 'Allow', a TLS certificate for a hostname
	// defined in a Route takes precedence over the certificate defined in
	// Gateway.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default=Deny
	Certificate TLSRouteOverrideType `json:"certificate,omitempty"`
}

// GatewayTLSConfig describes a TLS configuration.
//
// References:
//
// - nginx: https://nginx.org/en/docs/http/configuring_https_servers.html
// - envoy: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/cert.proto
// - haproxy: https://www.haproxy.com/documentation/aloha/9-5/traffic-management/lb-layer7/tls/
// - gcp: https://cloud.google.com/load-balancing/docs/use-ssl-policies#creating_an_ssl_policy_with_a_custom_profile
// - aws: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/create-https-listener.html#describe-ssl-policies
// - azure: https://docs.microsoft.com/en-us/azure/app-service/configure-ssl-bindings#enforce-tls-1112
type GatewayTLSConfig struct {
	// Mode defines the TLS behavior for the TLS session initiated by the client.
	// There are two possible modes:
	// - Terminate: The TLS session between the downstream client
	//   and the Gateway is terminated at the Gateway. This mode requires
	//   certificateRef to be set.
	// - Passthrough: The TLS session is NOT terminated by the Gateway. This
	//   implies that the Gateway can't decipher the TLS stream except for
	//   the ClientHello message of the TLS protocol.
	//   CertificateRef field is ignored in this mode.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default=Terminate
	Mode TLSModeType `json:"mode,omitempty"`

	// CertificateRef is the reference to Kubernetes object that contain a
	// TLS certificate and private key. This certificate MUST be used for
	// TLS handshakes for the domain this GatewayTLSConfig is associated with.
	//
	// This field is required when mode is set to "Terminate" (default) and
	// optional otherwise.
	//
	// If an entry in this list omits or specifies the empty string for both
	// the group and the resource, the resource defaults to "secrets". An
	// implementation may support other resources (for example, resource
	// "mycertificates" in group "networking.acme.io").
	//
	// Support: Core (Kubernetes Secrets)
	//
	// Support: Implementation-specific (Other resource types)
	//
	// +optional
	CertificateRef *LocalObjectReference `json:"certificateRef,omitempty"`

	// RouteOverride dictates if TLS settings can be configured
	// via Routes or not.
	//
	// CertificateRef must be defined even if `routeOverride.certificate` is
	// set to 'Allow' as it will be used as the default certificate for the
	// listener.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default={certificate:Deny}
	RouteOverride TLSOverridePolicy `json:"routeOverride,omitempty"`

	// Options are a list of key/value pairs to give extended options
	// to the provider.
	//
	// There variation among providers as to how ciphersuites are
	// expressed. If there is a common subset for expressing ciphers
	// then it will make sense to loft that as a core API
	// construct.
	//
	// Support: Implementation-specific
	//
	// +optional
	Options map[string]string `json:"options,omitempty"`
}

// TLSModeType type defines how a Gateway handles TLS sessions.
//
// +kubebuilder:validation:Enum=Terminate;Passthrough
type TLSModeType string

const (
	// In this mode, TLS session between the downstream client
	// and the Gateway is terminated at the Gateway.
	TLSModeTerminate TLSModeType = "Terminate"
	// In this mode, the TLS session is NOT terminated by the Gateway. This
	// implies that the Gateway can't decipher the TLS stream except for
	// the ClientHello message of the TLS protocol.
	TLSModePassthrough TLSModeType = "Passthrough"
)

// RouteBindingSelector defines a schema for associating routes with the Gateway.
// If Namespaces and Selector are defined, only routes matching both selectors are
// associated with the Gateway.
type RouteBindingSelector struct {
	// Namespaces indicates in which namespaces Routes should be selected
	// for this Gateway. This is restricted to the namespace of this Gateway by
	// default.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default={from: Same}
	Namespaces RouteNamespaces `json:"namespaces,omitempty"`
	// Selector specifies a set of route labels used for selecting
	// routes to associate with the Gateway. If this Selector is defined,
	// only routes matching the Selector are associated with the Gateway.
	// An empty Selector matches all routes.
	//
	// Support: Core
	//
	// +optional
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Group is the group of the route resource to select. Omitting the value or specifying
	// the empty string indicates the networking.x-k8s.io API group.
	// For example, use the following to select an HTTPRoute:
	//
	// routes:
	//   kind: HTTPRoute
	//
	// Otherwise, if an alternative API group is desired, specify the desired
	// group:
	//
	// routes:
	//   group: acme.io
	//   kind: FooRoute
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default=networking.x-k8s.io
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Group string `json:"group,omitempty"`
	// Kind is the kind of the route resource to select.
	//
	// Kind MUST correspond to kinds of routes that are compatible with the
	// application protocol specified in the Listener's Protocol field.
	//
	// If an implementation does not support or recognize this
	// resource type, it SHOULD set the "ResolvedRefs" condition to false for
	// this listener with the "InvalidRoutesRef" reason.
	//
	// Support: Core
	Kind string `json:"kind"`
}

// RouteSelectType specifies where Routes should be selected by a Gateway.
//
// +kubebuilder:validation:Enum=All;Selector;Same
type RouteSelectType string

const (
	// Routes in all namespaces may be used by this Gateway.
	RouteSelectAll RouteSelectType = "All"
	// Only Routes in namespaces selected by the selector may be used by this Gateway.
	RouteSelectSelector RouteSelectType = "Selector"
	// Only Routes in the same namespace as the Gateway may be used by this Gateway.
	RouteSelectSame RouteSelectType = "Same"
)

// RouteNamespaces indicate which namespaces Routes should be selected from.
type RouteNamespaces struct {
	// From indicates where Routes will be selected for this Gateway. Possible
	// values are:
	// * All: Routes in all namespaces may be used by this Gateway.
	// * Selector: Routes in namespaces selected by the selector may be used by
	//   this Gateway.
	// * Same: Only Routes in the same namespace may be used by this Gateway.
	//
	// Support: Core
	//
	// +optional
	// +kubebuilder:default=Same
	From RouteSelectType `json:"from,omitempty"`

	// Selector must be specified when From is set to "Selector". In that case,
	// only Routes in Namespaces matching this Selector will be selected by this
	// Gateway. This field is ignored for other values of "From".
	//
	// Support: Core
	//
	// +optional
	Selector metav1.LabelSelector `json:"selector,omitempty"`
}

// GatewayAddress describes an address that can be bound to a Gateway.
type GatewayAddress struct {
	// Type of the address.
	//
	// Support: Extended
	//
	// +optional
	// +kubebuilder:default=IPAddress
	Type AddressType `json:"type,omitempty"`

	// Value of the address. The validity of the values will depend
	// on the type and support by the controller.
	//
	// Examples: `1.2.3.4`, `128::1`, `my-ip-address`.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Value string `json:"value"`
}

// AddressType defines how a network address is represented as a text string.
//
// If the requested address is unsupported, the controller
// should raise the "Detached" listener status condition on
// the Gateway with the "UnsupportedAddress" reason.
//
// +kubebuilder:validation:Enum=IPAddress;NamedAddress
type AddressType string

const (
	// A textual representation of a numeric IP address. IPv4
	// addresses must be in dotted-decimal form. IPv6 addresses
	// must be in a standard IPv6 text representation
	// (see [RFC 5952](https://tools.ietf.org/html/rfc5952)).
	//
	// Support: Extended
	IPAddressType AddressType = "IPAddress"

	// An opaque identifier that represents a specific IP address. The
	// interpretation of the name is dependent on the controller. For
	// example, a "NamedAddress" might be a cloud-dependent identifier
	// for a static or elastic IP.
	//
	// Support: Implementation-specific
	NamedAddressType AddressType = "NamedAddress"
)

// GatewayStatus defines the observed state of Gateway.
type GatewayStatus struct {
	// Addresses lists the IP addresses that have actually been
	// bound to the Gateway. These addresses may differ from the
	// addresses in the Spec, e.g. if the Gateway automatically
	// assigns an address from a reserved pool.
	//
	// These addresses should all be of type "IPAddress".
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Addresses []GatewayAddress `json:"addresses,omitempty"`

	// Conditions describe the current conditions of the Gateway.
	//
	// Implementations should prefer to express Gateway conditions
	// using the `GatewayConditionType` and `GatewayConditionReason`
	// constants so that operators and tools can converge on a common
	// vocabulary to describe Gateway state.
	//
	// Known condition types are:
	//
	// * "Scheduled"
	// * "Ready"
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:default={{type: "Scheduled", status: "False", reason:"NotReconciled", message:"Waiting for controller", lastTransitionTime: "1970-01-01T00:00:00Z"}}
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Listeners provide status for each unique listener port defined in the Spec.
	//
	// +optional
	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MaxItems=64
	Listeners []ListenerStatus `json:"listeners,omitempty"`
}

// GatewayConditionType is a type of condition associated with a
// Gateway. This type should be used with the GatewayStatus.Conditions
// field.
type GatewayConditionType string

// GatewayConditionReason defines the set of reasons that explain
// why a particular Gateway condition type has been raised.
type GatewayConditionReason string

const (
	// This condition is true when the controller managing the
	// Gateway has scheduled the Gateway to the underlying network
	// infrastructure.
	//
	// Possible reasons for this condition to be false are:
	//
	// * "NotReconciled"
	// * "NoSuchGatewayClass"
	// * "NoResources"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	GatewayConditionScheduled GatewayConditionType = "Scheduled"

	// This reason is used with the "Scheduled" condition when
	// been recently created and no controller has reconciled it yet.
	GatewayReasonNotReconciled GatewayConditionReason = "NotReconciled"

	// This reason is used with the "Scheduled" condition when the Gateway
	// is not scheduled because there is no controller that recognizes
	// the GatewayClassName. This reason should only be set by
	// a controller that has cluster-wide visibility of all the
	// installed GatewayClasses.
	GatewayReasonNoSuchGatewayClass GatewayConditionReason = "NoSuchGatewayClass"

	// This reason is used with the "Scheduled" condition when the
	// Gateway is not scheduled because insufficient infrastructure
	// resources are available.
	GatewayReasonNoResources GatewayConditionReason = "NoResources"
)

const (
	// This condition is true when the Gateway is expected to be able
	// to serve traffic. Note that this does not indicate that the
	// Gateway configuration is current or even complete (e.g. the
	// controller may still not have reconciled the latest version,
	// or some parts of the configuration could be missing).
	//
	// If both the "ListenersNotValid" and "ListenersNotReady"
	// reasons are true, the Gateway controller should prefer the
	// "ListenersNotValid" reason.
	//
	// Possible reasons for this condition to be false are:
	//
	// * "ListenersNotValid"
	// * "ListenersNotReady"
	// * "AddressNotAssigned"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	GatewayConditionReady GatewayConditionType = "Ready"

	// This reason is used with the "Ready" condition when one or
	// more Listeners have an invalid or unsupported configuration
	// and cannot be configured on the Gateway.
	GatewayReasonListenersNotValid GatewayConditionReason = "ListenersNotValid"

	// This reason is used with the "Ready" condition when one or
	// more Listeners are not ready to serve traffic.
	GatewayReasonListenersNotReady GatewayConditionReason = "ListenersNotReady"

	// This reason is used with the "Ready" condition when the requested
	// address has not been assigned to the Gateway. This reason
	// can be used to express a range of circumstances, including
	// (but not limited to) IPAM address exhaustion, invalid
	// or unsupported address requests, or a named address not
	// being found.
	GatewayReasonAddressNotAssigned GatewayConditionReason = "AddressNotAssigned"
)

// ListenerStatus is the status associated with a Listener.
type ListenerStatus struct {
	// Port is the unique Listener port value for which this message is
	// reporting the status.
	Port PortNumber `json:"port"`

	// Protocol is the Listener protocol value for which this message is
	// reporting the status.
	Protocol ProtocolType `json:"protocol"`

	// Hostname is the Listener hostname value for which this message is
	// reporting the status.
	//
	// +optional
	Hostname *Hostname `json:"hostname,omitempty"`

	// Conditions describe the current condition of this listener.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions"`
}

// ListenerConditionType is a type of condition associated with the
// listener. This type should be used with the ListenerStatus.Conditions
// field.
type ListenerConditionType string

// ListenerConditionReason defines the set of reasons that explain
// why a particular Listener condition type has been raised.
type ListenerConditionReason string

const (
	// This condition indicates that the controller was unable to resolve
	// conflicting specification requirements for this Listener. If a
	// Listener is conflicted, its network port should not be configured
	// on any network elements.
	//
	// Possible reasons for this condition to be true are:
	//
	// * "HostnameConflict"
	// * "ProtocolConflict"
	// * "RouteConflict"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ListenerConditionConflicted ListenerConditionType = "Conflicted"

	// This reason is used with the "Conflicted" condition when
	// the Listener conflicts with hostnames in other Listeners. For
	// example, this reason would be used when multiple Listeners on
	// the same port use `*` in the hostname field.
	ListenerReasonHostnameConflict ListenerConditionReason = "HostnameConflict"

	// This reason is used with the "Conflicted" condition when
	// multiple Listeners are specified with the same Listener port
	// number, but have conflicting protocol specifications.
	ListenerReasonProtocolConflict ListenerConditionReason = "ProtocolConflict"

	// This reason is used with the "Conflicted" condition when the route
	// resources selected for this Listener conflict with other
	// specified properties of the Listener (e.g. Protocol).
	// For example, a Listener that specifies "UDP" as the protocol
	// but a route selector that resolves "TCPRoute" objects.
	ListenerReasonRouteConflict ListenerConditionReason = "RouteConflict"
)

const (
	// This condition indicates that, even though the listener is
	// syntactically and semantically valid, the controller is not able
	// to configure it on the underlying Gateway infrastructure.
	//
	// A Listener is specified as a logical requirement, but needs to be
	// configured on a network endpoint (i.e. address and port) by a
	// controller. The controller may be unable to attach the Listener
	// if it specifies an unsupported requirement, or prerequisite
	// resources are not available.
	//
	// Possible reasons for this condition to be true are:
	//
	// * "PortUnavailable"
	// * "UnsupportedExtension"
	// * "UnsupportedProtocol"
	// * "UnsupportedAddress"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ListenerConditionDetached ListenerConditionType = "Detached"

	// This reason is used with the "Detached" condition when the
	// Listener requests a port that cannot be used on the Gateway.
	ListenerReasonPortUnavailable ListenerConditionReason = "PortUnavailable"

	// This reason is used with the "Detached" condition when the
	// controller detects that an implementation-specific Listener
	// extension is being requested, but is not able to support
	// the extension.
	ListenerReasonUnsupportedExtension ListenerConditionReason = "UnsupportedExtension"

	// This reason is used with the "Detached" condition when the
	// Listener could not be attached to be Gateway because its
	// protocol type is not supported.
	ListenerReasonUnsupportedProtocol ListenerConditionReason = "UnsupportedProtocol"

	// This reason is used with the "Detached" condition when
	// the Listener could not be attached to the Gateway because the
	// requested address is not supported.
	ListenerReasonUnsupportedAddress ListenerConditionReason = "UnsupportedAddress"
)

const (
	// This condition indicates whether the controller was able to
	// resolve all the object references for the Listener.
	//
	// Possible reasons for this condition to be false are:
	//
	// * "DegradedRoutes"
	// * "InvalidCertificateRef"
	// * "InvalidRoutesRef"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ListenerConditionResolvedRefs ListenerConditionType = "ResolvedRefs"

	// This reason is used with the "ResolvedRefs" condition
	// when not all of the routes selected by this Listener could be
	// configured. The specific reason for the degraded route should
	// be indicated in the route's .Status.Conditions field.
	ListenerReasonDegradedRoutes ListenerConditionReason = "DegradedRoutes"

	// This reason is used with the "ResolvedRefs" condition when the
	// Listener has a TLS configuration with a TLS CertificateRef
	// that is invalid or cannot be resolved.
	ListenerReasonInvalidCertificateRef ListenerConditionReason = "InvalidCertificateRef"

	// This reason is used with the "ResolvedRefs" condition when
	// the Listener's Routes selector or kind is invalid or cannot
	// be resolved. Note that it is not an error for this selector to
	// not resolve any Routes, and the "ResolvedRefs" status condition
	// should not be raised in that case.
	ListenerReasonInvalidRoutesRef ListenerConditionReason = "InvalidRoutesRef"
)

const (
	// This condition indicates whether the Listener has been
	// configured on the Gateway.
	//
	// Possible reasons for this condition to be false are:
	//
	// * "Invalid"
	// * "Pending"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ListenerConditionReady ListenerConditionType = "Ready"

	// This reason is used with the "Ready" condition when the
	// Listener is syntactically or semantically invalid.
	ListenerReasonInvalid ListenerConditionReason = "Invalid"

	// This reason is used with the "Ready" condition when the
	// Listener is not yet not online and ready to accept client
	// traffic.
	ListenerReasonPending ListenerConditionReason = "Pending"
)
