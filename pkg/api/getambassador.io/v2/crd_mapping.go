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
	"errors"
	"strings"

	v3alpha1 "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
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
	Service            string                  `json:"service,omitempty"`
	AddRequestHeaders  *map[string]AddedHeader `json:"add_request_headers,omitempty"`
	AddResponseHeaders *map[string]AddedHeader `json:"add_response_headers,omitempty"`
	AddLinkerdHeaders  *bool                   `json:"add_linkerd_headers,omitempty"`
	AutoHostRewrite    *bool                   `json:"auto_host_rewrite,omitempty"`
	CaseSensitive      *bool                   `json:"case_sensitive,omitempty"`
	Docs               *DocsInfo               `json:"docs,omitempty"`
	DNSType            string                  `json:"dns_type,omitempty"`
	EnableIPv4         *bool                   `json:"enable_ipv4,omitempty"`
	EnableIPv6         *bool                   `json:"enable_ipv6,omitempty"`
	CircuitBreakers    []*CircuitBreaker       `json:"circuit_breakers,omitempty"`
	KeepAlive          *KeepAlive              `json:"keepalive,omitempty"`
	CORS               *CORS                   `json:"cors,omitempty"`
	RetryPolicy        *RetryPolicy            `json:"retry_policy,omitempty"`
	RespectDNSTTL      *bool                   `json:"respect_dns_ttl,omitempty"`
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
	RegexRedirect *RegexMap `json:"regex_redirect,omitempty"`
	// The response code to use when generating an HTTP redirect. Defaults to 301. Used with
	// `host_redirect`.
	// +kubebuilder:validation:Enum={301,302,303,307,308}
	RedirectResponseCode         *int                 `json:"redirect_response_code,omitempty"`
	Priority                     string               `json:"priority,omitempty"`
	Precedence                   *int                 `json:"precedence,omitempty"`
	ClusterTag                   string               `json:"cluster_tag,omitempty"`
	RemoveRequestHeaders         *StringOrStringList  `json:"remove_request_headers,omitempty"`
	RemoveResponseHeaders        *StringOrStringList  `json:"remove_response_headers,omitempty"`
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
	// +k8s:conversion-gen=false
	TLS *BoolOrString `json:"tls,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +k8s:conversion-gen:rename=HealthChecks
	V3HealthChecks []v3alpha1.HealthCheck `json:"v3health_checks,omitempty"`

	// use_websocket is deprecated, and is equivlaent to setting
	// `allow_upgrade: ["websocket"]`
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
	// +k8s:conversion-gen:rename=Hostname
	Host string `json:"host,omitempty"`
	// +k8s:conversion-gen:rename=DeprecatedHostRegex
	HostRegex *bool `json:"host_regex,omitempty"`
	// +k8s:conversion-gen=false
	Headers       map[string]BoolOrString `json:"headers,omitempty"`
	RegexHeaders  map[string]string       `json:"regex_headers,omitempty"`
	Labels        DomainMap               `json:"labels,omitempty"`
	EnvoyOverride *UntypedDict            `json:"envoy_override,omitempty"`
	LoadBalancer  *LoadBalancer           `json:"load_balancer,omitempty"`
	// +k8s:conversion-gen=false
	QueryParameters      map[string]BoolOrString `json:"query_parameters,omitempty"`
	RegexQueryParameters map[string]string       `json:"regex_query_parameters,omitempty"`

	// +k8s:conversion-gen:rename=StatsName
	V3StatsName string `json:"v3StatsName,omitempty"`
}

type RegexMap struct {
	Pattern      string `json:"pattern,omitempty"`
	Substitution string `json:"substitution,omitempty"`
}

// DocsInfo provides some extra information about the docs for the Mapping
// (used by the Dev Portal)
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

// A MappingLabelSpecifier (finally!) defines a single label. There are multiple
// kinds of label, so this is more complex than we'd like it to be. See the remarks
// about schema on custom types in `./common.go`.
//
// +kubebuilder:validation:Type="d6e-union:string,object"
type MappingLabelSpecifier struct {
	String  *string                  `json:"-"` // source-cluster, destination-cluster, remote-address, or shorthand generic
	Header  MappingLabelSpecHeader   `json:"-"` // header (NB: no need to make this a pointer because MappingLabelSpecHeader is already nil-able)
	Generic *MappingLabelSpecGeneric `json:"-"` // longhand generic
}

// A MappingLabelSpecHeaderStruct is the value struct for MappingLabelSpecifier.Header:
// the form of MappingLabelSpecifier to use when you want to take the label value from
// an HTTP header. (If we make this an anonymous struct like the others, it breaks the
// generation of its deepcopy routine. Sigh.)
type MappingLabelSpecHeaderStruct struct {
	Header           string `json:"header,omitifempty"`
	OmitIfNotPresent *bool  `json:"omit_if_not_present,omitempty"`
}

// A MappingLabelSpecHeader is just the aggregate map of MappingLabelSpecHeaderStruct,
// above. The key in the map is the label key that it will set to that header value;
// there must be exactly one key in the map.
type MappingLabelSpecHeader map[string]MappingLabelSpecHeaderStruct

// func (in *MappingLabelSpecHeader) DeepCopyInfo(out *MappingLabelSpecHeader) {
// 	x := in.OmitIfNotPresent

// 	out = MappingLabelSpecHeader{
// 		Header:           in.Header,
// 		OmitIfNotPresent: &x,
// 	}

// 	return &out
// }

// A MappingLabelSpecGeneric is a longhand generic key: it states a string which
// will be included literally in the label.
type MappingLabelSpecGeneric struct {
	GenericKey string `json:"generic_key"`
	V3Key      string `json:"v3Key,omitempty"`
}

// MarshalJSON is important both so that we generate the proper
// output, and to trigger controller-gen to not try to generate
// jsonschema for our sub-fields:
// https://github.com/kubernetes-sigs/controller-tools/pull/427
func (o MappingLabelSpecifier) MarshalJSON() ([]byte, error) {
	nonNil := 0
	if o.String != nil {
		nonNil++
	}
	if o.Header != nil {
		nonNil++
	}
	if o.Generic != nil {
		nonNil++
	}
	if nonNil > 1 {
		return nil, errors.New("invalid MappingLabelSpecifier")
	}

	switch {
	case o.String != nil:
		return json.Marshal(o.String)
	case o.Header != nil:
		return json.Marshal(o.Header)
	case o.Generic != nil:
		return json.Marshal(o.Generic)
	default:
		return json.Marshal(nil)
	}
}

// UnmarshalJSON is MarshalJSON's other half.
func (o *MappingLabelSpecifier) UnmarshalJSON(data []byte) error {
	// Handle "null" straight off...
	if string(data) == "null" {
		*o = MappingLabelSpecifier{}
		return nil
	}

	// ...and if it's anything else, try all the possibilities in turn.
	var err error

	var header MappingLabelSpecHeader

	if err = json.Unmarshal(data, &header); err == nil {
		*o = MappingLabelSpecifier{Header: header}
		return nil
	}

	var generic MappingLabelSpecGeneric

	if err = json.Unmarshal(data, &generic); err == nil {
		*o = MappingLabelSpecifier{Generic: &generic}
		return nil
	}

	var str string

	if err = json.Unmarshal(data, &str); err == nil {
		*o = MappingLabelSpecifier{String: &str}
		return nil
	}

	return errors.New("could not unmarshal MappingLabelSpecifier: invalid input")
}

// +kubebuilder:validation:Type="d6e-union:string,object"
type AddedHeader struct {
	Shorthand *string          `json:"-"`
	Full      *AddedHeaderFull `json:"-"`
}

type AddedHeaderFull struct {
	Value  string `json:"value,omitempty"`
	Append *bool  `json:"append,omitempty"`
}

// MarshalJSON is important both so that we generate the proper
// output, and to trigger controller-gen to not try to generate
// jsonschema for our sub-fields:
// https://github.com/kubernetes-sigs/controller-tools/pull/427
func (o AddedHeader) MarshalJSON() ([]byte, error) {
	switch {
	case o.Shorthand != nil:
		return json.Marshal(*o.Shorthand)
	case o.Full != nil:
		return json.Marshal(*o.Full)
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
		*o = AddedHeader{Shorthand: &str}
		return nil
	}

	var obj AddedHeaderFull
	if err = json.Unmarshal(data, &obj); err == nil {
		*o = AddedHeader{Full: &obj}
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
	// +k8s:conversion-gen=false
	Origins        *OriginList        `json:"origins,omitempty"`
	Methods        StringOrStringList `json:"methods,omitempty"`
	Headers        StringOrStringList `json:"headers,omitempty"`
	Credentials    *bool              `json:"credentials,omitempty"`
	ExposedHeaders StringOrStringList `json:"exposed_headers,omitempty"`
	MaxAge         string             `json:"max_age,omitempty"`
}

// OriginList is a list of origin strings, either as a `[]string` or as a comma-separated `string`.
//
// +kubebuilder:validation:Type="d6e-union:string,array"
type OriginList struct {
	CommaSeparated bool     `json:"-"`
	Values         []string `json:"-"`
}

func (l *OriginList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*l = OriginList{}
		return nil
	}

	var err error
	var list []string
	var single string

	if err := json.Unmarshal(data, &single); err == nil {
		*l = OriginList{
			CommaSeparated: true,
			Values:         strings.Split(single, ","),
		}
		return nil
	}

	if err = json.Unmarshal(data, &list); err == nil {
		*l = OriginList{
			Values: list,
		}
		return nil
	}

	return err
}

func (l OriginList) MarshalJSON() ([]byte, error) {
	if l.CommaSeparated {
		return json.Marshal(strings.Join(l.Values, ","))
	}
	return json.Marshal(l.Values)
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
// +kubebuilder:storageversion
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MappingSpec) DeepCopyInto(out *MappingSpec) {
	*out = *in
	if in.AmbassadorID != nil {
		in, out := &in.AmbassadorID, &out.AmbassadorID
		*out = make(AmbassadorID, len(*in))
		copy(*out, *in)
	}
	if in.PrefixRegex != nil {
		in, out := &in.PrefixRegex, &out.PrefixRegex
		*out = new(bool)
		**out = **in
	}
	if in.PrefixExact != nil {
		in, out := &in.PrefixExact, &out.PrefixExact
		*out = new(bool)
		**out = **in
	}
	if in.AddRequestHeaders != nil {
		in, out := &in.AddRequestHeaders, &out.AddRequestHeaders
		*out = new(map[string]AddedHeader)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]AddedHeader, len(*in))
			for key, val := range *in {
				(*out)[key] = *val.DeepCopy()
			}
		}
	}
	if in.AddResponseHeaders != nil {
		in, out := &in.AddResponseHeaders, &out.AddResponseHeaders
		*out = new(map[string]AddedHeader)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]AddedHeader, len(*in))
			for key, val := range *in {
				(*out)[key] = *val.DeepCopy()
			}
		}
	}
	if in.AddLinkerdHeaders != nil {
		in, out := &in.AddLinkerdHeaders, &out.AddLinkerdHeaders
		*out = new(bool)
		**out = **in
	}
	if in.AutoHostRewrite != nil {
		in, out := &in.AutoHostRewrite, &out.AutoHostRewrite
		*out = new(bool)
		**out = **in
	}
	if in.CaseSensitive != nil {
		in, out := &in.CaseSensitive, &out.CaseSensitive
		*out = new(bool)
		**out = **in
	}
	if in.Docs != nil {
		in, out := &in.Docs, &out.Docs
		*out = new(DocsInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.EnableIPv4 != nil {
		in, out := &in.EnableIPv4, &out.EnableIPv4
		*out = new(bool)
		**out = **in
	}
	if in.EnableIPv6 != nil {
		in, out := &in.EnableIPv6, &out.EnableIPv6
		*out = new(bool)
		**out = **in
	}
	if in.CircuitBreakers != nil {
		in, out := &in.CircuitBreakers, &out.CircuitBreakers
		*out = make([]*CircuitBreaker, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(CircuitBreaker)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.KeepAlive != nil {
		in, out := &in.KeepAlive, &out.KeepAlive
		*out = new(KeepAlive)
		(*in).DeepCopyInto(*out)
	}
	if in.CORS != nil {
		in, out := &in.CORS, &out.CORS
		*out = new(CORS)
		(*in).DeepCopyInto(*out)
	}
	if in.RetryPolicy != nil {
		in, out := &in.RetryPolicy, &out.RetryPolicy
		*out = new(RetryPolicy)
		(*in).DeepCopyInto(*out)
	}
	if in.RespectDNSTTL != nil {
		in, out := &in.RespectDNSTTL, &out.RespectDNSTTL
		*out = new(bool)
		**out = **in
	}
	if in.GRPC != nil {
		in, out := &in.GRPC, &out.GRPC
		*out = new(bool)
		**out = **in
	}
	if in.HostRedirect != nil {
		in, out := &in.HostRedirect, &out.HostRedirect
		*out = new(bool)
		**out = **in
	}
	if in.MethodRegex != nil {
		in, out := &in.MethodRegex, &out.MethodRegex
		*out = new(bool)
		**out = **in
	}
	if in.RegexRedirect != nil {
		in, out := &in.RegexRedirect, &out.RegexRedirect
		*out = new(RegexMap)
		**out = **in
	}
	if in.RedirectResponseCode != nil {
		in, out := &in.RedirectResponseCode, &out.RedirectResponseCode
		*out = new(int)
		**out = **in
	}
	if in.Precedence != nil {
		in, out := &in.Precedence, &out.Precedence
		*out = new(int)
		**out = **in
	}
	if in.RemoveRequestHeaders != nil {
		in, out := &in.RemoveRequestHeaders, &out.RemoveRequestHeaders
		*out = new(StringOrStringList)
		if **in != nil {
			in, out := *in, *out
			*out = make(StringOrStringList, len(*in))
			copy(*out, *in)
		}
	}
	if in.RemoveResponseHeaders != nil {
		in, out := &in.RemoveResponseHeaders, &out.RemoveResponseHeaders
		*out = new(StringOrStringList)
		if **in != nil {
			in, out := *in, *out
			*out = make(StringOrStringList, len(*in))
			copy(*out, *in)
		}
	}
	if in.Rewrite != nil {
		in, out := &in.Rewrite, &out.Rewrite
		*out = new(string)
		**out = **in
	}
	if in.RegexRewrite != nil {
		in, out := &in.RegexRewrite, &out.RegexRewrite
		*out = new(RegexMap)
		**out = **in
	}
	if in.Shadow != nil {
		in, out := &in.Shadow, &out.Shadow
		*out = new(bool)
		**out = **in
	}
	if in.ConnectTimeout != nil {
		in, out := &in.ConnectTimeout, &out.ConnectTimeout
		*out = new(MillisecondDuration)
		**out = **in
	}
	if in.ClusterIdleTimeout != nil {
		in, out := &in.ClusterIdleTimeout, &out.ClusterIdleTimeout
		*out = new(MillisecondDuration)
		**out = **in
	}
	if in.ClusterMaxConnectionLifetime != nil {
		in, out := &in.ClusterMaxConnectionLifetime, &out.ClusterMaxConnectionLifetime
		*out = new(MillisecondDuration)
		**out = **in
	}
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(MillisecondDuration)
		**out = **in
	}
	if in.IdleTimeout != nil {
		in, out := &in.IdleTimeout, &out.IdleTimeout
		*out = new(MillisecondDuration)
		**out = **in
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = new(BoolOrString)
		(*in).DeepCopyInto(*out)
	}
	if in.V3HealthChecks != nil {
		in, out := &in.V3HealthChecks, &out.V3HealthChecks
		*out = make([]v3alpha1.HealthCheck, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.DeprecatedUseWebsocket != nil {
		in, out := &in.DeprecatedUseWebsocket, &out.DeprecatedUseWebsocket
		*out = new(bool)
		**out = **in
	}
	if in.AllowUpgrade != nil {
		in, out := &in.AllowUpgrade, &out.AllowUpgrade
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Weight != nil {
		in, out := &in.Weight, &out.Weight
		*out = new(int)
		**out = **in
	}
	if in.BypassAuth != nil {
		in, out := &in.BypassAuth, &out.BypassAuth
		*out = new(bool)
		**out = **in
	}
	if in.AuthContextExtensions != nil {
		in, out := &in.AuthContextExtensions, &out.AuthContextExtensions
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.BypassErrorResponseOverrides != nil {
		in, out := &in.BypassErrorResponseOverrides, &out.BypassErrorResponseOverrides
		*out = new(bool)
		**out = **in
	}
	if in.ErrorResponseOverrides != nil {
		in, out := &in.ErrorResponseOverrides, &out.ErrorResponseOverrides
		*out = make([]ErrorResponseOverride, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Modules != nil {
		in, out := &in.Modules, &out.Modules
		*out = make([]UntypedDict, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.HostRegex != nil {
		in, out := &in.HostRegex, &out.HostRegex
		*out = new(bool)
		**out = **in
	}
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]BoolOrString, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.RegexHeaders != nil {
		in, out := &in.RegexHeaders, &out.RegexHeaders
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(DomainMap, len(*in))
		for key, val := range *in {
			var outVal []MappingLabelGroup
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(MappingLabelGroupsArray, len(*in))
				for i := range *in {
					if (*in)[i] != nil {
						in, out := &(*in)[i], &(*out)[i]
						*out = make(MappingLabelGroup, len(*in))
						for key, val := range *in {
							var outVal []MappingLabelSpecifier
							if val == nil {
								(*out)[key] = nil
							} else {
								in, out := &val, &outVal
								*out = make(MappingLabelsArray, len(*in))
								for i := range *in {
									(*in)[i].DeepCopyInto(&(*out)[i])
								}
							}
							(*out)[key] = outVal
						}
					}
				}
			}
			(*out)[key] = outVal
		}
	}
	if in.EnvoyOverride != nil {
		in, out := &in.EnvoyOverride, &out.EnvoyOverride
		*out = new(UntypedDict)
		(*in).DeepCopyInto(*out)
	}
	if in.LoadBalancer != nil {
		in, out := &in.LoadBalancer, &out.LoadBalancer
		*out = new(LoadBalancer)
		(*in).DeepCopyInto(*out)
	}
	if in.QueryParameters != nil {
		in, out := &in.QueryParameters, &out.QueryParameters
		*out = make(map[string]BoolOrString, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.RegexQueryParameters != nil {
		in, out := &in.RegexQueryParameters, &out.RegexQueryParameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}
