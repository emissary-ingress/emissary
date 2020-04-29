// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/config/filter/network/mysql_proxy/v1alpha1/mysql_proxy.proto

package envoy_config_filter_network_mysql_proxy_v1alpha1

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/gogo/protobuf/proto"
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

type MySQLProxy struct {
	// The human readable prefix to use when emitting :ref:`statistics
	// <config_network_filters_mysql_proxy_stats>`.
	StatPrefix string `protobuf:"bytes,1,opt,name=stat_prefix,json=statPrefix,proto3" json:"stat_prefix,omitempty"`
	// [#not-implemented-hide:] The optional path to use for writing MySQL access logs.
	// If the access log field is empty, access logs will not be written.
	AccessLog            string   `protobuf:"bytes,2,opt,name=access_log,json=accessLog,proto3" json:"access_log,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MySQLProxy) Reset()         { *m = MySQLProxy{} }
func (m *MySQLProxy) String() string { return proto.CompactTextString(m) }
func (*MySQLProxy) ProtoMessage()    {}
func (*MySQLProxy) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4bac5cccef760ed, []int{0}
}
func (m *MySQLProxy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MySQLProxy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MySQLProxy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MySQLProxy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MySQLProxy.Merge(m, src)
}
func (m *MySQLProxy) XXX_Size() int {
	return m.Size()
}
func (m *MySQLProxy) XXX_DiscardUnknown() {
	xxx_messageInfo_MySQLProxy.DiscardUnknown(m)
}

var xxx_messageInfo_MySQLProxy proto.InternalMessageInfo

func (m *MySQLProxy) GetStatPrefix() string {
	if m != nil {
		return m.StatPrefix
	}
	return ""
}

func (m *MySQLProxy) GetAccessLog() string {
	if m != nil {
		return m.AccessLog
	}
	return ""
}

func init() {
	proto.RegisterType((*MySQLProxy)(nil), "envoy.config.filter.network.mysql_proxy.v1alpha1.MySQLProxy")
}

func init() {
	proto.RegisterFile("envoy/config/filter/network/mysql_proxy/v1alpha1/mysql_proxy.proto", fileDescriptor_c4bac5cccef760ed)
}

var fileDescriptor_c4bac5cccef760ed = []byte{
	// 284 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x90, 0xbd, 0x4e, 0xc3, 0x30,
	0x14, 0x85, 0xe5, 0x8a, 0x1f, 0xd5, 0x0c, 0x48, 0x59, 0xa8, 0x2a, 0x11, 0x55, 0x4c, 0x9d, 0x6c,
	0xaa, 0xee, 0x0c, 0x99, 0x5b, 0x29, 0x14, 0x31, 0x47, 0x26, 0x75, 0x82, 0x45, 0xea, 0x1b, 0xec,
	0x4b, 0x48, 0x9e, 0x82, 0x0d, 0xf1, 0x48, 0x8c, 0x3c, 0x02, 0xca, 0x23, 0x30, 0x32, 0x20, 0x64,
	0xbb, 0x95, 0x3a, 0xb0, 0xb0, 0x59, 0xe7, 0x1c, 0x7f, 0xf2, 0x67, 0x9a, 0x48, 0xdd, 0x40, 0xc7,
	0x73, 0xd0, 0x85, 0x2a, 0x79, 0xa1, 0x2a, 0x94, 0x86, 0x6b, 0x89, 0xcf, 0x60, 0x1e, 0xf8, 0xa6,
	0xb3, 0x8f, 0x55, 0x56, 0x1b, 0x68, 0x3b, 0xde, 0xcc, 0x44, 0x55, 0xdf, 0x8b, 0xd9, 0x7e, 0xc8,
	0x6a, 0x03, 0x08, 0xd1, 0xa5, 0x67, 0xb0, 0xc0, 0x60, 0x81, 0xc1, 0xb6, 0x0c, 0xb6, 0x3f, 0xdf,
	0x31, 0xc6, 0xf1, 0xd3, 0xba, 0x16, 0x5c, 0x68, 0x0d, 0x28, 0x50, 0x81, 0xb6, 0x7c, 0xa3, 0x4a,
	0x23, 0x50, 0x06, 0xe2, 0xf8, 0xac, 0x11, 0x95, 0x5a, 0x0b, 0x94, 0x7c, 0x77, 0x08, 0xc5, 0xc5,
	0x2d, 0xa5, 0xcb, 0xee, 0xe6, 0x7a, 0x91, 0x3a, 0x5e, 0x34, 0xa5, 0x27, 0x16, 0x05, 0x66, 0xb5,
	0x91, 0x85, 0x6a, 0x47, 0x64, 0x42, 0xa6, 0xc3, 0xe4, 0xf8, 0x3b, 0x39, 0x30, 0x83, 0x09, 0x59,
	0x51, 0xd7, 0xa5, 0xbe, 0x8a, 0xce, 0x29, 0x15, 0x79, 0x2e, 0xad, 0xcd, 0x2a, 0x28, 0x47, 0x03,
	0x37, 0x5c, 0x0d, 0x43, 0xb2, 0x80, 0x32, 0x79, 0x25, 0xef, 0x7d, 0x4c, 0x3e, 0xfa, 0x98, 0x7c,
	0xf6, 0x31, 0xf9, 0x7a, 0xfb, 0x79, 0x39, 0x9c, 0x45, 0x3c, 0x68, 0xc9, 0x16, 0xa5, 0xb6, 0xee,
	0x91, 0x5b, 0x35, 0xfb, 0xb7, 0xdb, 0x9c, 0x5e, 0x29, 0x60, 0xfe, 0x4e, 0x48, 0xfe, 0xfb, 0x2b,
	0xc9, 0xe9, 0xd2, 0xa5, 0x5e, 0x2d, 0x75, 0xb6, 0x29, 0xb9, 0x3b, 0xf2, 0xda, 0xf3, 0xdf, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xcd, 0xf4, 0x67, 0x17, 0xa7, 0x01, 0x00, 0x00,
}

func (m *MySQLProxy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MySQLProxy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MySQLProxy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.AccessLog) > 0 {
		i -= len(m.AccessLog)
		copy(dAtA[i:], m.AccessLog)
		i = encodeVarintMysqlProxy(dAtA, i, uint64(len(m.AccessLog)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.StatPrefix) > 0 {
		i -= len(m.StatPrefix)
		copy(dAtA[i:], m.StatPrefix)
		i = encodeVarintMysqlProxy(dAtA, i, uint64(len(m.StatPrefix)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMysqlProxy(dAtA []byte, offset int, v uint64) int {
	offset -= sovMysqlProxy(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MySQLProxy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.StatPrefix)
	if l > 0 {
		n += 1 + l + sovMysqlProxy(uint64(l))
	}
	l = len(m.AccessLog)
	if l > 0 {
		n += 1 + l + sovMysqlProxy(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovMysqlProxy(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMysqlProxy(x uint64) (n int) {
	return sovMysqlProxy(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MySQLProxy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMysqlProxy
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
			return fmt.Errorf("proto: MySQLProxy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MySQLProxy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StatPrefix", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMysqlProxy
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
				return ErrInvalidLengthMysqlProxy
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMysqlProxy
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StatPrefix = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AccessLog", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMysqlProxy
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
				return ErrInvalidLengthMysqlProxy
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMysqlProxy
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AccessLog = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMysqlProxy(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMysqlProxy
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMysqlProxy
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
func skipMysqlProxy(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMysqlProxy
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
					return 0, ErrIntOverflowMysqlProxy
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
					return 0, ErrIntOverflowMysqlProxy
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
				return 0, ErrInvalidLengthMysqlProxy
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMysqlProxy
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMysqlProxy
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMysqlProxy        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMysqlProxy          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMysqlProxy = fmt.Errorf("proto: unexpected end of group")
)
