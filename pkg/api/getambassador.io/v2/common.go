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
// be dumb and say `+kubebuilder:validation:Type=""`, to define its schema
// as
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
//  > the proper thing to do.  But, um, it doesn't work... kubectl would
//  > say things like:
//  >
//  >    Invalid value: "array": spec.ambassador_id in body must be of type Any: "array"
//
// But honestly that's dumb, and we can do better than that.
//
// So, option one choice would be to send the controller-tools folks a PR
// to support the openapi-gen methods to allow that customization.  That's
// probably the Right Thing, but that seemed like more work than option
// two.  FIXME(lukeshu): Send the controller-tools folks a PR.
//
// Option two: Say something nonsensical like
// `+kubebuilder:validation:Type="d6e-union"`, and teach the `fix-crds`
// script to notice that and delete that nonsensical `type`, replacing it
// with the appropriate `oneOf: [type: A, type: B]` (note that the version
// of JSONSchema that OpenAPI/Kubernetes uses doesn't support type being an
// array).  And so that's what I did.
//
// FIXME(lukeshu): But all of that is still terrible.  Because the very
// structure of our data inherently means that we must have a
// non-structural[3] schema.  With "apiextensions.k8s.io/v1beta1" CRDs,
// non-structural schemas disable several features; and in v1 CRDs,
// non-structural schemas are entirely forbidden.  I mean it doesn't
// _really_ matter right now, because we give out v1beta1 CRDs anyway
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
	MaxConnections     int    `json:"max_connections,omitempty"`
	MaxPendingRequests int    `json:"max_pending_requests,omitempty"`
	MaxRequests        int    `json:"max_requests,omitempty"`
	MaxRetries         int    `json:"max_retries,omitempty"`
}

type KeepAlive struct {
	Probes   int `json:"probes,omitempty"`
	IdleTime int `json:"idle_time,omitempty"`
	Interval int `json:"interval,omitempty"`
}

type CORS struct {
	Origins        []string `json:"origins,omitempty"`
	Methods        []string `json:"methods,omitempty"`
	Headers        []string `json:"headers,omitempty"`
	ExposedHeaders []string `json:"exposed_headers,omitempty"`
	Credentials    bool     `json:"credentials,omitempty"`
	MaxAge         string   `json:"max_age,omitempty"`
}

type RetryPolicy struct {
	// +kubebuilder:validation:Enum={"5xx","gateway-error","connect-failure","retriable-4xx","refused-stream","retriable-status-codes"}
	RetryOn       string `json:"retry_on,omitempty"`
	NumRetries    int    `json:"num_retries,omitempty"`
	PerTryTimeout string `json:"per_try_timeout,omitempty"`
}

type LoadBalancer struct {
	// +kubebuilder:validation:Enum={"round_robin","ring_hash","maglev","least_request"}
	Policy   string              `json:"policy,omitempty"`
	Cookie   *LoadBalancerCookie `json:"cookie,omitempty"`
	Header   string              `json:"header,omitempty"`
	SourceIp bool                `json:"source_ip,omitempty"`
}

type LoadBalancerCookie struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	Ttl  string `json:"ttl,omitempty"`
}

// AmbassadorID declares which Ambassador instances should pay
// attention to this resource.  May either be a string or a list of
// strings.  If no value is provided, the default is:
//
//    ambassador_id:
//    - "default"
//
// +kubebuilder:validation:Type="d6e-union:string,array"
type AmbassadorID []string

func (aid *AmbassadorID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*aid = nil
		return nil
	}

	var err error
	var list []string
	var single string

	if err = json.Unmarshal(data, &single); err == nil {
		*aid = AmbassadorID([]string{single})
		return nil
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*aid = AmbassadorID(list)
		return nil
	}

	return err
}
