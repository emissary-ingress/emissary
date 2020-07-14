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
// So, for example, instead of having the AmbassadorID type define its
// schema as
//
//    anyOf:
//    - type: "string"
//    - type: "array"
//      items:
//        type: "string"
//
// we're forced to be dumb and say `+kubebuilder:validation:Type=""`, to
// define its schema as
//
//    # no `type:` setting because of the +kubebuilder marker
//    items:
//      type: "string"  # because of the raw type
//
// and then kubectl and/or the api-server won't be able to validate
// AmbassadorID, because it won't be validated until we actually go to
// UnmarshalJSON it when it makes it to Ambassador.
//
// That's pretty much what Kubernetes itself does for the JSON Schema
// types that are unions like that:
// https://github.com/kubernetes/apiextensions-apiserver/blob/kubernetes-1.18.4/pkg/apis/apiextensions/v1beta1/types_jsonschema.go#L195-L206
//
// Some recent work in controller-gen[1] *strongly* suggests that setting
// `+kubebuilder:validation:Type=Any` should work.  But, um, it
// doesn't... kubectl would say things like:
//
//    Invalid value: "array": spec.ambassador_id in body must be of type Any: "array"
//
// FIXME(lukeshu): Try sending the controller-tools folks a PR to support the
// openapi-gen methods?
//
// FIXME(lukeshu): Both the "don't set 'type'" and the "patch
// controller-tools to support anyOf" options are bad options, since in
// either case they make our schema non-structural[2].  With
// "apiextensions.k8s.io/v1beta1" CRDs, non-structural schemas disable
// several features; and in v1 CRDs, non-structural schemas are entirely
// forbidden.  I mean it doesn't _really_ matter right now, because we give
// out v1beta1 CRDs anyway because v1 only became available in Kube 1.16
// and we still support down to Kube 1.11; but I don't think that we want
// to lock ourselves out from v1 forever.  Anyway, as best as I can figure
// out, it isn't actually possible to specify AmbassadorID in way that
// doesn't violate rule 3 of structural schemas.
//
// [1]: https://github.com/kubernetes-sigs/controller-tools/pull/427
// [2]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema

type CircuitBreaker struct {
	// +kubebuilder:validation:Enum={"default", "high"}
	Priority           string `json:"priority,omitempty"`
	MaxConnections     int32  `json:"max_connections,omitempty"`
	MaxPendingRequests int32  `json:"max_pending_requests,omitempty"`
	MaxRequests        int32  `json:"max_requests,omitempty"`
	MaxRetries         int32  `json:"max_retries,omitempty"`
}

type KeepAlive struct {
	Probes   int32 `json:"probes,omitempty"`
	IdleTime int32 `json:"idle_time,omitempty"`
	Interval int32 `json:"interval,omitempty"`
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
	NumRetries    int32  `json:"num_retries,omitempty"`
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

// +kubebuilder:validation:Type=""
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
