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

// TCPMappingSpec defines the desired state of TCPMapping
type TCPMappingSpec struct {
	AmbassadorID AmbassadorID `json:"ambassadorID,omitempty" v3:"ambassador_id,omitempty"`

	// Port isn't a pointer because it's required.
	// +kubebuilder:validation:Required
	Port    int    `json:"port,omitempty"`
	Host    string `json:"host,omitempty"`
	Address string `json:"address,omitempty"`
	// +kubebuilder:validation:Required
	Service         string           `json:"service,omitempty"`
	EnableIPv4      *bool            `json:"enableIPv4,omitempty" v3:"enable_ipv4,omitempty"`
	EnableIPv6      *bool            `json:"enableIPv6,omitempty" v3:"enable_ipv6,omitempty"`
	CircuitBreakers []CircuitBreaker `json:"circuitBreakers,omitempty" v3:"circuit_breakers,omitempty"`

	IdleTimeoutMs string `json:"idleTimeoutMS,omitempty" v3:"idle_timeout_ms,omitempty"`

	Resolver   string `json:"resolver,omitempty"`
	TLS        string `json:"tls,omitempty"`
	Weight     *int   `json:"weight,omitempty"`
	ClusterTag string `json:"clusterTag,omitempty" v3:"cluster_tag,omitempty"`
	StatsName  string `json:"statsName,omitempty" v3:"stats_name,omitempty"`

	V2ExplicitTLS *V2ExplicitTLS `json:"v2ExplicitTLS,omitempty"`
}

// TCPMapping is the Schema for the tcpmappings API
//
// +kubebuilder:object:root=true
type TCPMapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TCPMappingSpec `json:"spec,omitempty"`
}

// TCPMappingList contains a list of TCPMappings.
//
// +kubebuilder:object:root=true
type TCPMappingList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TCPMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TCPMapping{}, &TCPMappingList{})
}
