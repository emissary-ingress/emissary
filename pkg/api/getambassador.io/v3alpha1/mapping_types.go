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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ambv2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
)

// AmbassadorMappingSpec defines the desired state of AmbassadorMapping
type AmbassadorMappingSpec struct {
	AmbassadorID ambv2.AmbassadorID `json:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Required
	Prefix      string `json:"prefix,omitempty"`
	PrefixRegex *bool  `json:"prefix_regex,omitempty"`
	PrefixExact *bool  `json:"prefix_exact,omitempty"`
	// +kubebuilder:validation:Required
	Service            string                  `json:"service,omitempty"`
	AddRequestHeaders  map[string]AddedHeader  `json:"add_request_headers,omitempty"`
	AddResponseHeaders map[string]AddedHeader  `json:"add_response_headers,omitempty"`
	AddLinkerdHeaders  *bool                   `json:"add_linkerd_headers,omitempty"`
	AutoHostRewrite    *bool                   `json:"auto_host_rewrite,omitempty"`
	CaseSensitive      *bool                   `json:"case_sensitive,omitempty"`
	Docs               *DocsInfo               `json:"docs,omitempty"`
	EnableIPv4         *bool                   `json:"enable_ipv4,omitempty"`
	EnableIPv6         *bool                   `json:"enable_ipv6,omitempty"`
	CircuitBreakers    []*ambv2.CircuitBreaker `json:"circuit_breakers,omitempty"`
	KeepAlive          *KeepAlive              `json:"keepalive,omitempty"`
	CORS               *CORS                   `json:"cors,omitempty"`
	RetryPolicy        *RetryPolicy            `json:"retry_policy,omitempty"`
	GRPC               *bool                   `json:"grpc,omitempty"`
	HostRedirect       *bool                   `json:"host_redirect,omitempty"`
	HostRewrite        string                  `json:"host_rewrite,omitempty"`
	Method             string                  `json:"method,omitempty"`
	MethodRegex        *bool                   `json:"method_regex,omitempty"`
	OutlierDetection   string                  `json:"outlier_detection,omitempty"`
	// Path replacement to use when generating an HTTP redirect. Used with `host_redirect`.
	PathRedirect string `json:"path_redirect,omitempty"`
	// Prefix rewrite to use when generating an HTTP redirect. Used with `host_redirect`.
	PrefixRedirect string `json:"prefix_redirect,omitempty"`
	// Prefix regex rewrite to use when generating an HTTP redirect. Used with `host_redirect`.
	RegexRedirect map[string]ambv2.BoolOrString `json:"regex_redirect,omitempty"`
	// The response code to use when generating an HTTP redirect. Defaults to 301. Used with
	// `host_redirect`.
	// +kubebuilder:validation:Enum={301,302,303,307,308}
	RedirectResponseCode           *int                          `json:"redirect_response_code,omitempty"`
	Priority                       string                        `json:"priority,omitempty"`
	Precedence                     *int                          `json:"precedence,omitempty"`
	ClusterTag                     string                        `json:"cluster_tag,omitempty"`
	RemoveRequestHeaders           []string                      `json:"remove_request_headers,omitempty"`
	RemoveResponseHeaders          []string                      `json:"remove_response_headers,omitempty"`
	Resolver                       string                        `json:"resolver,omitempty"`
	Rewrite                        *string                       `json:"rewrite,omitempty"`
	RegexRewrite                   map[string]ambv2.BoolOrString `json:"regex_rewrite,omitempty"`
	Shadow                         *bool                         `json:"shadow,omitempty"`
	ConnectTimeoutMs               *int                          `json:"connect_timeout_ms,omitempty"`
	ClusterIdleTimeoutMs           *int                          `json:"cluster_idle_timeout_ms,omitempty"`
	ClusterMaxConnectionLifetimeMs int                           `json:"cluster_max_connection_lifetime_ms,omitempty"`
	// The timeout for requests that use this AmbassadorMapping. Overrides `cluster_request_timeout_ms` set on the Ambassador Module, if it exists.
	TimeoutMs     *int                `json:"timeout_ms,omitempty"`
	IdleTimeoutMs *int                `json:"idle_timeout_ms,omitempty"`
	TLS           *ambv2.BoolOrString `json:"tls,omitempty"`

	// use_websocket is deprecated, and is equivlaent to setting
	// `allow_upgrade: ["websocket"]`
	UseWebsocket *bool `json:"use_websocket,omitempty"`

	// A case-insensitive list of the non-HTTP protocols to allow
	// "upgrading" to from HTTP via the "Connection: upgrade"
	// mechanism[1].  After the upgrade, Ambassador does not
	// interpret the traffic, and behaves similarly to how it does
	// for TCPMappings.
	//
	// [1]: https://tools.ietf.org/html/rfc7230#section-6.7
	//
	// For example, if your upstream service supports WebSockets,
	// you would write
	//
	//    allow_upgrade:
	//    - websocket
	//
	// Or if your upstream service supports upgrading from HTTP to
	// SPDY (as the Kubernetes apiserver does for `kubectl exec`
	// functionality), you would write
	//
	//    allow_upgrade:
	//    - spdy/3.1
	AllowUpgrade []string `json:"allow_upgrade,omitempty"`

	Weight                *int              `json:"weight,omitempty"`
	BypassAuth            *bool             `json:"bypass_auth,omitempty"`
	AuthContextExtensions map[string]string `json:"auth_context_extensions,omitempty"`
	// If true, bypasses any `error_response_overrides` set on the Ambassador module.
	BypassErrorResponseOverrides *bool `json:"bypass_error_response_overrides,omitempty"`
	// Error response overrides for this AmbassadorMapping. Replaces all of the `error_response_overrides`
	// set on the Ambassador module, if any.
	// +kubebuilder:validation:MinItems=1
	ErrorResponseOverrides []ambv2.ErrorResponseOverride `json:"error_response_overrides,omitempty"`
	Modules                []ambv2.UntypedDict           `json:"modules,omitempty"`
	Host                   string                        `json:"host,omitempty"`
	Hostname               string                        `json:"hostname,omitempty"`
	HostRegex              *bool                         `json:"host_regex,omitempty"`
	Headers                map[string]ambv2.BoolOrString `json:"headers,omitempty"`
	RegexHeaders           map[string]ambv2.BoolOrString `json:"regex_headers,omitempty"`
	Labels                 ambv2.DomainMap               `json:"labels,omitempty"`
	EnvoyOverride          *ambv2.UntypedDict            `json:"envoy_override,omitempty"`
	LoadBalancer           *ambv2.LoadBalancer           `json:"load_balancer,omitempty"`
	QueryParameters        map[string]ambv2.BoolOrString `json:"query_parameters,omitempty"`
	RegexQueryParameters   map[string]ambv2.BoolOrString `json:"regex_query_parameters,omitempty"`
	StatsName              string                        `json:"stats_name,omitempty"`
}

// DocsInfo provides some extra information about the docs for the AmbassadorMapping.
// Docs is used by both the agent and the DevPortal.
type DocsInfo struct {
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	Ignored     *bool  `json:"ignored,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// +kubebuilder:validation:Type="d6e-union:string,boolean,object"
type AddedHeader struct {
	String *string            `json:"-"`
	Bool   *bool              `json:"-"`
	Object *ambv2.UntypedDict `json:"-"`
}

// MarshalJSON is important both so that we generate the proper
// output, and to trigger controller-gen to not try to generate
// jsonschema for our sub-fields:
// https://github.com/kubernetes-sigs/controller-tools/pull/427
func (o AddedHeader) MarshalJSON() ([]byte, error) {
	switch {
	case o.String != nil:
		return json.Marshal(*o.String)
	case o.Bool != nil:
		return json.Marshal(*o.Bool)
	case o.Object != nil:
		return json.Marshal(*o.Object)
	default:
		return json.Marshal(nil)
	}
}

func (o *AddedHeader) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = AddedHeader{}
		return nil
	}

	var err error

	var str string
	if err = json.Unmarshal(data, &str); err == nil {
		*o = AddedHeader{String: &str}
		return nil
	}

	var b bool
	if err = json.Unmarshal(data, &b); err == nil {
		*o = AddedHeader{Bool: &b}
		return nil
	}

	var obj ambv2.UntypedDict
	if err = json.Unmarshal(data, &obj); err == nil {
		*o = AddedHeader{Object: &obj}
		return nil
	}

	return err
}

type KeepAlive struct {
	Probes   *int `json:"probes,omitempty"`
	IdleTime *int `json:"idle_time,omitempty"`
	Interval *int `json:"interval,omitempty"`
}

type CORS struct {
	Origins        []string `json:"origins,omitempty"`
	Methods        []string `json:"methods,omitempty"`
	Headers        []string `json:"headers,omitempty"`
	Credentials    *bool    `json:"credentials,omitempty"`
	ExposedHeaders []string `json:"exposed_headers,omitempty"`
	MaxAge         string   `json:"max_age,omitempty"`
}

type RetryPolicy struct {
	// +kubebuilder:validation:Enum={"5xx","gateway-error","connect-failure","retriable-4xx","refused-stream","retriable-status-codes"}
	RetryOn       string `json:"retry_on,omitempty"`
	NumRetries    *int   `json:"num_retries,omitempty"`
	PerTryTimeout string `json:"per_try_timeout,omitempty"`
}

type LoadBalancer struct {
	// +kubebuilder:validation:Enum={"round_robin","ring_hash","maglev","least_request"}
	// +kubebuilder:validation:Required
	Policy   string              `json:"policy,omitempty"`
	Cookie   *LoadBalancerCookie `json:"cookie,omitempty"`
	Header   string              `json:"header,omitempty"`
	SourceIp *bool               `json:"source_ip,omitempty"`
}

type LoadBalancerCookie struct {
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	Ttl  string `json:"ttl,omitempty"`
}

// AmbassadorMappingStatus defines the observed state of AmbassadorMapping
type AmbassadorMappingStatus struct {
	// +kubebuilder:validation:Enum={"","Inactive","Running"}
	State string `json:"state,omitempty"`

	Reason string `json:"reason,omitempty"`
}

// AmbassadorMapping is the Schema for the mappings API
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Source Host",type=string,JSONPath=`.spec.host`
// +kubebuilder:printcolumn:name="Source Prefix",type=string,JSONPath=`.spec.prefix`
// +kubebuilder:printcolumn:name="Dest Service",type=string,JSONPath=`.spec.service`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
type AmbassadorMapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AmbassadorMappingSpec    `json:"spec,omitempty"`
	Status *AmbassadorMappingStatus `json:"status,omitempty"`
}

// AmbassadorMappingList contains a list of AmbassadorMappings.
//
// +kubebuilder:object:root=true
type AmbassadorMappingList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AmbassadorMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AmbassadorMapping{}, &AmbassadorMappingList{})
}
