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

package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthServiceIncludeBody struct {
	// These aren't pointer types because they are required.
	// +kubebuilder:validation:Required
	MaxBytes int `json:"maxBytes,omitempty" v3:"max_bytes,omitempty"`

	// +kubebuilder:validation:Required
	AllowPartial bool `json:"allowPartial,omitempty" v3:"allow_partial,omitempty"`
}

type AuthServiceStatusOnError struct {
	Code int `json:"code,omitempty"`
}

// AuthServiceSpec defines the desired state of AuthService
type AuthServiceSpec struct {
	AmbassadorID AmbassadorID `json:"ambassadorID,omitempty" v3:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Required
	AuthService string `json:"authService,omitempty" v3:"auth_service,omitempty"`
	PathPrefix  string `json:"pathPrefix,omitempty" v3:"path_prefix,omitempty"`
	TLS         string `json:"tls,omitempty"`
	// +kubebuilder:validation:Enum={"http","grpc"}
	Proto                       string               `json:"proto,omitempty"`
	Timeout                     *MillisecondDuration `json:"timeoutMS,omitempty" v3:"timeout_ms,omitempty"`
	AllowedRequestHeaders       []string             `json:"allowedRequestHeaders,omitempty" v3:"allowed_request_headers,omitempty"`
	AllowedAuthorizationHeaders []string             `json:"allowedAuthorizationHeaders,omitempty" v3:"allowed_authorization_headers,omitempty"`
	AddAuthHeaders              map[string]string    `json:"addAuthHeaders,omitempty" v3:"add_auth_headers,omitempty"`

	AllowRequestBody  *bool                     `json:"allowRequestBody,omitempty" v3:"allow_request_body,omitempty"`
	AddLinkerdHeaders *bool                     `json:"addLinkerdHeaders,omitempty" v3:"add_linkerd_headers,omitempty"`
	FailureModeAllow  *bool                     `json:"failureModeAllow,omitempty" v3:"failure_mode_allow,omitempty"`
	IncludeBody       *AuthServiceIncludeBody   `json:"includeBody,omitempty" v3:"include_body,omitempty"`
	StatusOnError     *AuthServiceStatusOnError `json:"statusOnError,omitempty" v3:"status_on_error,omitempty"`

	// ProtocolVersion is the envoy api transport protocol version
	//
	// +kubebuilder:validation:Enum={"v2","v3"}
	ProtocolVersion string            `json:"protocolVersion,omitempty" v3:"protocol_version,omitempty"`
	StatsName       string            `json:"statsName,omitempty" v3:"stats_name,omitempty"`
	CircuitBreakers []*CircuitBreaker `json:"circuitBreakers,omitempty" v3:"circuit_breakers,omitempty"`

	V2ExplicitTLS *V2ExplicitTLS `json:"v2ExplicitTLS,omitempty"`
}

// AuthService is the Schema for the authservices API
//
// +kubebuilder:object:root=true
type AuthService struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthServiceSpec `json:"spec,omitempty"`
}

// AuthServiceList contains a list of AuthServices.
//
// +kubebuilder:object:root=true
type AuthServiceList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthService{}, &AuthServiceList{})
}
