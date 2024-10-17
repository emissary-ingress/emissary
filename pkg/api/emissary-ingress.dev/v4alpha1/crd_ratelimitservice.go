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

// RateLimitServiceSpec defines the desired state of RateLimitService
type RateLimitServiceSpec struct {
	// Common to all Ambassador objects.
	AmbassadorID AmbassadorID `json:"ambassadorID,omitempty" v3:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Required
	Service string               `json:"service,omitempty"`
	Timeout *MillisecondDuration `json:"timeoutMS,omitempty" v3:"timeout_ms,omitempty"`
	Domain  string               `json:"domain,omitempty"`
	TLS     string               `json:"tls,omitempty"`

	// ProtocolVersion is the envoy api transport protocol version
	//
	// +kubebuilder:validation:Enum={"v2","v3"}
	ProtocolVersion string `json:"protocolVersion,omitempty" v3:"protocol_version,omitempty"`
	StatsName       string `json:"statsName,omitempty" v3:"stats_name,omitempty"`

	// FailureModeDeny when set to true, envoy will deny traffic if it
	// is unable to communicate with the rate limit service.
	FailureModeDeny bool `json:"failureModeDeny,omitempty" v3:"failure_mode_deny,omitempty"`

	GRPC *RateLimitGRPCConfig `json:"grpc,omitempty"`

	V2ExplicitTLS *V2ExplicitTLS `json:"v2ExplicitTLS,omitempty"`
}

type RateLimitGRPCConfig struct {
	// UseResourceExhaustedCode, when set to true, will cause envoy
	// to return a `RESOURCE_EXHAUSTED` gRPC code instead of the default
	// `UNAVAILABLE` gRPC code.
	UseResourceExhaustedCode bool `json:"useResourceExhaustedCode,omitempty" v3:"use_resource_exhausted_code,omitempty"`
}

// RateLimitService is the Schema for the ratelimitservices API
//
// +kubebuilder:object:root=true
type RateLimitService struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RateLimitServiceSpec `json:"spec,omitempty"`
}

// RateLimitServiceList contains a list of RateLimitServices.
//
// +kubebuilder:object:root=true
type RateLimitServiceList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimitService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RateLimitService{}, &RateLimitServiceList{})
}
