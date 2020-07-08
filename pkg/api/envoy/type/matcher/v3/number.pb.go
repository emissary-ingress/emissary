// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/type/matcher/v3/number.proto

package envoy_type_matcher_v3

import (
	encoding_binary "encoding/binary"
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	v3 "github.com/datawire/ambassador/pkg/api/envoy/type/v3"
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

// Specifies the way to match a double value.
type DoubleMatcher struct {
	// Types that are valid to be assigned to MatchPattern:
	//	*DoubleMatcher_Range
	//	*DoubleMatcher_Exact
	MatchPattern         isDoubleMatcher_MatchPattern `protobuf_oneof:"match_pattern"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *DoubleMatcher) Reset()         { *m = DoubleMatcher{} }
func (m *DoubleMatcher) String() string { return proto.CompactTextString(m) }
func (*DoubleMatcher) ProtoMessage()    {}
func (*DoubleMatcher) Descriptor() ([]byte, []int) {
	return fileDescriptor_9de077d68a31b59d, []int{0}
}
func (m *DoubleMatcher) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DoubleMatcher) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DoubleMatcher.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DoubleMatcher) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DoubleMatcher.Merge(m, src)
}
func (m *DoubleMatcher) XXX_Size() int {
	return m.Size()
}
func (m *DoubleMatcher) XXX_DiscardUnknown() {
	xxx_messageInfo_DoubleMatcher.DiscardUnknown(m)
}

var xxx_messageInfo_DoubleMatcher proto.InternalMessageInfo

type isDoubleMatcher_MatchPattern interface {
	isDoubleMatcher_MatchPattern()
	MarshalTo([]byte) (int, error)
	Size() int
}

type DoubleMatcher_Range struct {
	Range *v3.DoubleRange `protobuf:"bytes,1,opt,name=range,proto3,oneof" json:"range,omitempty"`
}
type DoubleMatcher_Exact struct {
	Exact float64 `protobuf:"fixed64,2,opt,name=exact,proto3,oneof" json:"exact,omitempty"`
}

func (*DoubleMatcher_Range) isDoubleMatcher_MatchPattern() {}
func (*DoubleMatcher_Exact) isDoubleMatcher_MatchPattern() {}

func (m *DoubleMatcher) GetMatchPattern() isDoubleMatcher_MatchPattern {
	if m != nil {
		return m.MatchPattern
	}
	return nil
}

func (m *DoubleMatcher) GetRange() *v3.DoubleRange {
	if x, ok := m.GetMatchPattern().(*DoubleMatcher_Range); ok {
		return x.Range
	}
	return nil
}

func (m *DoubleMatcher) GetExact() float64 {
	if x, ok := m.GetMatchPattern().(*DoubleMatcher_Exact); ok {
		return x.Exact
	}
	return 0
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*DoubleMatcher) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*DoubleMatcher_Range)(nil),
		(*DoubleMatcher_Exact)(nil),
	}
}

func init() {
	proto.RegisterType((*DoubleMatcher)(nil), "envoy.type.matcher.v3.DoubleMatcher")
}

func init() { proto.RegisterFile("envoy/type/matcher/v3/number.proto", fileDescriptor_9de077d68a31b59d) }

var fileDescriptor_9de077d68a31b59d = []byte{
	// 293 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x4a, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x2f, 0xa9, 0x2c, 0x48, 0xd5, 0xcf, 0x4d, 0x2c, 0x49, 0xce, 0x48, 0x2d, 0xd2, 0x2f,
	0x33, 0xd6, 0xcf, 0x2b, 0xcd, 0x4d, 0x4a, 0x2d, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12,
	0x05, 0xab, 0xd1, 0x03, 0xa9, 0xd1, 0x83, 0xaa, 0xd1, 0x2b, 0x33, 0x96, 0x92, 0x44, 0xd2, 0x5a,
	0x66, 0xac, 0x5f, 0x94, 0x98, 0x97, 0x9e, 0x0a, 0xd1, 0x21, 0x25, 0x5b, 0x9a, 0x52, 0x90, 0xa8,
	0x9f, 0x98, 0x97, 0x97, 0x5f, 0x92, 0x58, 0x92, 0x99, 0x9f, 0x57, 0xac, 0x5f, 0x5c, 0x92, 0x58,
	0x52, 0x5a, 0x0c, 0x95, 0x56, 0xc4, 0x90, 0x2e, 0x4b, 0x2d, 0x2a, 0xce, 0xcc, 0xcf, 0xcb, 0xcc,
	0x4b, 0x87, 0x2a, 0x11, 0x2f, 0x4b, 0xcc, 0xc9, 0x4c, 0x49, 0x2c, 0x49, 0xd5, 0x87, 0x31, 0x20,
	0x12, 0x4a, 0xb3, 0x18, 0xb9, 0x78, 0x5d, 0xf2, 0x4b, 0x93, 0x72, 0x52, 0x7d, 0x21, 0x4e, 0x11,
	0x32, 0xe2, 0x62, 0x05, 0xdb, 0x2d, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1, 0x6d, 0x24, 0xa5, 0x87, 0xe4,
	0xdc, 0x32, 0x63, 0x3d, 0x88, 0xe2, 0x20, 0x90, 0x0a, 0x0f, 0x86, 0x20, 0x88, 0x52, 0x21, 0x31,
	0x2e, 0xd6, 0xd4, 0x8a, 0xc4, 0xe4, 0x12, 0x09, 0x26, 0x05, 0x46, 0x0d, 0x46, 0x90, 0x38, 0x98,
	0x6b, 0xa5, 0x3e, 0xeb, 0x68, 0x87, 0x9c, 0x12, 0x97, 0x02, 0x16, 0x1f, 0xa3, 0x58, 0xea, 0x24,
	0xc2, 0xc5, 0x0b, 0x96, 0x88, 0x2f, 0x48, 0x2c, 0x29, 0x49, 0x2d, 0xca, 0x13, 0x62, 0xfe, 0xe1,
	0xc4, 0xe8, 0xe4, 0x71, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31,
	0xee, 0x6a, 0x38, 0x71, 0x91, 0x8d, 0x49, 0x80, 0x91, 0x4b, 0x39, 0x33, 0x1f, 0xe2, 0xa6, 0x82,
	0xa2, 0xfc, 0x8a, 0x4a, 0x3d, 0xac, 0xa1, 0xe9, 0xc4, 0xed, 0x07, 0x0e, 0xf2, 0x00, 0x90, 0x27,
	0x03, 0x18, 0x93, 0xd8, 0xc0, 0xbe, 0x35, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x7e, 0x07, 0x36,
	0x6d, 0xa0, 0x01, 0x00, 0x00,
}

func (m *DoubleMatcher) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DoubleMatcher) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DoubleMatcher) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.MatchPattern != nil {
		{
			size := m.MatchPattern.Size()
			i -= size
			if _, err := m.MatchPattern.MarshalTo(dAtA[i:]); err != nil {
				return 0, err
			}
		}
	}
	return len(dAtA) - i, nil
}

func (m *DoubleMatcher_Range) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DoubleMatcher_Range) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.Range != nil {
		{
			size, err := m.Range.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintNumber(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}
func (m *DoubleMatcher_Exact) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DoubleMatcher_Exact) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= 8
	encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(math.Float64bits(float64(m.Exact))))
	i--
	dAtA[i] = 0x11
	return len(dAtA) - i, nil
}
func encodeVarintNumber(dAtA []byte, offset int, v uint64) int {
	offset -= sovNumber(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DoubleMatcher) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.MatchPattern != nil {
		n += m.MatchPattern.Size()
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *DoubleMatcher_Range) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Range != nil {
		l = m.Range.Size()
		n += 1 + l + sovNumber(uint64(l))
	}
	return n
}
func (m *DoubleMatcher_Exact) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 9
	return n
}

func sovNumber(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozNumber(x uint64) (n int) {
	return sovNumber(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DoubleMatcher) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowNumber
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
			return fmt.Errorf("proto: DoubleMatcher: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DoubleMatcher: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Range", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNumber
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
				return ErrInvalidLengthNumber
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthNumber
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := &v3.DoubleRange{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.MatchPattern = &DoubleMatcher_Range{v}
			iNdEx = postIndex
		case 2:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Exact", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint64(encoding_binary.LittleEndian.Uint64(dAtA[iNdEx:]))
			iNdEx += 8
			m.MatchPattern = &DoubleMatcher_Exact{float64(math.Float64frombits(v))}
		default:
			iNdEx = preIndex
			skippy, err := skipNumber(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthNumber
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthNumber
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
func skipNumber(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowNumber
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
					return 0, ErrIntOverflowNumber
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
					return 0, ErrIntOverflowNumber
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
				return 0, ErrInvalidLengthNumber
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupNumber
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthNumber
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthNumber        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowNumber          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupNumber = fmt.Errorf("proto: unexpected end of group")
)
