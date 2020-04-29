// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/extensions/filters/http/dynamo/v3/dynamo.proto

package envoy_extensions_filters_http_dynamo_v3

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
	return fileDescriptor_79057240c5b18ac4, []int{0}
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
	proto.RegisterType((*Dynamo)(nil), "envoy.extensions.filters.http.dynamo.v3.Dynamo")
}

func init() {
	proto.RegisterFile("envoy/extensions/filters/http/dynamo/v3/dynamo.proto", fileDescriptor_79057240c5b18ac4)
}

var fileDescriptor_79057240c5b18ac4 = []byte{
	// 195 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x32, 0x49, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x4f, 0xad, 0x28, 0x49, 0xcd, 0x2b, 0xce, 0xcc, 0xcf, 0x2b, 0xd6, 0x4f, 0xcb, 0xcc,
	0x29, 0x49, 0x2d, 0x2a, 0xd6, 0xcf, 0x28, 0x29, 0x29, 0xd0, 0x4f, 0xa9, 0xcc, 0x4b, 0xcc, 0xcd,
	0xd7, 0x2f, 0x33, 0x86, 0xb2, 0xf4, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0xd4, 0xc1, 0xba, 0xf4,
	0x10, 0xba, 0xf4, 0xa0, 0xba, 0xf4, 0x40, 0xba, 0xf4, 0xa0, 0x6a, 0xcb, 0x8c, 0xa5, 0x14, 0x4b,
	0x53, 0x0a, 0x12, 0xf5, 0x13, 0xf3, 0xf2, 0xf2, 0x4b, 0x12, 0x4b, 0xc0, 0xc6, 0x97, 0xa5, 0x16,
	0x81, 0x74, 0x64, 0xe6, 0xa5, 0x43, 0xcc, 0x52, 0xb2, 0xe2, 0x62, 0x73, 0x01, 0xab, 0xb7, 0x32,
	0x98, 0x75, 0xb4, 0x43, 0x4e, 0x9b, 0x4b, 0x13, 0x62, 0x78, 0x72, 0x7e, 0x5e, 0x5a, 0x66, 0x3a,
	0xd4, 0x60, 0x54, 0x73, 0x8d, 0xf4, 0x20, 0x3a, 0x9c, 0x82, 0x4e, 0x3c, 0x92, 0x63, 0xbc, 0xf0,
	0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0x46, 0x2e, 0xd3, 0xcc, 0x7c, 0x3d, 0xb0, 0xde, 0x82, 0xa2,
	0xfc, 0x8a, 0x4a, 0x3d, 0x22, 0xdd, 0xe8, 0xc4, 0x0d, 0x31, 0x2c, 0x00, 0xe4, 0x9a, 0x00, 0xc6,
	0x24, 0x36, 0xb0, 0xb3, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x73, 0x35, 0x83, 0x97, 0x1a,
	0x01, 0x00, 0x00,
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
