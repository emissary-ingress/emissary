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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ambv2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
)

// AmbassadorTCPMappingSpec defines the desired state of AmbassadorTCPMapping
type AmbassadorTCPMappingSpec struct {
	AmbassadorID ambv2.AmbassadorID `json:"ambassador_id,omitempty"`

	// Port isn't a pointer because it's required.
	// +kubebuilder:validation:Required
	Port    int    `json:"port,omitempty"`
	Host    string `json:"host,omitempty"`
	Address string `json:"address,omitempty"`
	// +kubebuilder:validation:Required
	Service         string                 `json:"service,omitempty"`
	EnableIPv4      *bool                  `json:"enable_ipv4,omitempty"`
	EnableIPv6      *bool                  `json:"enable_ipv6,omitempty"`
	CircuitBreakers []ambv2.CircuitBreaker `json:"circuit_breakers,omitempty"`

	// FIXME(lukeshu): Surely this should be an 'int'?
	IdleTimeoutMs string `json:"idle_timeout_ms,omitempty"`

	Resolver   string              `json:"resolver,omitempty"`
	TLS        *ambv2.BoolOrString `json:"tls,omitempty"`
	Weight     *int                `json:"weight,omitempty"`
	ClusterTag string              `json:"cluster_tag,omitempty"`
	StatsName  string              `json:"stats_name,omitempty"`
}

// AmbassadorTCPMapping is the Schema for the tcpmappings API
//
// +kubebuilder:object:root=true
type AmbassadorTCPMapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AmbassadorTCPMappingSpec `json:"spec,omitempty"`
}

// AmbassadorTCPMappingList contains a list of AmbassadorTCPMappings.
//
// +kubebuilder:object:root=true
type AmbassadorTCPMappingList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AmbassadorTCPMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AmbassadorTCPMapping{}, &AmbassadorTCPMappingList{})
}
