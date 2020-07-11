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

// NOTE: json tags are required.  Any new fields you add must have json tags
// for the fields to be serialized.

// KubernetesServiceResolver tells Ambassador to use Kubernetes Service
// resources to resolve services. It actually has no spec other than the
// AmbassadorID.

type KubernetesServiceResolverSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`
}

// KubernetesServiceResolverStatus defines the observed state of KubernetesServiceResolver
type KubernetesServiceResolverStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// KubernetesServiceResolver is the Schema for the kubernetesserviceresolver API
type KubernetesServiceResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesServiceResolverSpec   `json:"spec,omitempty"`
	Status KubernetesServiceResolverStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KubernetesServiceResolverList contains a list of KubernetesServiceResolver
type KubernetesServiceResolverList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesServiceResolver `json:"items"`
}

// KubernetesEndpointResolver tells Ambassador to use Kubernetes Endpoints
// resources to resolve services. It actually has no spec other than the
// AmbassadorID.

type KubernetesEndpointResolverSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`
}

// KubernetesEndpointResolverStatus defines the observed state of KubernetesEndpointResolver
type KubernetesEndpointResolverStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// KubernetesEndpointResolver is the Schema for the kubernetesendpointresolver API
type KubernetesEndpointResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesEndpointResolverSpec   `json:"spec,omitempty"`
	Status KubernetesEndpointResolverStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KubernetesEndpointResolverList contains a list of KubernetesEndpointResolver
type KubernetesEndpointResolverList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesEndpointResolver `json:"items"`
}

// ConsulResolver tells Ambassador to use Consul to resolve services. In addition
// to the AmbassadorID, it needs information about which Consul server and DC to
// use.

type ConsulResolverSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	Address    string `json:"address,omitempty"`
	Datacenter string `json:"datacenter,omitempty"`
}

// ConsulResolverStatus defines the observed state of ConsulResolver
type ConsulResolverStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// ConsulResolver is the Schema for the ConsulResolver API
type ConsulResolver struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConsulResolverSpec   `json:"spec,omitempty"`
	Status ConsulResolverStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConsulResolverList contains a list of ConsulResolver
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
