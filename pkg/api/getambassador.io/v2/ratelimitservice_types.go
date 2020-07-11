/*


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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RateLimitServiceSpec defines the desired state of RateLimitService
type RateLimitServiceSpec struct {
	// Common to all Ambassador objects.
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	Service   string `json:"service,omitempty"`
	TimeoutMs int32  `json:"timeout_ms,omitempty"`
	Domain    string `json:"domain,omitempty"`
	TLS       string `json:"tls,omitempty"`
}

// RateLimitServiceStatus defines the observed state of RateLimitService
type RateLimitServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// RateLimitService is the Schema for the ratelimitservices API
type RateLimitService struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RateLimitServiceSpec   `json:"spec,omitempty"`
	Status RateLimitServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RateLimitServiceList contains a list of RateLimitService
type RateLimitServiceList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimitService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RateLimitService{}, &RateLimitServiceList{})
}
