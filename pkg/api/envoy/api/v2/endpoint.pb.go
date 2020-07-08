// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/api/v2/endpoint.proto

package envoy_api_v2

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	endpoint "github.com/datawire/ambassador/pkg/api/envoy/api/v2/endpoint"
	_type "github.com/datawire/ambassador/pkg/api/envoy/type"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	io "io"
	_ "istio.io/gogo-genproto/googleapis/google/api"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Each route from RDS will map to a single cluster or traffic split across
// clusters using weights expressed in the RDS WeightedCluster.
//
// With EDS, each cluster is treated independently from a LB perspective, with
// LB taking place between the Localities within a cluster and at a finer
// granularity between the hosts within a locality. The percentage of traffic
// for each endpoint is determined by both its load_balancing_weight, and the
// load_balancing_weight of its locality. First, a locality will be selected,
// then an endpoint within that locality will be chose based on its weight.
// [#next-free-field: 6]
type ClusterLoadAssignment struct {
	// Name of the cluster. This will be the :ref:`service_name
	// <envoy_api_field_Cluster.EdsClusterConfig.service_name>` value if specified
	// in the cluster :ref:`EdsClusterConfig
	// <envoy_api_msg_Cluster.EdsClusterConfig>`.
	ClusterName string `protobuf:"bytes,1,opt,name=cluster_name,json=clusterName,proto3" json:"cluster_name,omitempty"`
	// List of endpoints to load balance to.
	Endpoints []*endpoint.LocalityLbEndpoints `protobuf:"bytes,2,rep,name=endpoints,proto3" json:"endpoints,omitempty"`
	// Map of named endpoints that can be referenced in LocalityLbEndpoints.
	// [#not-implemented-hide:]
	NamedEndpoints map[string]*endpoint.Endpoint `protobuf:"bytes,5,rep,name=named_endpoints,json=namedEndpoints,proto3" json:"named_endpoints,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Load balancing policy settings.
	Policy               *ClusterLoadAssignment_Policy `protobuf:"bytes,4,opt,name=policy,proto3" json:"policy,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                      `json:"-"`
	XXX_unrecognized     []byte                        `json:"-"`
	XXX_sizecache        int32                         `json:"-"`
}

func (m *ClusterLoadAssignment) Reset()         { *m = ClusterLoadAssignment{} }
func (m *ClusterLoadAssignment) String() string { return proto.CompactTextString(m) }
func (*ClusterLoadAssignment) ProtoMessage()    {}
func (*ClusterLoadAssignment) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dd3a9d301e6758b, []int{0}
}
func (m *ClusterLoadAssignment) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ClusterLoadAssignment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ClusterLoadAssignment.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ClusterLoadAssignment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ClusterLoadAssignment.Merge(m, src)
}
func (m *ClusterLoadAssignment) XXX_Size() int {
	return m.Size()
}
func (m *ClusterLoadAssignment) XXX_DiscardUnknown() {
	xxx_messageInfo_ClusterLoadAssignment.DiscardUnknown(m)
}

var xxx_messageInfo_ClusterLoadAssignment proto.InternalMessageInfo

func (m *ClusterLoadAssignment) GetClusterName() string {
	if m != nil {
		return m.ClusterName
	}
	return ""
}

func (m *ClusterLoadAssignment) GetEndpoints() []*endpoint.LocalityLbEndpoints {
	if m != nil {
		return m.Endpoints
	}
	return nil
}

func (m *ClusterLoadAssignment) GetNamedEndpoints() map[string]*endpoint.Endpoint {
	if m != nil {
		return m.NamedEndpoints
	}
	return nil
}

func (m *ClusterLoadAssignment) GetPolicy() *ClusterLoadAssignment_Policy {
	if m != nil {
		return m.Policy
	}
	return nil
}

// Load balancing policy settings.
// [#next-free-field: 6]
type ClusterLoadAssignment_Policy struct {
	// Action to trim the overall incoming traffic to protect the upstream
	// hosts. This action allows protection in case the hosts are unable to
	// recover from an outage, or unable to autoscale or unable to handle
	// incoming traffic volume for any reason.
	//
	// At the client each category is applied one after the other to generate
	// the 'actual' drop percentage on all outgoing traffic. For example:
	//
	// .. code-block:: json
	//
	//  { "drop_overloads": [
	//      { "category": "throttle", "drop_percentage": 60 }
	//      { "category": "lb", "drop_percentage": 50 }
	//  ]}
	//
	// The actual drop percentages applied to the traffic at the clients will be
	//    "throttle"_drop = 60%
	//    "lb"_drop = 20%  // 50% of the remaining 'actual' load, which is 40%.
	//    actual_outgoing_load = 20% // remaining after applying all categories.
	DropOverloads []*ClusterLoadAssignment_Policy_DropOverload `protobuf:"bytes,2,rep,name=drop_overloads,json=dropOverloads,proto3" json:"drop_overloads,omitempty"`
	// Priority levels and localities are considered overprovisioned with this
	// factor (in percentage). This means that we don't consider a priority
	// level or locality unhealthy until the percentage of healthy hosts
	// multiplied by the overprovisioning factor drops below 100.
	// With the default value 140(1.4), Envoy doesn't consider a priority level
	// or a locality unhealthy until their percentage of healthy hosts drops
	// below 72%. For example:
	//
	// .. code-block:: json
	//
	//  { "overprovisioning_factor": 100 }
	//
	// Read more at :ref:`priority levels <arch_overview_load_balancing_priority_levels>` and
	// :ref:`localities <arch_overview_load_balancing_locality_weighted_lb>`.
	OverprovisioningFactor *types.UInt32Value `protobuf:"bytes,3,opt,name=overprovisioning_factor,json=overprovisioningFactor,proto3" json:"overprovisioning_factor,omitempty"`
	// The max time until which the endpoints from this assignment can be used.
	// If no new assignments are received before this time expires the endpoints
	// are considered stale and should be marked unhealthy.
	// Defaults to 0 which means endpoints never go stale.
	EndpointStaleAfter *types.Duration `protobuf:"bytes,4,opt,name=endpoint_stale_after,json=endpointStaleAfter,proto3" json:"endpoint_stale_after,omitempty"`
	// The flag to disable overprovisioning. If it is set to true,
	// :ref:`overprovisioning factor
	// <arch_overview_load_balancing_overprovisioning_factor>` will be ignored
	// and Envoy will not perform graceful failover between priority levels or
	// localities as endpoints become unhealthy. Otherwise Envoy will perform
	// graceful failover as :ref:`overprovisioning factor
	// <arch_overview_load_balancing_overprovisioning_factor>` suggests.
	// [#not-implemented-hide:]
	DisableOverprovisioning bool     `protobuf:"varint,5,opt,name=disable_overprovisioning,json=disableOverprovisioning,proto3" json:"disable_overprovisioning,omitempty"` // Deprecated: Do not use.
	XXX_NoUnkeyedLiteral    struct{} `json:"-"`
	XXX_unrecognized        []byte   `json:"-"`
	XXX_sizecache           int32    `json:"-"`
}

func (m *ClusterLoadAssignment_Policy) Reset()         { *m = ClusterLoadAssignment_Policy{} }
func (m *ClusterLoadAssignment_Policy) String() string { return proto.CompactTextString(m) }
func (*ClusterLoadAssignment_Policy) ProtoMessage()    {}
func (*ClusterLoadAssignment_Policy) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dd3a9d301e6758b, []int{0, 0}
}
func (m *ClusterLoadAssignment_Policy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ClusterLoadAssignment_Policy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ClusterLoadAssignment_Policy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ClusterLoadAssignment_Policy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ClusterLoadAssignment_Policy.Merge(m, src)
}
func (m *ClusterLoadAssignment_Policy) XXX_Size() int {
	return m.Size()
}
func (m *ClusterLoadAssignment_Policy) XXX_DiscardUnknown() {
	xxx_messageInfo_ClusterLoadAssignment_Policy.DiscardUnknown(m)
}

var xxx_messageInfo_ClusterLoadAssignment_Policy proto.InternalMessageInfo

func (m *ClusterLoadAssignment_Policy) GetDropOverloads() []*ClusterLoadAssignment_Policy_DropOverload {
	if m != nil {
		return m.DropOverloads
	}
	return nil
}

func (m *ClusterLoadAssignment_Policy) GetOverprovisioningFactor() *types.UInt32Value {
	if m != nil {
		return m.OverprovisioningFactor
	}
	return nil
}

func (m *ClusterLoadAssignment_Policy) GetEndpointStaleAfter() *types.Duration {
	if m != nil {
		return m.EndpointStaleAfter
	}
	return nil
}

// Deprecated: Do not use.
func (m *ClusterLoadAssignment_Policy) GetDisableOverprovisioning() bool {
	if m != nil {
		return m.DisableOverprovisioning
	}
	return false
}

type ClusterLoadAssignment_Policy_DropOverload struct {
	// Identifier for the policy specifying the drop.
	Category string `protobuf:"bytes,1,opt,name=category,proto3" json:"category,omitempty"`
	// Percentage of traffic that should be dropped for the category.
	DropPercentage       *_type.FractionalPercent `protobuf:"bytes,2,opt,name=drop_percentage,json=dropPercentage,proto3" json:"drop_percentage,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *ClusterLoadAssignment_Policy_DropOverload) Reset() {
	*m = ClusterLoadAssignment_Policy_DropOverload{}
}
func (m *ClusterLoadAssignment_Policy_DropOverload) String() string { return proto.CompactTextString(m) }
func (*ClusterLoadAssignment_Policy_DropOverload) ProtoMessage()    {}
func (*ClusterLoadAssignment_Policy_DropOverload) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dd3a9d301e6758b, []int{0, 0, 0}
}
func (m *ClusterLoadAssignment_Policy_DropOverload) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ClusterLoadAssignment_Policy_DropOverload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ClusterLoadAssignment_Policy_DropOverload.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ClusterLoadAssignment_Policy_DropOverload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ClusterLoadAssignment_Policy_DropOverload.Merge(m, src)
}
func (m *ClusterLoadAssignment_Policy_DropOverload) XXX_Size() int {
	return m.Size()
}
func (m *ClusterLoadAssignment_Policy_DropOverload) XXX_DiscardUnknown() {
	xxx_messageInfo_ClusterLoadAssignment_Policy_DropOverload.DiscardUnknown(m)
}

var xxx_messageInfo_ClusterLoadAssignment_Policy_DropOverload proto.InternalMessageInfo

func (m *ClusterLoadAssignment_Policy_DropOverload) GetCategory() string {
	if m != nil {
		return m.Category
	}
	return ""
}

func (m *ClusterLoadAssignment_Policy_DropOverload) GetDropPercentage() *_type.FractionalPercent {
	if m != nil {
		return m.DropPercentage
	}
	return nil
}

func init() {
	proto.RegisterType((*ClusterLoadAssignment)(nil), "envoy.api.v2.ClusterLoadAssignment")
	proto.RegisterMapType((map[string]*endpoint.Endpoint)(nil), "envoy.api.v2.ClusterLoadAssignment.NamedEndpointsEntry")
	proto.RegisterType((*ClusterLoadAssignment_Policy)(nil), "envoy.api.v2.ClusterLoadAssignment.Policy")
	proto.RegisterType((*ClusterLoadAssignment_Policy_DropOverload)(nil), "envoy.api.v2.ClusterLoadAssignment.Policy.DropOverload")
}

func init() { proto.RegisterFile("envoy/api/v2/endpoint.proto", fileDescriptor_8dd3a9d301e6758b) }

var fileDescriptor_8dd3a9d301e6758b = []byte{
	// 659 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x94, 0x4d, 0x6e, 0xd3, 0x40,
	0x14, 0xc7, 0x3b, 0x4e, 0x13, 0xd2, 0xe9, 0xa7, 0x86, 0x8f, 0x1a, 0xd3, 0x86, 0x08, 0x36, 0x55,
	0x16, 0xb6, 0x94, 0x0a, 0x15, 0x21, 0xb1, 0xa8, 0x69, 0x2b, 0x40, 0x15, 0x8d, 0x8c, 0xca, 0xb2,
	0x61, 0x62, 0x4f, 0xac, 0x11, 0xce, 0xcc, 0x68, 0x3c, 0x31, 0x58, 0x6c, 0xb8, 0x01, 0x5b, 0xce,
	0xc0, 0x11, 0x38, 0x41, 0x97, 0x70, 0x03, 0xe8, 0x11, 0x58, 0x16, 0x09, 0x21, 0x8f, 0x3f, 0x9a,
	0xa6, 0xa9, 0xc4, 0x6e, 0xec, 0xff, 0xfb, 0xff, 0xde, 0x9b, 0xf7, 0x9e, 0x0d, 0xef, 0x11, 0x96,
	0xf0, 0xd4, 0xc1, 0x82, 0x3a, 0x49, 0xd7, 0x21, 0x2c, 0x10, 0x9c, 0x32, 0x65, 0x0b, 0xc9, 0x15,
	0x47, 0x4b, 0x5a, 0xb4, 0xb1, 0xa0, 0x76, 0xd2, 0xb5, 0x9c, 0x99, 0xa1, 0xd5, 0xa1, 0xef, 0xf3,
	0x91, 0xe0, 0x8c, 0x30, 0x15, 0xe7, 0x76, 0xcb, 0xcc, 0x0d, 0x2a, 0x15, 0xc4, 0x11, 0x44, 0xfa,
	0xa4, 0x04, 0x5b, 0x1b, 0x21, 0xe7, 0x61, 0x44, 0x34, 0x0b, 0x33, 0xc6, 0x15, 0x56, 0x94, 0xb3,
	0xd2, 0xd7, 0x2a, 0x54, 0xfd, 0x34, 0x18, 0x0f, 0x9d, 0x60, 0x2c, 0x75, 0xc0, 0x75, 0xfa, 0x7b,
	0x89, 0x85, 0x20, 0xb2, 0xf2, 0x8f, 0x03, 0x81, 0x27, 0xb9, 0xce, 0x88, 0x86, 0x12, 0x2b, 0x52,
	0xe8, 0x9b, 0x57, 0xf4, 0x58, 0x61, 0x35, 0x2e, 0xed, 0xeb, 0x09, 0x8e, 0x68, 0x80, 0x15, 0x71,
	0xca, 0x43, 0x2e, 0x3c, 0xf8, 0xd3, 0x80, 0xb7, 0x9f, 0x45, 0xe3, 0x58, 0x11, 0x79, 0xc8, 0x71,
	0xb0, 0x1b, 0xc7, 0x34, 0x64, 0x23, 0xc2, 0x14, 0xea, 0xc0, 0x25, 0x3f, 0x17, 0xfa, 0x0c, 0x8f,
	0x88, 0x09, 0xda, 0x60, 0x6b, 0xc1, 0xbd, 0x71, 0xee, 0xce, 0x4b, 0xa3, 0x0d, 0xbc, 0xc5, 0x42,
	0x7c, 0x85, 0x47, 0x04, 0x3d, 0x87, 0x0b, 0x65, 0xcb, 0x62, 0xd3, 0x68, 0xd7, 0xb6, 0x16, 0xbb,
	0x1d, 0x7b, 0xb2, 0xd1, 0x76, 0x35, 0x85, 0x43, 0xee, 0xe3, 0x88, 0xaa, 0xf4, 0x70, 0xb0, 0x5f,
	0x3a, 0xbc, 0x0b, 0x33, 0x7a, 0x0b, 0x57, 0xb3, 0x6c, 0x41, 0xff, 0x82, 0x57, 0xd7, 0xbc, 0x9d,
	0xcb, 0xbc, 0x99, 0x35, 0xdb, 0x59, 0x31, 0x41, 0xc5, 0xdd, 0x67, 0x4a, 0xa6, 0xde, 0x0a, 0xbb,
	0xf4, 0x12, 0xb9, 0xb0, 0x21, 0x78, 0x44, 0xfd, 0xd4, 0x9c, 0x6f, 0x83, 0xab, 0x85, 0xce, 0x06,
	0xf7, 0xb4, 0xc3, 0x2b, 0x9c, 0xd6, 0xaf, 0x1a, 0x6c, 0xe4, 0xaf, 0xd0, 0x09, 0x5c, 0x09, 0x24,
	0x17, 0x7d, 0x9e, 0x10, 0x19, 0x71, 0x1c, 0x94, 0xf7, 0xdf, 0xf9, 0x7f, 0xac, 0xbd, 0x27, 0xb9,
	0x38, 0x2a, 0xfc, 0xde, 0x72, 0x30, 0xf1, 0x14, 0xa3, 0x13, 0xb8, 0x9e, 0xa1, 0x85, 0xe4, 0x09,
	0x8d, 0x29, 0x67, 0x94, 0x85, 0xfd, 0x21, 0xf6, 0x15, 0x97, 0x66, 0x4d, 0xd7, 0xbf, 0x61, 0xe7,
	0xab, 0x63, 0x97, 0xab, 0x63, 0x1f, 0xbf, 0x60, 0x6a, 0xbb, 0xfb, 0x06, 0x47, 0x63, 0xa2, 0xe7,
	0xd5, 0x31, 0xda, 0x73, 0xde, 0x9d, 0x69, 0xca, 0x81, 0x86, 0xa0, 0x63, 0x78, 0xab, 0xda, 0xf6,
	0x58, 0xe1, 0x88, 0xf4, 0xf1, 0x50, 0x11, 0x59, 0x34, 0xe7, 0xee, 0x15, 0xf8, 0x5e, 0xb1, 0xb7,
	0x6e, 0xf3, 0xdc, 0xad, 0x7f, 0x05, 0x46, 0x67, 0xce, 0x43, 0x25, 0xe0, 0x75, 0xe6, 0xdf, 0xcd,
	0xec, 0xe8, 0x29, 0x34, 0x03, 0x1a, 0xe3, 0x41, 0x44, 0xfa, 0xd3, 0x89, 0xcd, 0x7a, 0x1b, 0x6c,
	0x35, 0x5d, 0xc3, 0x04, 0xde, 0x7a, 0x11, 0x73, 0x34, 0x15, 0x62, 0x7d, 0x84, 0x4b, 0x93, 0x4d,
	0x41, 0x0f, 0x61, 0xd3, 0xc7, 0x8a, 0x84, 0x5c, 0xa6, 0xd3, 0x8b, 0x58, 0x09, 0xe8, 0x00, 0xae,
	0xea, 0x51, 0x14, 0xdf, 0x25, 0x0e, 0x89, 0x69, 0xe8, 0x5b, 0x6c, 0x16, 0xb3, 0xc8, 0xbe, 0x5a,
	0xfb, 0x40, 0x62, 0x3f, 0xbb, 0x00, 0x8e, 0x7a, 0x79, 0x9c, 0xa7, 0x07, 0xd8, 0xab, 0x4c, 0x2f,
	0xe7, 0x9b, 0x60, 0xcd, 0xb0, 0x06, 0xf0, 0xe6, 0x8c, 0x75, 0x42, 0x6b, 0xb0, 0xf6, 0x8e, 0x14,
	0x45, 0x78, 0xd9, 0x11, 0x3d, 0x82, 0xf5, 0x24, 0xeb, 0x75, 0x91, 0xec, 0xfe, 0x35, 0x8b, 0x5f,
	0x72, 0xbc, 0x3c, 0xfa, 0x89, 0xf1, 0x18, 0xb8, 0xf8, 0xf4, 0xac, 0x05, 0xbe, 0x9f, 0xb5, 0xc0,
	0xcf, 0xb3, 0x16, 0xf8, 0xfd, 0xe5, 0xef, 0xe7, 0xba, 0x85, 0xf2, 0x3f, 0x8c, 0xed, 0x73, 0x36,
	0xa4, 0xe1, 0x85, 0x3d, 0xd9, 0xfe, 0xf6, 0xe9, 0xf4, 0x47, 0xc3, 0x58, 0x9b, 0x83, 0x16, 0xe5,
	0x79, 0x0e, 0x21, 0xf9, 0x87, 0xf4, 0x52, 0x3a, 0x77, 0xb9, 0x4c, 0xd3, 0xcb, 0x06, 0xd6, 0x03,
	0x83, 0x86, 0x9e, 0xdc, 0xf6, 0xbf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xf9, 0x50, 0xd2, 0x9d, 0x15,
	0x05, 0x00, 0x00,
}

func (m *ClusterLoadAssignment) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ClusterLoadAssignment) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ClusterLoadAssignment) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.NamedEndpoints) > 0 {
		for k := range m.NamedEndpoints {
			v := m.NamedEndpoints[k]
			baseI := i
			if v != nil {
				{
					size, err := v.MarshalToSizedBuffer(dAtA[:i])
					if err != nil {
						return 0, err
					}
					i -= size
					i = encodeVarintEndpoint(dAtA, i, uint64(size))
				}
				i--
				dAtA[i] = 0x12
			}
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintEndpoint(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintEndpoint(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x2a
		}
	}
	if m.Policy != nil {
		{
			size, err := m.Policy.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEndpoint(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if len(m.Endpoints) > 0 {
		for iNdEx := len(m.Endpoints) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Endpoints[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintEndpoint(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.ClusterName) > 0 {
		i -= len(m.ClusterName)
		copy(dAtA[i:], m.ClusterName)
		i = encodeVarintEndpoint(dAtA, i, uint64(len(m.ClusterName)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ClusterLoadAssignment_Policy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ClusterLoadAssignment_Policy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ClusterLoadAssignment_Policy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.DisableOverprovisioning {
		i--
		if m.DisableOverprovisioning {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	if m.EndpointStaleAfter != nil {
		{
			size, err := m.EndpointStaleAfter.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEndpoint(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if m.OverprovisioningFactor != nil {
		{
			size, err := m.OverprovisioningFactor.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEndpoint(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if len(m.DropOverloads) > 0 {
		for iNdEx := len(m.DropOverloads) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.DropOverloads[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintEndpoint(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	return len(dAtA) - i, nil
}

func (m *ClusterLoadAssignment_Policy_DropOverload) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ClusterLoadAssignment_Policy_DropOverload) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ClusterLoadAssignment_Policy_DropOverload) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.DropPercentage != nil {
		{
			size, err := m.DropPercentage.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEndpoint(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Category) > 0 {
		i -= len(m.Category)
		copy(dAtA[i:], m.Category)
		i = encodeVarintEndpoint(dAtA, i, uint64(len(m.Category)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintEndpoint(dAtA []byte, offset int, v uint64) int {
	offset -= sovEndpoint(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ClusterLoadAssignment) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ClusterName)
	if l > 0 {
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if len(m.Endpoints) > 0 {
		for _, e := range m.Endpoints {
			l = e.Size()
			n += 1 + l + sovEndpoint(uint64(l))
		}
	}
	if m.Policy != nil {
		l = m.Policy.Size()
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if len(m.NamedEndpoints) > 0 {
		for k, v := range m.NamedEndpoints {
			_ = k
			_ = v
			l = 0
			if v != nil {
				l = v.Size()
				l += 1 + sovEndpoint(uint64(l))
			}
			mapEntrySize := 1 + len(k) + sovEndpoint(uint64(len(k))) + l
			n += mapEntrySize + 1 + sovEndpoint(uint64(mapEntrySize))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ClusterLoadAssignment_Policy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.DropOverloads) > 0 {
		for _, e := range m.DropOverloads {
			l = e.Size()
			n += 1 + l + sovEndpoint(uint64(l))
		}
	}
	if m.OverprovisioningFactor != nil {
		l = m.OverprovisioningFactor.Size()
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if m.EndpointStaleAfter != nil {
		l = m.EndpointStaleAfter.Size()
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if m.DisableOverprovisioning {
		n += 2
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ClusterLoadAssignment_Policy_DropOverload) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Category)
	if l > 0 {
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if m.DropPercentage != nil {
		l = m.DropPercentage.Size()
		n += 1 + l + sovEndpoint(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovEndpoint(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEndpoint(x uint64) (n int) {
	return sovEndpoint(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ClusterLoadAssignment) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEndpoint
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ClusterLoadAssignment: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ClusterLoadAssignment: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Endpoints", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Endpoints = append(m.Endpoints, &endpoint.LocalityLbEndpoints{})
			if err := m.Endpoints[len(m.Endpoints)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Policy", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Policy == nil {
				m.Policy = &ClusterLoadAssignment_Policy{}
			}
			if err := m.Policy.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NamedEndpoints", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.NamedEndpoints == nil {
				m.NamedEndpoints = make(map[string]*endpoint.Endpoint)
			}
			var mapkey string
			var mapvalue *endpoint.Endpoint
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowEndpoint
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowEndpoint
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthEndpoint
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthEndpoint
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var mapmsglen int
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowEndpoint
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapmsglen |= int(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					if mapmsglen < 0 {
						return ErrInvalidLengthEndpoint
					}
					postmsgIndex := iNdEx + mapmsglen
					if postmsgIndex < 0 {
						return ErrInvalidLengthEndpoint
					}
					if postmsgIndex > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = &endpoint.Endpoint{}
					if err := mapvalue.Unmarshal(dAtA[iNdEx:postmsgIndex]); err != nil {
						return err
					}
					iNdEx = postmsgIndex
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipEndpoint(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if skippy < 0 {
						return ErrInvalidLengthEndpoint
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.NamedEndpoints[mapkey] = mapvalue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEndpoint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ClusterLoadAssignment_Policy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEndpoint
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Policy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Policy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DropOverloads", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DropOverloads = append(m.DropOverloads, &ClusterLoadAssignment_Policy_DropOverload{})
			if err := m.DropOverloads[len(m.DropOverloads)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OverprovisioningFactor", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.OverprovisioningFactor == nil {
				m.OverprovisioningFactor = &types.UInt32Value{}
			}
			if err := m.OverprovisioningFactor.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndpointStaleAfter", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.EndpointStaleAfter == nil {
				m.EndpointStaleAfter = &types.Duration{}
			}
			if err := m.EndpointStaleAfter.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DisableOverprovisioning", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.DisableOverprovisioning = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipEndpoint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ClusterLoadAssignment_Policy_DropOverload) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEndpoint
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DropOverload: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DropOverload: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Category", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Category = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DropPercentage", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEndpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEndpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DropPercentage == nil {
				m.DropPercentage = &_type.FractionalPercent{}
			}
			if err := m.DropPercentage.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEndpoint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEndpoint
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipEndpoint(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEndpoint
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEndpoint
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthEndpoint
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEndpoint
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEndpoint
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEndpoint        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEndpoint          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEndpoint = fmt.Errorf("proto: unexpected end of group")
)
