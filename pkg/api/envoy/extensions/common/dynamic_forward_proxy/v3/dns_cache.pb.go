// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/extensions/common/dynamic_forward_proxy/v3/dns_cache.proto

package envoy_extensions_common_dynamic_forward_proxy_v3

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	v3 "github.com/datawire/ambassador/pkg/api/envoy/config/cluster/v3"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	io "io"
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

// Configuration for the dynamic forward proxy DNS cache. See the :ref:`architecture overview
// <arch_overview_http_dynamic_forward_proxy>` for more information.
// [#next-free-field: 7]
type DnsCacheConfig struct {
	// The name of the cache. Multiple named caches allow independent dynamic forward proxy
	// configurations to operate within a single Envoy process using different configurations. All
	// configurations with the same name *must* otherwise have the same settings when referenced
	// from different configuration components. Configuration will fail to load if this is not
	// the case.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The DNS lookup family to use during resolution.
	//
	// [#comment:TODO(mattklein123): Figure out how to support IPv4/IPv6 "happy eyeballs" mode. The
	// way this might work is a new lookup family which returns both IPv4 and IPv6 addresses, and
	// then configures a host to have a primary and fall back address. With this, we could very
	// likely build a "happy eyeballs" connection pool which would race the primary / fall back
	// address and return the one that wins. This same method could potentially also be used for
	// QUIC to TCP fall back.]
	DnsLookupFamily v3.Cluster_DnsLookupFamily `protobuf:"varint,2,opt,name=dns_lookup_family,json=dnsLookupFamily,proto3,enum=envoy.config.cluster.v3.Cluster_DnsLookupFamily" json:"dns_lookup_family,omitempty"`
	// The DNS refresh rate for currently cached DNS hosts. If not specified defaults to 60s.
	//
	// .. note:
	//
	//  The returned DNS TTL is not currently used to alter the refresh rate. This feature will be
	//  added in a future change.
	//
	// .. note:
	//
	// The refresh rate is rounded to the closest millisecond, and must be at least 1ms.
	DnsRefreshRate *types.Duration `protobuf:"bytes,3,opt,name=dns_refresh_rate,json=dnsRefreshRate,proto3" json:"dns_refresh_rate,omitempty"`
	// The TTL for hosts that are unused. Hosts that have not been used in the configured time
	// interval will be purged. If not specified defaults to 5m.
	//
	// .. note:
	//
	//   The TTL is only checked at the time of DNS refresh, as specified by *dns_refresh_rate*. This
	//   means that if the configured TTL is shorter than the refresh rate the host may not be removed
	//   immediately.
	//
	//  .. note:
	//
	//   The TTL has no relation to DNS TTL and is only used to control Envoy's resource usage.
	HostTtl *types.Duration `protobuf:"bytes,4,opt,name=host_ttl,json=hostTtl,proto3" json:"host_ttl,omitempty"`
	// The maximum number of hosts that the cache will hold. If not specified defaults to 1024.
	//
	// .. note:
	//
	//   The implementation is approximate and enforced independently on each worker thread, thus
	//   it is possible for the maximum hosts in the cache to go slightly above the configured
	//   value depending on timing. This is similar to how other circuit breakers work.
	MaxHosts *types.UInt32Value `protobuf:"bytes,5,opt,name=max_hosts,json=maxHosts,proto3" json:"max_hosts,omitempty"`
	// If the DNS failure refresh rate is specified,
	// this is used as the cache's DNS refresh rate when DNS requests are failing. If this setting is
	// not specified, the failure refresh rate defaults to the dns_refresh_rate.
	DnsFailureRefreshRate *v3.Cluster_RefreshRate `protobuf:"bytes,6,opt,name=dns_failure_refresh_rate,json=dnsFailureRefreshRate,proto3" json:"dns_failure_refresh_rate,omitempty"`
	XXX_NoUnkeyedLiteral  struct{}                `json:"-"`
	XXX_unrecognized      []byte                  `json:"-"`
	XXX_sizecache         int32                   `json:"-"`
}

func (m *DnsCacheConfig) Reset()         { *m = DnsCacheConfig{} }
func (m *DnsCacheConfig) String() string { return proto.CompactTextString(m) }
func (*DnsCacheConfig) ProtoMessage()    {}
func (*DnsCacheConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_f57893b6dd868364, []int{0}
}
func (m *DnsCacheConfig) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DnsCacheConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DnsCacheConfig.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DnsCacheConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DnsCacheConfig.Merge(m, src)
}
func (m *DnsCacheConfig) XXX_Size() int {
	return m.Size()
}
func (m *DnsCacheConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_DnsCacheConfig.DiscardUnknown(m)
}

var xxx_messageInfo_DnsCacheConfig proto.InternalMessageInfo

func (m *DnsCacheConfig) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DnsCacheConfig) GetDnsLookupFamily() v3.Cluster_DnsLookupFamily {
	if m != nil {
		return m.DnsLookupFamily
	}
	return v3.Cluster_AUTO
}

func (m *DnsCacheConfig) GetDnsRefreshRate() *types.Duration {
	if m != nil {
		return m.DnsRefreshRate
	}
	return nil
}

func (m *DnsCacheConfig) GetHostTtl() *types.Duration {
	if m != nil {
		return m.HostTtl
	}
	return nil
}

func (m *DnsCacheConfig) GetMaxHosts() *types.UInt32Value {
	if m != nil {
		return m.MaxHosts
	}
	return nil
}

func (m *DnsCacheConfig) GetDnsFailureRefreshRate() *v3.Cluster_RefreshRate {
	if m != nil {
		return m.DnsFailureRefreshRate
	}
	return nil
}

func init() {
	proto.RegisterType((*DnsCacheConfig)(nil), "envoy.extensions.common.dynamic_forward_proxy.v3.DnsCacheConfig")
}

func init() {
	proto.RegisterFile("envoy/extensions/common/dynamic_forward_proxy/v3/dns_cache.proto", fileDescriptor_f57893b6dd868364)
}

var fileDescriptor_f57893b6dd868364 = []byte{
	// 535 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x93, 0xc1, 0x6e, 0xd3, 0x30,
	0x18, 0xc7, 0xe7, 0xd0, 0x75, 0x6d, 0x80, 0x52, 0x22, 0x21, 0xc2, 0x80, 0xaa, 0x20, 0x21, 0x55,
	0x13, 0xb2, 0xa7, 0xf6, 0x86, 0xc4, 0x84, 0xd2, 0x6a, 0x80, 0xc4, 0x61, 0x44, 0xc0, 0x35, 0xf2,
	0x12, 0xa7, 0x8d, 0x48, 0xec, 0xc8, 0x76, 0xb2, 0xf6, 0x86, 0x10, 0x07, 0x9e, 0x01, 0xf1, 0x04,
	0x7b, 0x04, 0x4e, 0x5c, 0x90, 0x76, 0x84, 0x37, 0x40, 0x7d, 0x8c, 0x9e, 0x90, 0xed, 0x04, 0x56,
	0x0d, 0x98, 0x76, 0x73, 0xfc, 0xe5, 0xff, 0xfb, 0xfc, 0xfd, 0x64, 0xdb, 0x4f, 0x08, 0x2d, 0xd9,
	0x02, 0x91, 0xb9, 0x24, 0x54, 0x24, 0x8c, 0x0a, 0x14, 0xb2, 0x2c, 0x63, 0x14, 0x45, 0x0b, 0x8a,
	0xb3, 0x24, 0x0c, 0x62, 0xc6, 0x8f, 0x30, 0x8f, 0x82, 0x9c, 0xb3, 0xf9, 0x02, 0x95, 0x23, 0x14,
	0x51, 0x11, 0x84, 0x38, 0x9c, 0x11, 0x98, 0x73, 0x26, 0x99, 0xb3, 0xab, 0x09, 0xf0, 0x0f, 0x01,
	0x1a, 0x02, 0xfc, 0x2b, 0x01, 0x96, 0xa3, 0xed, 0x07, 0xa6, 0x67, 0xc8, 0x68, 0x9c, 0x4c, 0x51,
	0x98, 0x16, 0x42, 0x12, 0xae, 0xd0, 0xd5, 0xd2, 0x80, 0xb7, 0x7b, 0x53, 0xc6, 0xa6, 0x29, 0x41,
	0xfa, 0xeb, 0xb0, 0x88, 0x51, 0x54, 0x70, 0x2c, 0x13, 0x46, 0xff, 0x55, 0x3f, 0xe2, 0x38, 0xcf,
	0x09, 0x17, 0x55, 0xfd, 0x6e, 0x11, 0xe5, 0x18, 0x61, 0x4a, 0x99, 0xd4, 0x31, 0x81, 0x84, 0xc4,
	0xb2, 0xa8, 0xcb, 0xf7, 0xce, 0x94, 0x4b, 0xc2, 0xd5, 0x00, 0x09, 0x9d, 0x56, 0xbf, 0xdc, 0x2c,
	0x71, 0x9a, 0x44, 0x58, 0x12, 0x54, 0x2f, 0x4c, 0xe1, 0xfe, 0xe7, 0x86, 0xdd, 0x99, 0x50, 0x31,
	0x56, 0x1a, 0xc6, 0x7a, 0x0c, 0xe7, 0xb6, 0xdd, 0xa0, 0x38, 0x23, 0x2e, 0xe8, 0x83, 0x41, 0xdb,
	0xdb, 0x5a, 0x79, 0x0d, 0x6e, 0xf5, 0x81, 0xaf, 0x37, 0x9d, 0xd8, 0xbe, 0xae, 0xb4, 0xa5, 0x8c,
	0xbd, 0x2d, 0xf2, 0x20, 0xc6, 0x59, 0x92, 0x2e, 0x5c, 0xab, 0x0f, 0x06, 0x9d, 0xe1, 0x2e, 0x34,
	0xfe, 0x8c, 0x0d, 0x58, 0x2b, 0x28, 0x47, 0x70, 0x5c, 0x2d, 0x27, 0x54, 0xbc, 0xd0, 0xc1, 0x7d,
	0x9d, 0xf3, 0x5a, 0x2b, 0x6f, 0xf3, 0x3d, 0xb0, 0xba, 0xc0, 0xbf, 0x16, 0xad, 0x97, 0x9c, 0x97,
	0x76, 0x57, 0xf5, 0xe1, 0x24, 0xe6, 0x44, 0xcc, 0x02, 0x8e, 0x25, 0x71, 0x2f, 0xf5, 0xc1, 0xe0,
	0xf2, 0xf0, 0x16, 0x34, 0xb6, 0x60, 0x6d, 0x0b, 0x4e, 0x2a, 0x9b, 0xde, 0x95, 0x95, 0xd7, 0x3e,
	0x06, 0xcd, 0x61, 0xa3, 0xfb, 0xf5, 0xc3, 0x63, 0xbf, 0x13, 0x51, 0xe1, 0x9b, 0xbc, 0x8f, 0x25,
	0x71, 0xf6, 0xec, 0xd6, 0x8c, 0x09, 0x19, 0x48, 0x99, 0xba, 0x8d, 0xf3, 0x50, 0xea, 0x68, 0xc7,
	0xc0, 0xda, 0xd9, 0xf0, 0xb7, 0x54, 0xe8, 0x95, 0x4c, 0x1d, 0xcf, 0x6e, 0x67, 0x78, 0x1e, 0xa8,
	0x4f, 0xe1, 0x6e, 0x6a, 0xc0, 0x9d, 0x33, 0x80, 0xd7, 0xcf, 0xa9, 0x1c, 0x0d, 0xdf, 0xe0, 0xb4,
	0x20, 0x5a, 0xdd, 0x8e, 0xd5, 0xdf, 0xf0, 0x5b, 0x19, 0x9e, 0x3f, 0x53, 0x31, 0x87, 0xd8, 0xae,
	0x1a, 0x2b, 0xc6, 0x49, 0x5a, 0x70, 0xb2, 0x3e, 0x5e, 0x53, 0x23, 0x1f, 0x9e, 0x6b, 0xf1, 0xd4,
	0x4c, 0xfe, 0x8d, 0x88, 0x8a, 0x7d, 0x03, 0x3b, 0xb5, 0xfd, 0xe8, 0xe9, 0xa7, 0x6f, 0x1f, 0x7b,
	0x5e, 0xf5, 0x24, 0x7e, 0xa3, 0xfe, 0x7b, 0x99, 0x87, 0x38, 0xcd, 0x67, 0x18, 0xae, 0xdf, 0x05,
	0x8f, 0x9c, 0x2c, 0x7b, 0xe0, 0xfb, 0xb2, 0x07, 0x7e, 0x2e, 0x7b, 0xe0, 0xcb, 0xbb, 0x93, 0x1f,
	0x4d, 0xab, 0x0b, 0xec, 0xbd, 0x84, 0x99, 0x53, 0x9a, 0xec, 0x45, 0x9f, 0x8d, 0x77, 0xb5, 0xee,
	0x72, 0xa0, 0xb4, 0x1d, 0x80, 0xc3, 0xa6, 0xf6, 0x37, 0xfa, 0x15, 0x00, 0x00, 0xff, 0xff, 0xe3,
	0x22, 0x61, 0x62, 0xc4, 0x03, 0x00, 0x00,
}

func (m *DnsCacheConfig) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DnsCacheConfig) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DnsCacheConfig) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.DnsFailureRefreshRate != nil {
		{
			size, err := m.DnsFailureRefreshRate.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDnsCache(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x32
	}
	if m.MaxHosts != nil {
		{
			size, err := m.MaxHosts.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDnsCache(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x2a
	}
	if m.HostTtl != nil {
		{
			size, err := m.HostTtl.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDnsCache(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if m.DnsRefreshRate != nil {
		{
			size, err := m.DnsRefreshRate.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDnsCache(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if m.DnsLookupFamily != 0 {
		i = encodeVarintDnsCache(dAtA, i, uint64(m.DnsLookupFamily))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintDnsCache(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintDnsCache(dAtA []byte, offset int, v uint64) int {
	offset -= sovDnsCache(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DnsCacheConfig) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovDnsCache(uint64(l))
	}
	if m.DnsLookupFamily != 0 {
		n += 1 + sovDnsCache(uint64(m.DnsLookupFamily))
	}
	if m.DnsRefreshRate != nil {
		l = m.DnsRefreshRate.Size()
		n += 1 + l + sovDnsCache(uint64(l))
	}
	if m.HostTtl != nil {
		l = m.HostTtl.Size()
		n += 1 + l + sovDnsCache(uint64(l))
	}
	if m.MaxHosts != nil {
		l = m.MaxHosts.Size()
		n += 1 + l + sovDnsCache(uint64(l))
	}
	if m.DnsFailureRefreshRate != nil {
		l = m.DnsFailureRefreshRate.Size()
		n += 1 + l + sovDnsCache(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovDnsCache(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDnsCache(x uint64) (n int) {
	return sovDnsCache(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DnsCacheConfig) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDnsCache
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
			return fmt.Errorf("proto: DnsCacheConfig: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DnsCacheConfig: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
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
				return ErrInvalidLengthDnsCache
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDnsCache
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DnsLookupFamily", wireType)
			}
			m.DnsLookupFamily = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DnsLookupFamily |= v3.Cluster_DnsLookupFamily(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DnsRefreshRate", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
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
				return ErrInvalidLengthDnsCache
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDnsCache
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DnsRefreshRate == nil {
				m.DnsRefreshRate = &types.Duration{}
			}
			if err := m.DnsRefreshRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field HostTtl", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
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
				return ErrInvalidLengthDnsCache
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDnsCache
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.HostTtl == nil {
				m.HostTtl = &types.Duration{}
			}
			if err := m.HostTtl.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxHosts", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
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
				return ErrInvalidLengthDnsCache
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDnsCache
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.MaxHosts == nil {
				m.MaxHosts = &types.UInt32Value{}
			}
			if err := m.MaxHosts.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DnsFailureRefreshRate", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDnsCache
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
				return ErrInvalidLengthDnsCache
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDnsCache
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DnsFailureRefreshRate == nil {
				m.DnsFailureRefreshRate = &v3.Cluster_RefreshRate{}
			}
			if err := m.DnsFailureRefreshRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDnsCache(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDnsCache
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthDnsCache
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
func skipDnsCache(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDnsCache
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
					return 0, ErrIntOverflowDnsCache
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
					return 0, ErrIntOverflowDnsCache
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
				return 0, ErrInvalidLengthDnsCache
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDnsCache
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDnsCache
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDnsCache        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDnsCache          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDnsCache = fmt.Errorf("proto: unexpected end of group")
)
