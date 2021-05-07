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
// Important: Run "make update-yaml" to regenerate code after modifying
// this file.
///////////////////////////////////////////////////////////////////////////

package v2

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
// +kubebuilder:validation:Enum=SECURE;INSECURE;XFP;WIRE
type SecurityModelType string

const (
	// SECURESecurityModelType specifies that connections on this port are always secure
	SECURESecurityModelType SecurityModelType = "SECURE"

	// INSECURESecurityModelType specifies that connections on this port are never secure
	INSECURESecurityModelType SecurityModelType = "INSECURE"

	// XFPSecurityModelType specifies that connections on this port use X-Forwarded-Proto to
	// determine security: if the protocol is HTTPS, the connection is secure; otherwise
	// it is insecure.
	XFPSecurityModelType SecurityModelType = "XFP"

	// WIRESecurityModelType specifies that connections on this port use the actual wire
	// protocol to determine security: if the protocol is HTTPS, the connection is secure;
	// otherwise it is insecure.
	WIRESecurityModelType SecurityModelType = "WIRE"
)

// ListenerSpec defines the desired state of this Port
type ListenerSpec struct {
	// Port is the network port. Only one Listener can use a given port.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Protocol is a shorthand for certain predefined stacks. Exactly one of Protocol
	// or ProtocolStack must be supplied.
	Protocol ProtocolType `json:"protocol,omitempty"`

	// ProtocolStack explicitly specifies the protocol stack to set up. Exactly one of Protocol
	// or ProtocolStack must be supplied.
	ProtocolStack []ProtocolStackElement `json:"protocolStack,omitempty"`

	// SecurityModel specifies how to determine whether connections to this port are secure
	// or insecure.
	SecurityModel SecurityModelType `json:"securityModel"`

	// L7Depth specifies how many layer 7 load balancers are between us and the edge of
	// the network.
	L7Depth int32 `json:"l7Depth"`

	// HostSelector allows restricting which Hosts will be used for this Listener.
	HostSelector *metav1.LabelSelector `json:"hostSelector"`
}

// Listener is the Schema for the hosts API
//
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Port",type=string,JSONPath=`.spec.port`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Stack",type=string,JSONPath=`.spec.protocolStack`
// +kubebuilder:printcolumn:name="Security",type=string,JSONPath=`.spec.securityModel`
// +kubebuilder:printcolumn:name="L7Depth",type=string,JSONPath=`.spec.l7Depth`
type Listener struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec *ListenerSpec `json:"spec,omitempty"`
}

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
