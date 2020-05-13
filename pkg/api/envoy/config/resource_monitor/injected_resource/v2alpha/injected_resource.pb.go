// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: envoy/config/resource_monitor/injected_resource/v2alpha/injected_resource.proto

package envoy_config_resource_monitor_injected_resource_v2alpha

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

// The injected resource monitor allows injecting a synthetic resource pressure into Envoy
// via a text file, which must contain a floating-point number in the range [0..1] representing
// the resource pressure and be updated atomically by a symbolic link swap.
// This is intended primarily for integration tests to force Envoy into an overloaded state.
type InjectedResourceConfig struct {
	Filename             string   `protobuf:"bytes,1,opt,name=filename,proto3" json:"filename,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *InjectedResourceConfig) Reset()         { *m = InjectedResourceConfig{} }
func (m *InjectedResourceConfig) String() string { return proto.CompactTextString(m) }
func (*InjectedResourceConfig) ProtoMessage()    {}
func (*InjectedResourceConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_de2fb4e1cfb2f415, []int{0}
}
func (m *InjectedResourceConfig) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InjectedResourceConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_InjectedResourceConfig.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *InjectedResourceConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InjectedResourceConfig.Merge(m, src)
}
func (m *InjectedResourceConfig) XXX_Size() int {
	return m.Size()
}
func (m *InjectedResourceConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_InjectedResourceConfig.DiscardUnknown(m)
}

var xxx_messageInfo_InjectedResourceConfig proto.InternalMessageInfo

func (m *InjectedResourceConfig) GetFilename() string {
	if m != nil {
		return m.Filename
	}
	return ""
}

func init() {
	proto.RegisterType((*InjectedResourceConfig)(nil), "envoy.config.resource_monitor.injected_resource.v2alpha.InjectedResourceConfig")
}

func init() {
	proto.RegisterFile("envoy/config/resource_monitor/injected_resource/v2alpha/injected_resource.proto", fileDescriptor_de2fb4e1cfb2f415)
}

var fileDescriptor_de2fb4e1cfb2f415 = []byte{
	// 238 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xf2, 0x4f, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x4f, 0xce, 0xcf, 0x4b, 0xcb, 0x4c, 0xd7, 0x2f, 0x4a, 0x2d, 0xce, 0x2f, 0x2d, 0x4a,
	0x4e, 0x8d, 0xcf, 0xcd, 0xcf, 0xcb, 0x2c, 0xc9, 0x2f, 0xd2, 0xcf, 0xcc, 0xcb, 0x4a, 0x4d, 0x2e,
	0x49, 0x4d, 0x89, 0x87, 0xc9, 0xe8, 0x97, 0x19, 0x25, 0xe6, 0x14, 0x64, 0x24, 0x62, 0xca, 0xe8,
	0x15, 0x14, 0xe5, 0x97, 0xe4, 0x0b, 0x99, 0x83, 0x0d, 0xd4, 0x83, 0x18, 0xa8, 0x87, 0x6e, 0xa0,
	0x1e, 0xa6, 0x36, 0xa8, 0x81, 0x52, 0xb2, 0xa5, 0x29, 0x05, 0x89, 0xfa, 0x89, 0x79, 0x79, 0xf9,
	0x25, 0x89, 0x25, 0x99, 0xf9, 0x79, 0xc5, 0xfa, 0xc5, 0x25, 0x89, 0x25, 0xa5, 0xc5, 0x10, 0x73,
	0xa5, 0xc4, 0xcb, 0x12, 0x73, 0x32, 0x53, 0x12, 0x4b, 0x52, 0xf5, 0x61, 0x0c, 0x88, 0x84, 0x92,
	0x2d, 0x97, 0x98, 0x27, 0xd4, 0xd0, 0x20, 0xa8, 0x99, 0xce, 0x60, 0xcb, 0x85, 0x94, 0xb9, 0x38,
	0xd2, 0x32, 0x73, 0x52, 0xf3, 0x12, 0x73, 0x53, 0x25, 0x18, 0x15, 0x18, 0x35, 0x38, 0x9d, 0xd8,
	0x7f, 0x39, 0xb1, 0x14, 0x31, 0x29, 0x30, 0x06, 0xc1, 0x25, 0x9c, 0x4a, 0x4e, 0x3c, 0x92, 0x63,
	0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x5d, 0x0d, 0x27, 0x2e, 0xb2, 0x31, 0x09,
	0x30, 0x72, 0xb9, 0x66, 0xe6, 0xeb, 0x81, 0x3d, 0x52, 0x50, 0x94, 0x5f, 0x51, 0xa9, 0x47, 0xa6,
	0x9f, 0x9c, 0x44, 0xd1, 0x5d, 0x16, 0x00, 0x72, 0x72, 0x00, 0x63, 0x12, 0x1b, 0xd8, 0xed, 0xc6,
	0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xc9, 0xc9, 0x58, 0x0b, 0x7f, 0x01, 0x00, 0x00,
}

func (m *InjectedResourceConfig) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InjectedResourceConfig) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InjectedResourceConfig) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Filename) > 0 {
		i -= len(m.Filename)
		copy(dAtA[i:], m.Filename)
		i = encodeVarintInjectedResource(dAtA, i, uint64(len(m.Filename)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintInjectedResource(dAtA []byte, offset int, v uint64) int {
	offset -= sovInjectedResource(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *InjectedResourceConfig) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Filename)
	if l > 0 {
		n += 1 + l + sovInjectedResource(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovInjectedResource(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozInjectedResource(x uint64) (n int) {
	return sovInjectedResource(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *InjectedResourceConfig) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowInjectedResource
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
			return fmt.Errorf("proto: InjectedResourceConfig: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InjectedResourceConfig: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Filename", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInjectedResource
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
				return ErrInvalidLengthInjectedResource
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthInjectedResource
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Filename = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipInjectedResource(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthInjectedResource
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthInjectedResource
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
func skipInjectedResource(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowInjectedResource
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
					return 0, ErrIntOverflowInjectedResource
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
					return 0, ErrIntOverflowInjectedResource
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
				return 0, ErrInvalidLengthInjectedResource
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupInjectedResource
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthInjectedResource
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthInjectedResource        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowInjectedResource          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupInjectedResource = fmt.Errorf("proto: unexpected end of group")
)
