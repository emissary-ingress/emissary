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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ambv2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
)

// KubernetesServiceResolver is the Schema for the kubernetesserviceresolver API
//
// +kubebuilder:object:root=true
type KubernetesServiceResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ambv2.KubernetesServiceResolverSpec `json:"spec,omitempty"`

	// dumbWorkaround is a dumb workaround for a bug in conversion-gen that it doesn't pay
	// attention to +k8s:conversion-fn=drop or +k8s:conversion-gen=false when checking if it can
	// do the direct-assignment or direct-conversion optimizations, and therefore might disobey
	// the +k8s:conversion-fn=drop on metav1.TypeMeta.
	//
	// +k8s:conversion-gen=false
	dumbWorkaround byte `json:"-"` //nolint:unused // dumb workaround
}

// KubernetesServiceResolverList contains a list of KubernetesServiceResolvers.
//
// +kubebuilder:object:root=true
type KubernetesServiceResolverList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesServiceResolver `json:"items"`
}

// KubernetesEndpointResolver is the Schema for the kubernetesendpointresolver API
//
// +kubebuilder:object:root=true
type KubernetesEndpointResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ambv2.KubernetesEndpointResolverSpec `json:"spec,omitempty"`

	// dumbWorkaround is a dumb workaround for a bug in conversion-gen that it doesn't pay
	// attention to +k8s:conversion-fn=drop or +k8s:conversion-gen=false when checking if it can
	// do the direct-assignment or direct-conversion optimizations, and therefore might disobey
	// the +k8s:conversion-fn=drop on metav1.TypeMeta.
	//
	// +k8s:conversion-gen=false
	dumbWorkaround byte `json:"-"` //nolint:unused // dumb workaround
}

// KubernetesEndpointResolverList contains a list of KubernetesEndpointResolvers.
//
// +kubebuilder:object:root=true
type KubernetesEndpointResolverList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesEndpointResolver `json:"items"`
}

// ConsulResolver is the Schema for the ConsulResolver API
//
// +kubebuilder:object:root=true
type ConsulResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ambv2.ConsulResolverSpec `json:"spec,omitempty"`

	// dumbWorkaround is a dumb workaround for a bug in conversion-gen that it doesn't pay
	// attention to +k8s:conversion-fn=drop or +k8s:conversion-gen=false when checking if it can
	// do the direct-assignment or direct-conversion optimizations, and therefore might disobey
	// the +k8s:conversion-fn=drop on metav1.TypeMeta.
	//
	// +k8s:conversion-gen=false
	dumbWorkaround byte `json:"-"` //nolint:unused // dumb workaround
}

// ConsulResolverList contains a list of ConsulResolvers.
//
// +kubebuilder:object:root=true
type ConsulResolverList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConsulResolver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubernetesServiceResolver{}, &KubernetesServiceResolverList{})
	SchemeBuilder.Register(&KubernetesEndpointResolver{}, &KubernetesEndpointResolverList{})
	SchemeBuilder.Register(&ConsulResolver{}, &ConsulResolverList{})
}
