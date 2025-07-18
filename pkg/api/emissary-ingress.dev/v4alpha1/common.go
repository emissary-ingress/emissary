// -*- fill-column: 75 -*-

// Copyright 2020 Datawire.  All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.  You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file deals with common things that are shared between multiple
// CRDs, but are ultimately used by individual CRDs (rather than by the
// apiVersion as a whole).

package v4alpha1

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
)

func BuildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	runtimeutil.Must(AddToScheme(scheme))
	return scheme
}

// V2ExplicitTLS controls some vanity/stylistic elements when converting
// from v3alpha1 to v2.  The values in an V2ExplicitTLS should not in any
// way affect the runtime operation of Emissary; except that it may affect
// internal names in the Envoy config, which may in turn affect stats
// names.  But it should not affect any end-user observable behavior.
type V2ExplicitTLS struct {
	// TLS controls whether and how to represent the "tls" field when
	// its value could be implied by the "service" field.  In v2, there
	// were a lot of different ways to spell an "empty" value, and this
	// field specifies which way to spell it (and will therefore only
	// be used if the value will indeed be empty).
	//
	//  | Value        | Representation                        | Meaning of representation          |
	//  |--------------+---------------------------------------+------------------------------------|
	//  | ""           | omit the field                        | defer to service (no TLSContext)   |
	//  | "null"       | store an explicit "null" in the field | defer to service (no TLSContext)   |
	//  | "string"     | store an empty string in the field    | defer to service (no TLSContext)   |
	//  | "bool:false" | store a Boolean "false" in the field  | defer to service (no TLSContext)   |
	//  | "bool:true"  | store a Boolean "true" in the field   | originate TLS (no TLSContext)      |
	//
	// If the meaning of the representation contradicts anything else
	// (if a TLSContext is to be used, or in the case of "bool:true" if
	// TLS is not to be originated), then this field is ignored.
	//
	// +kubebuilder:validation:Enum={"","null","bool:true","bool:false","string"}
	TLS string `json:"tls,omitempty"`

	// ServiceScheme specifies how to spell and capitalize the scheme-part of the
	// service URL.
	//
	// Acceptable values are "http://" (case-insensitive), "https://"
	// (case-insensitive), or "".  The value is used if it agrees with
	// whether or not this resource enables TLS origination, or if
	// something else in the resource overrides the scheme.
	//
	// +kubebuilder:validation:Pattern="^([hH][tT][tT][pP][sS]?://)?$"
	ServiceScheme *string `json:"serviceScheme,omitempty"`
}

type CircuitBreaker struct {
	// +kubebuilder:validation:Enum={"default", "high"}
	Priority           string `json:"priority,omitempty"`
	MaxConnections     *int   `json:"max_connections,omitempty"`
	MaxPendingRequests *int   `json:"max_pending_requests,omitempty"`
	MaxRequests        *int   `json:"max_requests,omitempty"`
	MaxRetries         *int   `json:"max_retries,omitempty"`
}

// ErrorResponseTextFormatSource specifies a source for an error response body
type ErrorResponseTextFormatSource struct {
	// The name of a file on the Ambassador pod that contains a format text string.
	Filename string `json:"filename"`
}

// ErorrResponseOverrideBody specifies the body of an error response
type ErrorResponseOverrideBody struct {
	// A format string representing a text response body.
	// Content-Type can be set using the `content_type` field below.
	ErrorResponseTextFormat *string `json:"text_format,omitempty"`

	// A JSON response with content-type: application/json. The values can
	// contain format text like in text_format.
	ErrorResponseJsonFormat *map[string]string `json:"json_format,omitempty"`

	// A format string sourced from a file on the Ambassador container.
	// Useful for larger response bodies that should not be placed inline
	// in configuration.
	ErrorResponseTextFormatSource *ErrorResponseTextFormatSource `json:"text_format_source,omitempty"`

	// The content type to set on the error response body when
	// using text_format or text_format_source. Defaults to 'text/plain'.
	ContentType string `json:"content_type,omitempty"`
}

// A response rewrite for an HTTP error response
type ErrorResponseOverride struct {
	// The status code to match on -- not a pointer because it's required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=400
	// +kubebuilder:validation:Maximum=599
	OnStatusCode int `json:"on_status_code,omitempty"`

	// The new response body
	// +kubebuilder:validation:Required
	Body ErrorResponseOverrideBody `json:"body,omitempty"`
}

// A range of response statuses from Start to End inclusive
type StatusRange struct {
	// Start of the statuses to include. Must be between 100 and 599 (inclusive)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	Min int `json:"min,omitempty"`
	// End of the statuses to include. Must be between 100 and 599 (inclusive)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	Max int `json:"max,omitempty"`
}

// HealthCheck specifies settings for performing active health checking on upstreams
type HealthCheck struct {
	// Timeout for connecting to the health checking endpoint. Defaults to 3 seconds.
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// Interval between health checks. Defaults to every 5 seconds.
	Interval *metav1.Duration `json:"interval,omitempty"`
	// Number of non-expected responses for the upstream to be considered unhealthy. A single 503 will mark the upstream as unhealthy regardless of the threshold. Defaults to 2.
	UnhealthyThreshold *int `json:"unhealthy_threshold,omitempty"`
	// Number of expected responses for the upstream to be considered healthy. Defaults to 1.
	HealthyThreshold *int `json:"healthy_threshold,omitempty"`

	// Configuration for where the healthcheck request should be made to
	// +kubebuilder:validation:Required
	HealthCheckLocation HealthCheckLocation `json:"health_check,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type HealthCheckLocation struct {
	HTTPHealthCheck *HTTPHealthCheck `json:"http,omitempty"`
	GRPCHealthCheck *GRPCHealthCheck `json:"grpc,omitempty"`
}

// HealthCheck for HTTP upstreams. Only one of http_health_check or grpc_health_check may be specified
type HTTPHealthCheck struct {
	// +kubebuilder:validation:Required
	Path                 string                 `json:"path,omitempty"`
	Host                 string                 `json:"hostname,omitempty"`
	AddRequestHeaders    map[string]AddedHeader `json:"add_request_headers,omitempty"`
	RemoveRequestHeaders []string               `json:"remove_request_headers,omitempty"`
	ExpectedStatuses     []StatusRange          `json:"expected_statuses,omitempty"`
}

// HealthCheck for gRPC upstreams. Only one of grpc_health_check or http_health_check may be specified
type GRPCHealthCheck struct {
	// The upstream name parameter which will be sent to gRPC service in the health check message
	// +kubebuilder:validation:Required
	UpstreamName string `json:"upstream_name,omitempty"`
	// The value of the :authority header in the gRPC health check request. If left empty the upstream name will be used.
	Authority string `json:"authority,omitempty"`
}

// AmbassadorID declares which Ambassador instances should pay
// attention to this resource. If no value is provided, the default is:
//
//	ambassador_id:
//	- "default"
type AmbassadorID []string

func (aid AmbassadorID) Matches(envVar string) bool {
	if len(aid) == 0 {
		aid = []string{"default"}
	}
	for _, item := range aid {
		if item == envVar {
			return true
		}
		if item == "_automatic_" {
			// We always pay attention to the "_automatic_" ID -- it gives us a to
			// easily always include certain configuration resources for Edge Stack.
			return true
		}
	}
	return false
}

// +kubebuilder:validation:Type="integer"
type MillisecondDuration struct {
	time.Duration `json:"-"`
}

func (d *MillisecondDuration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Duration = 0
		return nil
	}

	var intval int64
	if err := json.Unmarshal(data, &intval); err != nil {
		return err
	}
	d.Duration = time.Duration(intval) * time.Millisecond
	return nil
}

func (d MillisecondDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Milliseconds())
}

// +kubebuilder:validation:Type="integer"
type SecondDuration struct {
	time.Duration `json:"-"`
}

func (d *SecondDuration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Duration = 0
		return nil
	}

	var intval int64
	if err := json.Unmarshal(data, &intval); err != nil {
		return err
	}
	d.Duration = time.Duration(intval) * time.Second
	return nil
}

func (d SecondDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(d.Seconds()))
}

// UntypedDict is relatively opaque as a Go type, but it preserves its
// contents in a roundtrippable way.
//
// +kubebuilder:validation:Type="object"
// +kubebuilder:pruning:PreserveUnknownFields
type UntypedDict struct {
	// We have to hide this from controller-gen inside of a struct
	// (instead of just `type UntypedDict map[string]json.RawMessage`)
	// so that controller-gen doesn't generate an `items` field in the
	// schema.
	Values map[string]json.RawMessage `json:"-"`
}

func (u UntypedDict) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Values)
}

func (u *UntypedDict) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u.Values)
}
