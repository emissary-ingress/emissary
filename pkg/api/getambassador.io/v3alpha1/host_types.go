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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ambv2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
)

type ACMEProviderSpec struct {
	// Specifies who to talk ACME with to get certs. Defaults to Let's
	// Encrypt; if "none" (case-insensitive), do not try to do ACME for
	// this AmbassadorHost.
	Authority string `json:"authority,omitempty"`
	Email     string `json:"email,omitempty"`

	// Specifies the Kubernetes Secret to use to store the private key of the ACME
	// account (essentially, where to store the auto-generated password for the
	// auto-created ACME account).  You should not normally need to set this--the
	// default value is based on a combination of the ACME authority being registered
	// wit and the email address associated with the account.
	//
	// Note that this is a native-Kubernetes-style core.v1.LocalObjectReference, not
	// an Ambassador-style `{name}.{namespace}` string.  Because we're opinionated, it
	// does not support referencing a Secret in another namespace (because most native
	// Kubernetes resources don't support that), but if we ever abandon that opinion
	// and decide to support non-local references it, it would be by adding a
	// `namespace:` field by changing it from a core.v1.LocalObjectReference to a
	// core.v1.SecretReference, not by adopting the `{name}.{namespace}` notation.
	PrivateKeySecret *corev1.LocalObjectReference `json:"privateKeySecret,omitempty"`

	// This is normally set automatically
	Registration string `json:"registration,omitempty"`
}

type InsecureRequestPolicy struct {
	// +kubebuilder:validation:Enum={"Redirect","Reject","Route"}
	Action         string `json:"action,omitempty"`
	AdditionalPort *int   `json:"additionalPort,omitempty"`
}

type RequestPolicy struct {
	Insecure InsecureRequestPolicy `json:"insecure,omitempty"`

	// Later we may define a 'secure' section too.
}

type PreviewURLSpec struct {
	// Is the Preview URL feature enabled?
	Enabled *bool `json:"enabled,omitempty"`

	// What type of Preview URL is allowed?
	Type PreviewURLType `json:"type,omitempty"`
}

// What type of Preview URL is allowed?
//
//  - path
//  - wildcard
//  - datawire // FIXME rename this before release
//
// +kubebuilder:validation:Enum={"Path"}
type PreviewURLType string

// AmbassadorHostSpec defines the desired state of AmbassadorHost
type AmbassadorHostSpec struct {
	// Common to all Ambassador objects (and optional).
	AmbassadorID ambv2.AmbassadorID `json:"ambassador_id,omitempty"`

	// Hostname by which the Ambassador can be reached.
	Hostname string `json:"hostname,omitempty"`

	// Selector by which we can find further configuration.
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Specifies whether/who to talk ACME with to automatically manage the $tlsSecret.
	AcmeProvider *ambv2.ACMEProviderSpec `json:"acmeProvider,omitempty"`

	// Name of the Kubernetes secret into which to save generated
	// certificates.  If ACME is enabled (see $acmeProvider), then the
	// default is $hostname; otherwise the default is "".  If the value
	// is "", then we do not do TLS for this AmbassadorHost.
	//
	// Note that this is a native-Kubernetes-style core.v1.LocalObjectReference, not
	// an Ambassador-style `{name}.{namespace}` string.  Because we're opinionated, it
	// does not support referencing a Secret in another namespace (because most native
	// Kubernetes resources don't support that), but if we ever abandon that opinion
	// and decide to support non-local references it, it would be by adding a
	// `namespace:` field by changing it from a core.v1.LocalObjectReference to a
	// core.v1.SecretReference, not by adopting the `{name}.{namespace}` notation.
	TLSSecret *corev1.LocalObjectReference `json:"tlsSecret,omitempty"`

	// Request policy definition.
	RequestPolicy *RequestPolicy `json:"requestPolicy,omitempty"`

	// Configuration for the Preview URL feature of Service Preview. Defaults to preview URLs not enabled.
	PreviewUrl *PreviewURLSpec `json:"previewUrl,omitempty"`

	// Name of the TLSContext the AmbassadorHost resource is linked with.
	// It is not valid to specify both `tlsContext` and `tls`.
	//
	// Note that this is a native-Kubernetes-style core.v1.LocalObjectReference, not
	// an Ambassador-style `{name}.{namespace}` string.  Because we're opinionated, it
	// does not support referencing a Secret in another namespace (because most native
	// Kubernetes resources don't support that), but if we ever abandon that opinion
	// and decide to support non-local references it, it would be by adding a
	// `namespace:` field by changing it from a core.v1.LocalObjectReference to a
	// core.v1.SecretReference, not by adopting the `{name}.{namespace}` notation.
	TLSContext *corev1.LocalObjectReference `json:"tlsContext,omitempty"`

	// TLS configuration.  It is not valid to specify both
	// `tlsContext` and `tls`.
	TLS *ambv2.TLSConfig `json:"tls,omitempty"`
}

// The first value listed in the Enum marker becomes the "zero" value,
// and it would be great if "Pending" could be the default value; but
// it's Important that the "zero" value be able to be shown as
// empty/omitted from display, and we really do want `kubectl get
// hosts` to say "Pending" in the "STATE" column, and not leave the
// column empty.
//
// +kubebuilder:validation:Type=string
// +kubebuilder:validation:Enum={"Initial","Pending","Ready","Error"}
type AmbassadorHostState int

// +kubebuilder:validation:Type=string
// +kubebuilder:validation:Enum={"NA","DefaultsFilled","ACMEUserPrivateKeyCreated","ACMEUserRegistered","ACMECertificateChallenge"}
type AmbassadorHostPhase int

// AmbassadorHostStatus defines the observed state of AmbassadorHost
type AmbassadorHostStatus struct {
	TLSCertificateSource AmbassadorHostTLSCertificateSource `json:"tlsCertificateSource,omitempty"`

	State AmbassadorHostState `json:"state,omitempty"`

	// phaseCompleted and phasePending are valid when state==Pending or
	// state==Error.
	PhaseCompleted AmbassadorHostPhase `json:"phaseCompleted,omitempty"`
	// phaseCompleted and phasePending are valid when state==Pending or
	// state==Error.
	PhasePending AmbassadorHostPhase `json:"phasePending,omitempty"`

	// errorReason, errorTimestamp, and errorBackoff are valid when state==Error.
	ErrorReason    string           `json:"errorReason,omitempty"`
	ErrorTimestamp *metav1.Time     `json:"errorTimestamp,omitempty"`
	ErrorBackoff   *metav1.Duration `json:"errorBackoff,omitempty"`
}

// +kubebuilder:validation:Enum={"Unknown","None","Other","ACME"}
type AmbassadorHostTLSCertificateSource string

// AmbassadorHost is the Schema for the hosts API
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.spec.hostname`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Phase Completed",type=string,JSONPath=`.status.phaseCompleted`
// +kubebuilder:printcolumn:name="Phase Pending",type=string,JSONPath=`.status.phasePending`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AmbassadorHost struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *AmbassadorHostSpec  `json:"spec,omitempty"`
	Status AmbassadorHostStatus `json:"status,omitempty"`
}

// AmbassadorHostList contains a list of AmbassadorHosts.
//
// +kubebuilder:object:root=true
type AmbassadorHostList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AmbassadorHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AmbassadorHost{}, &AmbassadorHostList{})
}
