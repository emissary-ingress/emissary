// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/config/filter/http/dynamo/v2/dynamo.proto

package envoy_config_filter_http_dynamo_v2

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
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

// Dynamo filter config.
type Dynamo struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Dynamo) Reset()         { *m = Dynamo{} }
func (m *Dynamo) String() string { return proto.CompactTextString(m) }
func (*Dynamo) ProtoMessage()    {}
func (*Dynamo) Descriptor() ([]byte, []int) {
	return fileDescriptor_37ee82f4c86cc210, []int{0}
}
func (m *Dynamo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Dynamo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Dynamo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Dynamo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Dynamo.Merge(m, src)
}
func (m *Dynamo) XXX_Size() int {
	return m.Size()
}
func (m *Dynamo) XXX_DiscardUnknown() {
	xxx_messageInfo_Dynamo.DiscardUnknown(m)
}

var xxx_messageInfo_Dynamo proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Dynamo)(nil), "envoy.config.filter.http.dynamo.v2.Dynamo")
}

func init() {
	proto.RegisterFile("envoy/config/filter/http/dynamo/v2/dynamo.proto", fileDescriptor_37ee82f4c86cc210)
}

var fileDescriptor_37ee82f4c86cc210 = []byte{
	// 212 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0x4f, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x4f, 0xce, 0xcf, 0x4b, 0xcb, 0x4c, 0xd7, 0x4f, 0xcb, 0xcc, 0x29, 0x49, 0x2d, 0xd2,
	0xcf, 0x28, 0x29, 0x29, 0xd0, 0x4f, 0xa9, 0xcc, 0x4b, 0xcc, 0xcd, 0xd7, 0x2f, 0x33, 0x82, 0xb2,
	0xf4, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0x94, 0xc0, 0x1a, 0xf4, 0x20, 0x1a, 0xf4, 0x20, 0x1a,
	0xf4, 0x40, 0x1a, 0xf4, 0xa0, 0xca, 0xca, 0x8c, 0xa4, 0xe4, 0x4a, 0x53, 0x0a, 0x12, 0xf5, 0x13,
	0xf3, 0xf2, 0xf2, 0x4b, 0x12, 0x4b, 0x32, 0xf3, 0xf3, 0x8a, 0xf5, 0x73, 0x33, 0xd3, 0x8b, 0x12,
	0x4b, 0x52, 0x21, 0x66, 0x48, 0xc9, 0x62, 0xc8, 0x17, 0x97, 0x24, 0x96, 0x94, 0x16, 0x43, 0xa4,
	0x95, 0x38, 0xb8, 0xd8, 0x5c, 0xc0, 0x66, 0x39, 0xb5, 0x30, 0x9e, 0x78, 0x24, 0xc7, 0x78, 0xe1,
	0x91, 0x1c, 0xe3, 0x83, 0x47, 0x72, 0x8c, 0x9f, 0x66, 0xfc, 0xeb, 0x67, 0xd5, 0x14, 0x52, 0x87,
	0xb8, 0x20, 0xb5, 0xa2, 0x24, 0x35, 0xaf, 0x18, 0xa4, 0x1b, 0xea, 0x8a, 0x62, 0x54, 0x67, 0x18,
	0xef, 0x6a, 0x38, 0x71, 0x91, 0x8d, 0x49, 0x80, 0x81, 0xcb, 0x20, 0x33, 0x5f, 0x0f, 0xac, 0xa7,
	0xa0, 0x28, 0xbf, 0xa2, 0x52, 0x8f, 0xb0, 0x07, 0x9c, 0xb8, 0x21, 0xf6, 0x07, 0x80, 0x9c, 0x13,
	0xc0, 0x98, 0xc4, 0x06, 0x76, 0x97, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xae, 0x34, 0xd7, 0x52,
	0x2d, 0x01, 0x00, 0x00,
}

func (m *Dynamo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Dynamo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Dynamo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	return len(dAtA) - i, nil
}

func encodeVarintDynamo(dAtA []byte, offset int, v uint64) int {
	offset -= sovDynamo(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Dynamo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovDynamo(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDynamo(x uint64) (n int) {
	return sovDynamo(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Dynamo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDynamo
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
			return fmt.Errorf("proto: Dynamo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Dynamo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipDynamo(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDynamo
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthDynamo
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
func skipDynamo(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDynamo
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
					return 0, ErrIntOverflowDynamo
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
					return 0, ErrIntOverflowDynamo
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
				return 0, ErrInvalidLengthDynamo
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDynamo
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDynamo
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDynamo        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDynamo          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDynamo = fmt.Errorf("proto: unexpected end of group")
)
