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

// I'm not sure where a better place to put this is, so I'm putting it here:
//
// # API design guidelines
//
// Ambassador's API has inconsistencies because it has historical
// baggage.  Not all of Ambassador's existing API (or even most of
// it!?) follow these guidelines, but new additions to the API should.
// If/when we advance to getambassador.io/v3 and we can break
// compatibility, these are things that we should apply everywhere.
//
// - Prefer `camelCase` to `snake_case`
//   * Exception: Except for consistency with existing fields in the
//     same resource, or symmetry with identical fields in another
//     resource.
//   * Justification: Kubernetes style is to use camelCase. But
//     historically Ambassador used snake_case for everything.
//
// - Give _every_ field a `json:""` struct tag.
//   * Justification: Marshaling and unmarshaling are key to what we
//     do, and it's critical to carefully define how it happens.
//   * Notes: This is not optional. Do it for _every field_. (It's OK
//     if the tag is literally `json:""` for fields that must never be
//     exposed during marshaling.)
//
// - Prefer `*int`, and `*bool`; rather than just `int`, `bool`.
//   * Justification: The Ambassador API is rooted in Python, where it
//     is always possible to tell if a given element was present in in
//     a CRD, or left unset.  This is at odds with Go's `omitempty`
//     specifier, which really means "omit if empty _or if set to the
//     default (zero) value_".  For int in particular, this results in
//     a value of 0 being omitted, and for many Ambassador fields, 0
//     is not the correct default value.
//
//     This resulted in a lot of bugs in the 1.10 timeframe, so be
//     careful going forward.
//
// - Prefer for object references to not support namespacing
//   * Exception: If there's a real use-case for it.
//   * Justification: Most native Kubernetes resources don't support
//     referencing things in a different namespace.  We should be
//     opinionated and not support it either, unless there's a good
//     reason to in a specific case.
//
// - Prefer to use `corev1.LocalObjectReference` or
//   `corev1.SecretReference` references instead of
//   `{name}.{namespace}` strings.
//   * Justification: The `{name}.{namespace}` thing evolved "an
//     opaque DNS name" in the `service` field of Mappings, and that
//     was generalized to other things.  Outside of the context of
//     "this is usable as a DNS name to make a request to", it's just
//     confusing and introduces needless ambiguity.  Nothing other
//     than Ambassador uses that notation.
//   * Notes: For things that don't support cross-namespace references
//     (see above), use LocalObjectReference; if you really must
//     support cross-namespace references, then use SecretReference.
//
// - Prefer to use `metav1.Duration` fields instead of "_s" or "_ms"
//   numeric fields.
//
// - Don't have Ambassador populate anything in the `.spec` or
//   `.metadata` of something a user might edit, only let Ambassador
//   set things in the `.status`.
//   * Exception: If Ambassador 100% owns the resource and a user will
//     never edit it.
//   * Notes: I didn't write "Prefer" on this one.  Don't violate it.
//     Just don't do it.  Ever.  Designing the Host resource in
//     violation of this was a HUGE mistake and one that I regret very
//     much.  Learn from my mistakes.
//   * Justification: Having Ambassador-set things in a subresource
//     from user-set things:
//     1. avoids races between the user updating the spec and us
//        updating the status
//     2. allows watt/whatever to only pay attention to
//        .metadata.generation instead of .metadata.resourceVersion;
//        avoiding pointless reconfigures.
//     3. allows the RBAC to be simpler
//     4. avoids the whole class of bugs where we need to make sure
//        that everything round-trips correctly
//     5. provides clarity on which things a user is expected to know
//        how to fill in

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
	// +k8s:conversion-gen=false
	raw json.RawMessage
}

func (u UntypedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.raw)
}

func (u *UntypedValue) UnmarshalJSON(data []byte) error {
	*u = UntypedValue{raw: json.RawMessage(data)}
	return nil
}
