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
// Important: Run "make update-yaml" to regenerate code after modifying
// this file.
///////////////////////////////////////////////////////////////////////////

package v2

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MappingSpec defines the desired state of Mapping
type MappingSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	Prefix                string            `json:"prefix,omitempty"`
	PrefixRegex           bool              `json:"prefix_regex,omitempty"`
	PrefixExact           bool              `json:"prefix_exact,omitempty"`
	Service               string            `json:"service,omitempty"`
	AddRequestHeaders     []string          `json:"add_request_headers,omitempty"`
	AddResponseHeaders    []string          `json:"add_response_headers,omitempty"`
	AddLinkerdHeaders     bool              `json:"add_linkerd_headers,omitempty"`
	AutoHostRewrite       bool              `json:"auto_host_rewrite,omitempty"`
	CaseSensitive         bool              `json:"case_sensitive,omitempty"`
	EnableIPv4            bool              `json:"enable_ipv4,omitempty"`
	EnableIPv6            bool              `json:"enable_ipv6,omitempty"`
	CircuitBreakers       []*CircuitBreaker `json:"circuit_breakers,omitempty"`
	KeepAlive             *KeepAlive        `json:"keepalive,omitempty"`
	CORS                  *CORS             `json:"cors,omitempty"`
	RetryPolicy           *RetryPolicy      `json:"retry_policy,omitempty"`
	GRPC                  bool              `json:"grpc,omitempty"`
	HostRedirect          bool              `json:"host_redirect,omitempty"`
	HostRewrite           string            `json:"host_rewrite,omitempty"`
	Method                string            `json:"method,omitempty"`
	MethodRegex           bool              `json:"method_regex,omitempty"`
	OutlierDetection      string            `json:"outlier_detection,omitempty"`
	PathRedirect          string            `json:"path_redirect,omitempty"`
	Priority              string            `json:"priority,omitempty"`
	Precedence            int32             `json:"precedence,omitempty"`
	RemoveRequestHeaders  []string          `json:"remove_request_headers,omitempty"`
	RemoveResponseHeaders []string          `json:"remove_response_headers,omitempty"`
	Resolver              string            `json:"resolver,omitempty"`
	Rewrite               string            `json:"rewrite,omitempty"`
	RegexRewrite          bool              `json:"regex_rewrite,omitempty"`
	Shadow                bool              `json:"shadow,omitempty"`
	ConnectTimeoutMs      int32             `json:"connect_timeout_ms,omitempty"`
	ClusterIdleTimeoutMs  int32             `json:"cluster_idle_timeout_ms,omitempty"`
	TimeoutMs             int32             `json:"timeout_ms,omitempty"`
	IdleTimeoutMs         int32             `json:"idle_timeout_ms,omitempty"`
	TLS                   string            `json:"tls,omitempty"`
	UseWebsocket          bool              `json:"use_websocket,omitempty"`
	Weight                int32             `json:"weight,omitempty"`
	BypassAuth            bool              `json:"bypass_auth,omitempty"`
	Host                  string            `json:"host,omitempty"`
	HostRegex             bool              `json:"host_regex,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	RegexHeaders          map[string]string `json:"regex_headers,omitempty"`
	Labels                MappingLabels     `json:"labels,omitempty"`
	LoadBalancer          *LoadBalancer     `json:"load_balancer,omitempty"`
}

// Python: MappingLabels = Dict[str, Union[str,'MappingLabels']]
type MappingLabels map[string]StringOrMappingLabels

// StringOrMapping labels is the `Union[str,'MappingLabels']` part of
// the MappingLabels type.
//
// See the remarks about schema on custom types in `./common.go`.
//
// +kubebuilder:validation:Type=""
type StringOrMappingLabels struct {
	String *string
	Labels MappingLabels
}

// MarshalJSON is important both so that we generate the proper
// output, and to trigger controller-gen to not try to generate
// jsonschema for our sub-fields:
// https://github.com/kubernetes-sigs/controller-tools/pull/427
func (o StringOrMappingLabels) MarshalJSON() ([]byte, error) {
	switch {
	case o.String == nil && o.Labels == nil:
		return json.Marshal(nil)
	case o.String == nil && o.Labels != nil:
		return json.Marshal(o.Labels)
	case o.String != nil && o.Labels == nil:
		return json.Marshal(o.String)
	case o.String != nil && o.Labels != nil:
		panic("invalid StringOrMappingLabels")
	}
	panic("not reached")
}

func (o *StringOrMappingLabels) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = StringOrMappingLabels{}
		return nil
	}

	var err error

	var labels MappingLabels
	if err = json.Unmarshal(data, &labels); err == nil {
		*o = StringOrMappingLabels{Labels: labels}
		return nil
	}

	var str string
	if err = json.Unmarshal(data, &str); err == nil {
		*o = StringOrMappingLabels{String: &str}
		return nil
	}

	return err
}

// MappingStatus defines the observed state of Mapping
type MappingStatus struct {
}

// Mapping is the Schema for the mappings API
//
// +kubebuilder:object:root=true
type Mapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MappingSpec   `json:"spec,omitempty"`
	Status MappingStatus `json:"status,omitempty"`
}

// MappingList contains a list of Mappings.
//
// +kubebuilder:object:root=true
type MappingList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mapping{}, &MappingList{})
}
