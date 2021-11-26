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

package v3alpha1

import (
	"encoding/json"
	"time"
)

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
	ErrorResponseTextFormat string `json:"text_format,omitempty"`

	// A JSON response with content-type: application/json. The values can
	// contain format text like in text_format.
	ErrorResponseJsonFormat map[string]string `json:"json_format,omitempty"`

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

// AmbassadorID declares which Ambassador instances should pay
// attention to this resource. If no value is provided, the default is:
//
//    ambassador_id:
//    - "default"
//
// TODO(lukeshu): In v3alpha2, consider renaming all of the `ambassador_id` (singular) fields to
// `ambassador_ids` (plural).
type AmbassadorID []string

func (aid AmbassadorID) Matches(envVar string) bool {
	if len(aid) == 0 {
		aid = []string{"default"}
	}
	for _, item := range aid {
		if item == envVar {
			return true
		}
	}
	return false
}

// TODO(lukeshu): In v3alpha2, change all of the `{foo}_ms`/`MillisecondDuration` fields to
// `{foo}`/`metav1.Duration`.
//
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

func (d *MillisecondDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Milliseconds())
}

// UntypedDict is relatively opaque as a Go type, but it preserves its contents in a roundtrippable
// way.
// +kubebuilder:validation:Type="object"
// +kubebuilder:pruning:PreserveUnknownFields
type UntypedDict struct {
	Values map[string]UntypedValue `json:"-"`
}

func (u UntypedDict) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Values)
}

func (u *UntypedDict) UnmarshalJSON(data []byte) error {
	var values map[string]UntypedValue
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}
	*u = UntypedDict{Values: values}
	return nil
}

type UntypedValue struct {
	Raw json.RawMessage `json:"-"`
}

func (u UntypedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Raw)
}

func (u *UntypedValue) UnmarshalJSON(data []byte) error {
	*u = UntypedValue{Raw: json.RawMessage(data)}
	return nil
}
