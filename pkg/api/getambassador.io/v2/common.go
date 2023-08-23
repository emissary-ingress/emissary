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

package v2

import (
	"encoding/json"
	"errors"
	"time"
)

// The old `k8s.io/kube-openapi/cmd/openapi-gen` command had ways to
// specify custom schemas for your types (1: define a "OpenAPIDefinition"
// method, or 2: define a "OpenAPIV3Definition" method, or 3: define
// "OpenAPISchemaType" and "OpenAPISchemaFormat" methods).  But the new
// `sigs.k8s.io/controller-tools/controller-gen` command doesn't; it just
// has a small number of "+kubebuilder:" magic comments ("markers") that we
// can use to influence the schema it generates.
//
// So, for example, we'd like to define the AmbassadorID schema as:
//
//    oneOf:
//    - type: "string"
//    - type: "array"
//    items:             # only matters if type=array
//      type: "string"
//
// but if we're going to use just vanilla controller-gen, we're forced to
// say `+kubebuilder:validation:Type=""`, to define its schema as
//
//    # no `type:` setting because of the +kubebuilder marker
//    items:
//      type: "string"  # because of the raw type
//
// and then kubectl and/or the apiserver won't be able to validate
// AmbassadorID, because it won't be validated until we actually go to
// UnmarshalJSON it when it makes it to Ambassador.  That's pretty much
// what Kubernetes itself[1] does for the JSON Schema types that are unions
// like that.
//
//  > Aside: Some recent work in controller-gen[2] *strongly* suggests that
//  > setting `+kubebuilder:validation:Type=Any` instead of `:Type=""` is
//  > the proper thing to do.  But, it doesn't work... kubectl would
//  > say things like:
//  >
//  >    Invalid value: "array": spec.ambassador_id in body must be of type Any: "array"
//
// So, option one choice would be to send the controller-tools folks a PR
// to support the openapi-gen methods to allow that customization.  That's
// probably the Right Thing, but that seemed like more work than option
// two.
//
// Option two: Say something nonsensical like
// `+kubebuilder:validation:Type="d6e-union"`, and teach the `fix-crds`
// script to notice that and delete that nonsensical `type`, replacing it
// with the appropriate `oneOf: [type: A, type: B]` (note that the version
// of JSONSchema that OpenAPI/Kubernetes uses doesn't support type being an
// array).
//
// Because the very structure of our data inherently means that we must have a
// non-structural[3] schema.  With "apiextensions.k8s.io/v1beta1" CRDs,
// non-structural schemas disable several features; and in v1 CRDs,
// non-structural schemas are entirely forbidden.
//
// It doesn't really matter right now, because we give out v1beta1 CRDs anyway
// because v1 only became available in Kubernetes 1.16 and we still support
// down to Kubernetes 1.11; but I don't think that we want to lock
// ourselves out from v1 forever.  So I guess that means when it comes time
// for `getambassador.io/v3` (`ambassadorlabs.com/v1`?), we need to
// strictly avoid union types, in order to avoid violating rule 3 of
// structural schemas.  Or hope that the Kubernetes folks decide to relax
// some of the structural-schema rules.
//
// [1]: https://github.com/kubernetes/apiextensions-apiserver/blob/kubernetes-1.18.4/pkg/apis/apiextensions/v1beta1/types_jsonschema.go#L195-L206
// [2]: https://github.com/kubernetes-sigs/controller-tools/pull/427
// [3]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema

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

// AmbassadorID declares which Ambassador instances should pay
// attention to this resource.  May either be a string or a list of
// strings.  If no value is provided, the default is:
//
//	ambassador_id:
//	- "default"
//
// +kubebuilder:validation:Type="d6e-union:string,array"
type AmbassadorID []string

func (aid *AmbassadorID) UnmarshalJSON(data []byte) error {
	return (*StringOrStringList)(aid).UnmarshalJSON(data)
}

// StringOrStringList is just what it says on the tin, but note that it will always
// marshal as a list of strings right now.
// +kubebuilder:validation:Type="d6e-union:string,array"
type StringOrStringList []string

func (sl *StringOrStringList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*sl = nil
		return nil
	}

	var err error
	var list []string
	var single string

	if err = json.Unmarshal(data, &single); err == nil {
		*sl = StringOrStringList([]string{single})
		return nil
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*sl = StringOrStringList(list)
		return nil
	}

	return err
}

// BoolOrString is a type that can hold a Boolean or a string.
//
// +kubebuilder:validation:Type="d6e-union:string,boolean"
type BoolOrString struct {
	String *string `json:"-"`
	Bool   *bool   `json:"-"`
}

// MarshalJSON is important both so that we generate the proper
// output, and to trigger controller-gen to not try to generate
// jsonschema for our sub-fields:
// https://github.com/kubernetes-sigs/controller-tools/pull/427
func (o BoolOrString) MarshalJSON() ([]byte, error) {
	nonNil := 0
	if o.String != nil {
		nonNil++
	}
	if o.Bool != nil {
		nonNil++
	}
	if nonNil > 1 {
		return nil, errors.New("invalid BoolOrString")
	}
	switch {
	case o.String != nil:
		return json.Marshal(o.String)
	case o.Bool != nil:
		return json.Marshal(o.Bool)
	default:
		return json.Marshal(nil)
	}
}

func (o *BoolOrString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = BoolOrString{}
		return nil
	}

	var err error

	var b bool
	if err = json.Unmarshal(data, &b); err == nil {
		*o = BoolOrString{Bool: &b}
		return nil
	}

	var str string
	if err = json.Unmarshal(data, &str); err == nil {
		*o = BoolOrString{String: &str}
		return nil
	}

	return err
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
