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

// MappingSpec defines the desired state of Mapping
type MappingSpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id,omitempty"`

	// +kubebuilder:validation:Required
	Prefix      string `json:"prefix,omitempty"`
	PrefixRegex *bool  `json:"prefix_regex,omitempty"`
	PrefixExact *bool  `json:"prefix_exact,omitempty"`
	// +kubebuilder:validation:Required
	Service            string                 `json:"service,omitempty"`
	AddRequestHeaders  map[string]AddedHeader `json:"add_request_headers,omitempty"`
	AddResponseHeaders map[string]AddedHeader `json:"add_response_headers,omitempty"`
	AddLinkerdHeaders  *bool                  `json:"add_linkerd_headers,omitempty"`
	AutoHostRewrite    *bool                  `json:"auto_host_rewrite,omitempty"`
	CaseSensitive      *bool                  `json:"case_sensitive,omitempty"`
	DNSType            string                 `json:"dns_type,omitempty"`
	Docs               *DocsInfo              `json:"docs,omitempty"`
	EnableIPv4         *bool                  `json:"enable_ipv4,omitempty"`
	EnableIPv6         *bool                  `json:"enable_ipv6,omitempty"`
	CircuitBreakers    []*CircuitBreaker      `json:"circuit_breakers,omitempty"`
	KeepAlive          *KeepAlive             `json:"keepalive,omitempty"`
	CORS               *CORS                  `json:"cors,omitempty"`
	RetryPolicy        *RetryPolicy           `json:"retry_policy,omitempty"`
	RespectDNSTTL      *bool                  `json:"respect_dns_ttl,omitempty"`
	GRPC               *bool                  `json:"grpc,omitempty"`
	HostRedirect       *bool                  `json:"host_redirect,omitempty"`
	HostRewrite        string                 `json:"host_rewrite,omitempty"`
	Method             string                 `json:"method,omitempty"`
	MethodRegex        *bool                  `json:"method_regex,omitempty"`
	OutlierDetection   string                 `json:"outlier_detection,omitempty"`
	// Path replacement to use when generating an HTTP redirect. Used with `host_redirect`.
	PathRedirect string `json:"path_redirect,omitempty"`
	// Prefix rewrite to use when generating an HTTP redirect. Used with `host_redirect`.
	PrefixRedirect string `json:"prefix_redirect,omitempty"`
	// Prefix regex rewrite to use when generating an HTTP redirect. Used with `host_redirect`.
	RegexRedirect *RegexMap `json:"regex_redirect,omitempty"`
	// The response code to use when generating an HTTP redirect. Defaults to 301. Used with
	// `host_redirect`.
	// +kubebuilder:validation:Enum={301,302,303,307,308}
	RedirectResponseCode         *int                 `json:"redirect_response_code,omitempty"`
	Priority                     string               `json:"priority,omitempty"`
	Precedence                   *int                 `json:"precedence,omitempty"`
	ClusterTag                   string               `json:"cluster_tag,omitempty"`
	RemoveRequestHeaders         []string             `json:"remove_request_headers,omitempty"`
	RemoveResponseHeaders        []string             `json:"remove_response_headers,omitempty"`
	Resolver                     string               `json:"resolver,omitempty"`
	Rewrite                      *string              `json:"rewrite,omitempty"`
	RegexRewrite                 *RegexMap            `json:"regex_rewrite,omitempty"`
	Shadow                       *bool                `json:"shadow,omitempty"`
	ConnectTimeout               *MillisecondDuration `json:"connect_timeout_ms,omitempty"`
	ClusterIdleTimeout           *MillisecondDuration `json:"cluster_idle_timeout_ms,omitempty"`
	ClusterMaxConnectionLifetime *MillisecondDuration `json:"cluster_max_connection_lifetime_ms,omitempty"`
	// The timeout for requests that use this Mapping. Overrides `cluster_request_timeout_ms` set on the Ambassador Module, if it exists.
	Timeout     *MillisecondDuration `json:"timeout_ms,omitempty"`
	IdleTimeout *MillisecondDuration `json:"idle_timeout_ms,omitempty"`
	TLS         string               `json:"tls,omitempty"`

	// use_websocket is deprecated, and is equivlaent to setting
	// `allow_upgrade: ["websocket"]`
	//
	// TODO(lukeshu): In v3alpha2, get rid of MappingSpec.DeprecatedUseWebsocket.
	DeprecatedUseWebsocket *bool `json:"use_websocket,omitempty"`

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
	// Error response overrides for this Mapping. Replaces all of the `error_response_overrides`
	// set on the Ambassador module, if any.
	// +kubebuilder:validation:MinItems=1
	ErrorResponseOverrides []ErrorResponseOverride `json:"error_response_overrides,omitempty"`
	Modules                []UntypedDict           `json:"modules,omitempty"`

	// Exact match for the hostname of a request if HostRegex is false; regex match for the
	// hostname if HostRegex is true.
	//
	// Host specifies both a match for the ':authority' header of a request, as well as a match
	// criterion for Host CRDs: a Mapping that specifies Host will not associate with a Host that
	// doesn't have a matching Hostname.
	//
	// If both Host and Hostname are set, an error is logged, Host is ignored, and Hostname is
	// used.
	//
	// DEPRECATED: Host is either an exact match or a regex, depending on HostRegex. Use HostName instead.
	//
	// TODO(lukeshu): In v3alpha2, get rid of MappingSpec.host and MappingSpec.host_regex in
	// favor of a MappingSpec.deprecated_hostname_regex.
	DeprecatedHost string `json:"host,omitempty"`
	// DEPRECATED: Host is either an exact match or a regex, depending on HostRegex. Use HostName instead.
	//
	// TODO(lukeshu): In v3alpha2, get rid of MappingSpec.host and MappingSpec.host_regex in
	// favor of a MappingSpec.deprecated_hostname_regex.
	DeprecatedHostRegex *bool `json:"host_regex,omitempty"`
	// Hostname is a DNS glob specifying the hosts to which this Mapping applies.
	//
	// Hostname specifies both a match for the ':authority' header of a request, as well as a
	// match criterion for Host CRDs: a Mapping that specifies Hostname will not associate with
	// a Host that doesn't have a matching Hostname.
	//
	// If both Host and Hostname are set, an error is logged, Host is ignored, and Hostname is
	// used.
	Hostname string `json:"hostname,omitempty"`

	Headers              map[string]string `json:"headers,omitempty"`
	RegexHeaders         map[string]string `json:"regex_headers,omitempty"`
	Labels               DomainMap         `json:"labels,omitempty"`
	EnvoyOverride        *UntypedDict      `json:"envoy_override,omitempty"`
	LoadBalancer         *LoadBalancer     `json:"load_balancer,omitempty"`
	QueryParameters      map[string]string `json:"query_parameters,omitempty"`
	RegexQueryParameters map[string]string `json:"regex_query_parameters,omitempty"`
	StatsName            string            `json:"stats_name,omitempty"`

	V2ExplicitTLS         *V2ExplicitTLS `json:"v2ExplicitTLS,omitempty"`
	V2BoolHeaders         []string       `json:"v2BoolHeaders,omitempty"`
	V2BoolQueryParameters []string       `json:"v2BoolQueryParameters,omitempty"`
}

type RegexMap struct {
	Pattern      string `json:"pattern,omitempty"`
	Substitution string `json:"substitution,omitempty"`
}

// DocsInfo provides some extra information about the docs for the Mapping.
// Docs is used by both the agent and the DevPortal.
type DocsInfo struct {
	Path        string               `json:"path,omitempty"`
	URL         string               `json:"url,omitempty"`
	Ignored     *bool                `json:"ignored,omitempty"`
	DisplayName string               `json:"display_name,omitempty"`
	Timeout     *MillisecondDuration `json:"timeout_ms,omitempty"`
}

// These are separate types partly because it makes it easier to think about
// how `DomainMap` is built up, but also because it's pretty awful to read
// a type definition that's four or five levels deep with maps and arrays.

// A DomainMap is the overall Mapping.spec.Labels type. It maps domains (kind of
// like namespaces for Mapping labels) to arrays of label groups.
type DomainMap map[string]MappingLabelGroupsArray

// A MappingLabelGroupsArray is an array of MappingLabelGroups. I know, complex.
type MappingLabelGroupsArray []MappingLabelGroup

// A MappingLabelGroup is a single element of a MappingLabelGroupsArray: a second
// map, where the key is a human-readable name that identifies the group.
//
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type MappingLabelGroup map[string]MappingLabelsArray

// A MappingLabelsArray is the value in the MappingLabelGroup: an array of label
// specifiers.
type MappingLabelsArray []MappingLabelSpecifier

// A MappingLabelSpecifier (finally!) defines a single label.
//
// This mimics envoy/config/route/v3/route_components.proto:RateLimit:Action:action_specifier.
//
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type MappingLabelSpecifier struct {
	// Sets the label "source_cluster=«Envoy source cluster name»".
	SourceCluster *MappingLabelSpecifier_SourceCluster `json:"source_cluster,omitempty"`

	// Sets the label "destination_cluster=«Envoy destination cluster name»".
	DestinationCluster *MappingLabelSpecifier_DestinationCluster `json:"destination_cluster,omitempty"`

	// If the «header_name» header is set, then set the label "«key»=«Value of the
	// «header_name» header»"; otherwise skip applying this label group.
	RequestHeaders *MappingLabelSpecifier_RequestHeaders `json:"request_headers,omitempty"`

	// Sets the label "remote_address=«IP address of the client»".
	RemoteAddress *MappingLabelSpecifier_RemoteAddress `json:"remote_address,omitempty"`

	// Sets the label "«key»=«value»" (where by default «key»
	// is "generic_key").
	GenericKey *MappingLabelSpecifier_GenericKey `json:"generic_key,omitempty"`

	// TODO: Consider implementing `header_value_match`, `metadata`, or `extension`?
}

type MappingLabelSpecifier_SourceCluster struct {
	// +kubebuilder:validation:Enum={"source_cluster"}
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

type MappingLabelSpecifier_DestinationCluster struct {
	// +kubebuilder:validation:Enum={"destination_cluster"}
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

type MappingLabelSpecifier_RequestHeaders struct {
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// +kubebuilder:validation:Required
	HeaderName string `json:"header_name"`

	OmitIfNotPresent *bool `json:"omit_if_not_present,omitempty"`
}

type MappingLabelSpecifier_RemoteAddress struct {
	// +kubebuilder:validation:Enum={"remote_address"}
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

type MappingLabelSpecifier_GenericKey struct {
	// The default is "generic_key".
	Key string `json:"key,omitempty"`

	// +kubebuilder:validation:Required
	Value string `json:"value"`

	V2Shorthand bool `json:"v2Shorthand,omitempty"`
}

type AddedHeader struct {
	Value  string `json:"value,omitempty"`
	Append *bool  `json:"append,omitempty"`

	// +kubebuilder:validation:Enum={"","string","null"}
	V2Representation string `json:"v2Representation,omitempty"`
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

	V2CommaSeparatedOrigins bool `json:"v2CommaSeparatedOrigins,omitempty"`
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

// MappingStatus defines the observed state of Mapping
type MappingStatus struct {
	// +kubebuilder:validation:Enum={"","Inactive","Running"}
	State string `json:"state,omitempty"`

	Reason string `json:"reason,omitempty"`
}

// Mapping is the Schema for the mappings API
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Source Host",type=string,JSONPath=`.spec.host`
// +kubebuilder:printcolumn:name="Source Prefix",type=string,JSONPath=`.spec.prefix`
// +kubebuilder:printcolumn:name="Dest Service",type=string,JSONPath=`.spec.service`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
type Mapping struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MappingSpec    `json:"spec,omitempty"`
	Status *MappingStatus `json:"status,omitempty"`
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
