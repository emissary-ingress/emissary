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

// TCPMapping is the Schema for the tcpmappings API
//
// +kubebuilder:object:root=true
type TCPMapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ambv2.TCPMappingSpec `json:"spec,omitempty"`

	// dumbWorkaround is a dumb workaround for a bug in conversion-gen that it doesn't pay
	// attention to +k8s:conversion-fn=drop or +k8s:conversion-gen=false when checking if it can
	// do the direct-assignment or direct-conversion optimizations, and therefore might disobey
	// the +k8s:conversion-fn=drop on metav1.TypeMeta.
	//
	// +k8s:conversion-gen=false
	dumbWorkaround byte `json:"-"` //nolint:unused // dumb workaround
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
