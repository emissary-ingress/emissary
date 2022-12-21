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
)

type TraceSampling struct {
	Client  *int `json:"client,omitempty"`
	Random  *int `json:"random,omitempty"`
	Overall *int `json:"overall,omitempty"`
}

// +kubebuilder:validation:Enum=ENVOY;LIGHTSTEP;B3;TRACE_CONTEXT
type PropagationMode string

type TraceConfig struct {
	AccessTokenFile   string `json:"access_token_file,omitempty"`
	CollectorCluster  string `json:"collector_cluster,omitempty"`
	CollectorEndpoint string `json:"collector_endpoint,omitempty"`
	// +kubebuilder:validation:Enum=HTTP_JSON_V1;HTTP_JSON;HTTP_PROTO
	CollectorEndpointVersion string            `json:"collector_endpoint_version,omitempty"`
	CollectorHostname        string            `json:"collector_hostname,omitempty"`
	PropagationModes         []PropagationMode `json:"propagation_modes,omitempty"`
	TraceID128Bit            *bool             `json:"trace_id_128bit,omitempty"`
	SharedSpanContext        *bool             `json:"shared_span_context,omitempty"`
	ServiceName              string            `json:"service_name,omitempty"`
}

// TracingCustomTagTypeLiteral provides a data structure for capturing envoy's `type.tracing.v3.CustomTag.Literal`
type TracingCustomTagTypeLiteral struct {
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// TracingCustomTagTypeEnvironment provides a data structure for capturing envoy's `type.tracing.v3.CustomTag.Environment`
type TracingCustomTagTypeEnvironment struct {
	// +kubebuilder:validation:Required
	Name         string  `json:"name"`
	DefaultValue *string `json:"default_value,omitempty"`
}

// TracingCustomTagTypeRequestHeader provides a data structure for capturing envoy's `type.tracing.v3.CustomTag.Header`
type TracingCustomTagTypeRequestHeader struct {
	// +kubebuilder:validation:Required
	Name         string  `json:"name"`
	DefaultValue *string `json:"default_value,omitempty"`
}

// TracingCustomTag provides a data structure for capturing envoy's `type.tracing.v3.CustomTag`
type TracingCustomTag struct {
	// +kubebuilder:validation:Required
	Tag string `json:"tag"`

	// There is no oneOf support in kubebuilder https://github.com/kubernetes-sigs/controller-tools/issues/461

	// Literal explicitly specifies the protocol stack to set up. Exactly one of Literal,
	// Environment or Header must be supplied.
	Literal *TracingCustomTagTypeLiteral `json:"literal,omitempty"`
	// Environment explicitly specifies the protocol stack to set up. Exactly one of Literal,
	// Environment or Header must be supplied.
	Environment *TracingCustomTagTypeEnvironment `json:"environment,omitempty"`
	// Header explicitly specifies the protocol stack to set up. Exactly one of Literal,
	// Environment or Header must be supplied.
	Header *TracingCustomTagTypeRequestHeader `json:"request_header,omitempty"`
}

// TracingServiceSpec defines the desired state of TracingService
type TracingServiceSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Enum={"lightstep","zipkin","datadog","opentelemetry"}
	// +kubebuilder:validation:Required
	Driver string `json:"driver,omitempty"`
	// +kubebuilder:validation:Required
	Service  string         `json:"service,omitempty"`
	Sampling *TraceSampling `json:"sampling,omitempty"`
	// Deprecated: tag_headers is deprecated. Use custom_tags instead.
	// `tag_headers: ["header"]` can be defined as `custom_tags: [{"request_header": {"name": "header"}}]`.
	DeprecatedTagHeaders []string           `json:"tag_headers,omitempty"`
	CustomTags           []TracingCustomTag `json:"custom_tags,omitempty"`
	Config               *TraceConfig       `json:"config,omitempty"`
	StatsName            string             `json:"stats_name,omitempty"`
}

// TracingService is the Schema for the tracingservices API
//
// +kubebuilder:object:root=true
type TracingService struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TracingServiceSpec `json:"spec,omitempty"`
}

// TracingServiceList contains a list of TracingServices.
//
// +kubebuilder:object:root=true
type TracingServiceList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TracingService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TracingService{}, &TracingServiceList{})
}
