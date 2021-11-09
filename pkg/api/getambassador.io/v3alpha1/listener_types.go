// Copyright 2020 Datawire.  All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

///////////////////////////////////////////////////////////////////////////
// Important: Run "make generate-fast" to regenerate code after modifying
// this file.
///////////////////////////////////////////////////////////////////////////

package v3alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ProtocolStackElement defines specific layers that may be combined in a protocol
// stack for processing connections to a port.
// +kubebuilder:validation:Enum=HTTP;PROXY;TLS;TCP;UDP
type ProtocolStackElement string

const (
	// HTTPProtocolStackElement represents the HTTP protocol.
	HTTPProtocolStackElement ProtocolStackElement = "HTTP"

	// PROXYProtocolStackElement represents the HAProxy PROXY protocol.
	PROXYProtocolStackElement ProtocolStackElement = "PROXY"

	// TLSProtocolStackElement represents the TLS protocol.
	TLSProtocolStackElement ProtocolStackElement = "TLS"

	// TCPProtocolStackElement represents raw TCP sessions.
	TCPProtocolStackElement ProtocolStackElement = "TCP"

	// UDPProtocolStackElement represents UDP packets.
	UDPProtocolStackElement ProtocolStackElement = "UDP"
)

// ProtocolType defines shorthands for well-known protocol stacks.
// +kubebuilder:validation:Enum=HTTP;HTTPS;HTTPPROXY;HTTPSPROXY;TCP;TLS;UDP
type ProtocolType string

const (
	// HTTPProtocolType accepts cleartext HTTP/1.1 sessions over TCP.
	// HTTP;TCP
	HTTPProtocolType ProtocolType = "HTTP"

	// HTTPSProtocolType accepts encrypted HTTP/1.1 or HTTP/2 sessions using TLS over TCP.
	// TLS;HTTP;TCP
	HTTPSProtocolType ProtocolType = "HTTPS"

	// HTTPPROXYProtocolType accepts cleartext HTTP/1.1 sessions using the HAProxy PROXY protocol over TCP.
	// PROXY;HTTP;TCP
	HTTPPROXYProtocolType ProtocolType = "HTTPPROXY"

	// HTTPSPROXYProtocolType accepts encrypted HTTP/1.1 or HTTP/2 sessions using the HAProxy PROXY protocol over TLS over TCP.
	// TLS;PROXY;HTTP;TCP
	HTTPSPROXYProtocolType ProtocolType = "HTTPSPROXY"

	// RAWTCPProtocolType accepts raw TCP sessions.
	// TCP
	RAWTCPProtocolType ProtocolType = "TCP"

	// TLSProtocolType accepts TLS over TCP.
	// TLS;TCP
	TLSProtocolType ProtocolType = "TLS"

	// UDPProtocolType accepts UDP packets.
	// UDP
	UDPProtocolType ProtocolType = "UDP"
)

// SecurityModelType defines the mechanisms we can use to determine whether connections to
// a port are secure or insecure.
// +kubebuilder:validation:Enum=XFP;SECURE;INSECURE
type SecurityModelType string

const (
	// XFPSecurityModelType specifies that connections on this port use X-Forwarded-Proto to
	// determine security: if the protocol is HTTPS, the connection is secure; otherwise
	// it is insecure.
	XFPSecurityModelType SecurityModelType = "XFP"

	// SECURESecurityModelType specifies that connections on this port are always secure
	SECURESecurityModelType SecurityModelType = "SECURE"

	// INSECURESecurityModelType specifies that connections on this port are never secure
	INSECURESecurityModelType SecurityModelType = "INSECURE"
)

// NamespaceFromType defines how we evaluate a NamespaceBindingType.
// +kubebuilder:validation:Enum=SELF;ALL;SELECTOR
type NamespaceFromType string

const (
	// SELFNamespaceFromType specifies that an Listener should consider Hosts only in the
	// Listener's namespaces.
	SELFNamespaceFromType NamespaceFromType = "SELF"

	// ALLNamespaceFromType specifies that an Listener should consider Hosts in ALL
	// namespaces. This is the simplest way to build an Listener that matches all Hosts.
	ALLNamespaceFromType NamespaceFromType = "ALL"

	// XXX We can't support from=SELECTOR until we're doing Listener handling in
	// XXX Golang: the Python side of Emissary doesn't have access to namespace selectors.
	//
	// // SELECTORNamespaceFromType specifies to use the NamespaceBinding selector to
	// // determine namespaces to consider for Hosts.
	// SELECTORNamespaceFromType NamespaceFromType = "SELECTOR"
)

// NamespaceBindingType defines we we specify which namespaces to look for Hosts in.
type NamespaceBindingType struct {
	From NamespaceFromType `json:"from,omitempty"`

	// XXX We can't support from=SELECTOR until we're doing Listener handling in
	// XXX Golang: the Python side of Emissary doesn't have access to namespace selectors.
	//
	// Selector *metav1.LabelSelector `json:"hostSelector,omitempty"`
}

// HostBindingType defines how we specify Hosts to bind to this Listener.
type HostBindingType struct {
	Namespace NamespaceBindingType  `json:"namespace"`
	Selector  *metav1.LabelSelector `json:"selector,omitempty"`
}

// ListenerSpec defines the desired state of this Port
type ListenerSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	// Port is the network port. Only one Listener can use a given port.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`

	// Protocol is a shorthand for certain predefined stacks. Exactly one of Protocol
	// or ProtocolStack must be supplied.
	Protocol ProtocolType `json:"protocol,omitempty"`

	// ProtocolStack explicitly specifies the protocol stack to set up. Exactly one of Protocol
	// or ProtocolStack must be supplied.
	ProtocolStack []ProtocolStackElement `json:"protocolStack,omitempty"`

	// SecurityModel specifies how to determine whether connections to this port are secure
	// or insecure.
	// +kubebuilder:validation:Required
	SecurityModel SecurityModelType `json:"securityModel"`

	// StatsPrefix specifies the prefix for statistics sent by Envoy about this
	// Listener. The default depends on the protocol: "ingress-http",
	// "ingress-https", "ingress-tls-$port", or "ingress-$port".
	StatsPrefix string `json:"statsPrefix,omitempty"`

	// L7Depth specifies how many layer 7 load balancers are between us and the edge of
	// the network.
	L7Depth int32 `json:"l7Depth,omitempty"`

	// HostBinding allows restricting which Hosts will be used for this Listener.
	// +kubebuilder:validation:Required
	HostBinding HostBindingType `json:"hostBinding"`
}

// Listener is the Schema for the hosts API
//
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Port",type=string,JSONPath=`.spec.port`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Stack",type=string,JSONPath=`.spec.protocolStack`
// +kubebuilder:printcolumn:name="StatsPrefix",type=string,JSONPath=`.spec.statsPrefix`
// +kubebuilder:printcolumn:name="Security",type=string,JSONPath=`.spec.securityModel`
// +kubebuilder:printcolumn:name="L7Depth",type=string,JSONPath=`.spec.l7Depth`
// +kubebuilder:storageversion
type Listener struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec *ListenerSpec `json:"spec,omitempty"`
}
func (*Listener) Hub() {}

// ListenerList contains a list of Listener.
//
// +kubebuilder:object:root=true
type ListenerList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Listener `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Listener{}, &ListenerList{})
}
