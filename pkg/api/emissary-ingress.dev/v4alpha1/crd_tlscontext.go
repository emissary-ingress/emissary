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

// TLSContextSpec defines the desired state of TLSContext
type TLSContextSpec struct {
	AmbassadorID AmbassadorID `json:"ambassadorID,omitempty" v3:"ambassador_id,omitempty"`

	Hosts           []string `json:"hosts,omitempty"`
	Secret          string   `json:"secret,omitempty"`
	CertChainFile   string   `json:"certChainFile,omitempty" v3:"cert_chain_file,omitempty"`
	PrivateKeyFile  string   `json:"privateKeyFile,omitempty" v3:"private_key_file,omitempty"`
	CASecret        string   `json:"caSecret,omitempty" v3:"ca_secret,omitempty"`
	CACertChainFile string   `json:"cacertChainFile,omitempty" v3:"cacert_chain_file,omitempty"`
	CRLSecret       string   `json:"crlSecret,omitempty" v3:"crl_secret,omitempty"`
	ALPNProtocols   string   `json:"alpnProtocols,omitempty" v3:"alpn_protocols,omitempty"`
	CertRequired    *bool    `json:"certRequired,omitempty" v3:"cert_required,omitempty"`
	// +kubebuilder:validation:Enum={"v1.0", "v1.1", "v1.2", "v1.3"}
	MinTLSVersion string `json:"minTLSVersion,omitempty" v3:"min_tls_version,omitempty"`
	// +kubebuilder:validation:Enum={"v1.0", "v1.1", "v1.2", "v1.3"}
	MaxTLSVersion         string   `json:"maxTLSVersion,omitempty" v3:"max_tls_version,omitempty"`
	CipherSuites          []string `json:"cipherSuites,omitempty" v3:"cipher_suites,omitempty"`
	ECDHCurves            []string `json:"ecdhCurves,omitempty" v3:"ecdh_curves,omitempty"`
	SecretNamespacing     *bool    `json:"secretNamespacing,omitempty" v3:"secret_namespacing,omitempty"`
	RedirectCleartextFrom *int     `json:"redirectCleartextFrom,omitempty" v3:"redirect_cleartext_from,omitempty"`
	SNI                   string   `json:"sni,omitempty"`
}

// TLSContext is the Schema for the tlscontexts API
//
// +kubebuilder:object:root=true
type TLSContext struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TLSContextSpec `json:"spec,omitempty"`
}

// TLSContextList contains a list of TLSContexts.
//
// +kubebuilder:object:root=true
type TLSContextList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TLSContext `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TLSContext{}, &TLSContextList{})
}
