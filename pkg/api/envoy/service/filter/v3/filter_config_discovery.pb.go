// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.22.0
// 	protoc        v3.10.1
// source: envoy/service/filter/v3/filter_config_discovery.proto

package envoy_service_filter_v3

import (
	context "context"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/annotations"
	v3 "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

var File_envoy_service_filter_v3_filter_config_discovery_proto protoreflect.FileDescriptor

var file_envoy_service_filter_v3_filter_config_discovery_proto_rawDesc = []byte{
	0x0a, 0x35, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f,
	0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x2f, 0x76, 0x33, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x33,
	0x1a, 0x2a, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f,
	0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2f, 0x76, 0x33, 0x2f, 0x64, 0x69, 0x73,
	0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1d, 0x75, 0x64,
	0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x75, 0x64, 0x70,
	0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32, 0xf2,
	0x03, 0x0a, 0x1c, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x44,
	0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x78, 0x0a, 0x13, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x12, 0x2c, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79,
	0x2e, 0x76, 0x33, 0x2e, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x76,
	0x33, 0x2e, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x12, 0x81, 0x01, 0x0a, 0x12, 0x44, 0x65,
	0x6c, 0x74, 0x61, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73,
	0x12, 0x31, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x33, 0x2e, 0x44, 0x65,
	0x6c, 0x74, 0x61, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x32, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x33,
	0x2e, 0x44, 0x65, 0x6c, 0x74, 0x61, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x12, 0xa0, 0x01,
	0x0a, 0x12, 0x46, 0x65, 0x74, 0x63, 0x68, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x73, 0x12, 0x2c, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x76,
	0x33, 0x2e, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x33, 0x2e,
	0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x2d, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1e, 0x22, 0x1c, 0x2f, 0x76, 0x33, 0x2f, 0x64,
	0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x3a, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x5f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x03, 0x3a, 0x01, 0x2a,
	0x1a, 0x31, 0x8a, 0xa4, 0x96, 0xf3, 0x07, 0x2b, 0x0a, 0x29, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x76, 0x33, 0x2e, 0x54,
	0x79, 0x70, 0x65, 0x64, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x42, 0x50, 0x0a, 0x25, 0x69, 0x6f, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x70,
	0x72, 0x6f, 0x78, 0x79, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x33, 0x42, 0x1a, 0x46, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76,
	0x65, 0x72, 0x79, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x88, 0x01, 0x01, 0xba, 0x80, 0xc8,
	0xd1, 0x06, 0x02, 0x10, 0x02, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_envoy_service_filter_v3_filter_config_discovery_proto_goTypes = []interface{}{
	(*v3.DiscoveryRequest)(nil),       // 0: envoy.service.discovery.v3.DiscoveryRequest
	(*v3.DeltaDiscoveryRequest)(nil),  // 1: envoy.service.discovery.v3.DeltaDiscoveryRequest
	(*v3.DiscoveryResponse)(nil),      // 2: envoy.service.discovery.v3.DiscoveryResponse
	(*v3.DeltaDiscoveryResponse)(nil), // 3: envoy.service.discovery.v3.DeltaDiscoveryResponse
}
var file_envoy_service_filter_v3_filter_config_discovery_proto_depIdxs = []int32{
	0, // 0: envoy.service.filter.v3.FilterConfigDiscoveryService.StreamFilterConfigs:input_type -> envoy.service.discovery.v3.DiscoveryRequest
	1, // 1: envoy.service.filter.v3.FilterConfigDiscoveryService.DeltaFilterConfigs:input_type -> envoy.service.discovery.v3.DeltaDiscoveryRequest
	0, // 2: envoy.service.filter.v3.FilterConfigDiscoveryService.FetchFilterConfigs:input_type -> envoy.service.discovery.v3.DiscoveryRequest
	2, // 3: envoy.service.filter.v3.FilterConfigDiscoveryService.StreamFilterConfigs:output_type -> envoy.service.discovery.v3.DiscoveryResponse
	3, // 4: envoy.service.filter.v3.FilterConfigDiscoveryService.DeltaFilterConfigs:output_type -> envoy.service.discovery.v3.DeltaDiscoveryResponse
	2, // 5: envoy.service.filter.v3.FilterConfigDiscoveryService.FetchFilterConfigs:output_type -> envoy.service.discovery.v3.DiscoveryResponse
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_envoy_service_filter_v3_filter_config_discovery_proto_init() }
func file_envoy_service_filter_v3_filter_config_discovery_proto_init() {
	if File_envoy_service_filter_v3_filter_config_discovery_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_envoy_service_filter_v3_filter_config_discovery_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_envoy_service_filter_v3_filter_config_discovery_proto_goTypes,
		DependencyIndexes: file_envoy_service_filter_v3_filter_config_discovery_proto_depIdxs,
	}.Build()
	File_envoy_service_filter_v3_filter_config_discovery_proto = out.File
	file_envoy_service_filter_v3_filter_config_discovery_proto_rawDesc = nil
	file_envoy_service_filter_v3_filter_config_discovery_proto_goTypes = nil
	file_envoy_service_filter_v3_filter_config_discovery_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// FilterConfigDiscoveryServiceClient is the client API for FilterConfigDiscoveryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FilterConfigDiscoveryServiceClient interface {
	StreamFilterConfigs(ctx context.Context, opts ...grpc.CallOption) (FilterConfigDiscoveryService_StreamFilterConfigsClient, error)
	DeltaFilterConfigs(ctx context.Context, opts ...grpc.CallOption) (FilterConfigDiscoveryService_DeltaFilterConfigsClient, error)
	FetchFilterConfigs(ctx context.Context, in *v3.DiscoveryRequest, opts ...grpc.CallOption) (*v3.DiscoveryResponse, error)
}

type filterConfigDiscoveryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFilterConfigDiscoveryServiceClient(cc grpc.ClientConnInterface) FilterConfigDiscoveryServiceClient {
	return &filterConfigDiscoveryServiceClient{cc}
}

func (c *filterConfigDiscoveryServiceClient) StreamFilterConfigs(ctx context.Context, opts ...grpc.CallOption) (FilterConfigDiscoveryService_StreamFilterConfigsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_FilterConfigDiscoveryService_serviceDesc.Streams[0], "/envoy.service.filter.v3.FilterConfigDiscoveryService/StreamFilterConfigs", opts...)
	if err != nil {
		return nil, err
	}
	x := &filterConfigDiscoveryServiceStreamFilterConfigsClient{stream}
	return x, nil
}

type FilterConfigDiscoveryService_StreamFilterConfigsClient interface {
	Send(*v3.DiscoveryRequest) error
	Recv() (*v3.DiscoveryResponse, error)
	grpc.ClientStream
}

type filterConfigDiscoveryServiceStreamFilterConfigsClient struct {
	grpc.ClientStream
}

func (x *filterConfigDiscoveryServiceStreamFilterConfigsClient) Send(m *v3.DiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *filterConfigDiscoveryServiceStreamFilterConfigsClient) Recv() (*v3.DiscoveryResponse, error) {
	m := new(v3.DiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *filterConfigDiscoveryServiceClient) DeltaFilterConfigs(ctx context.Context, opts ...grpc.CallOption) (FilterConfigDiscoveryService_DeltaFilterConfigsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_FilterConfigDiscoveryService_serviceDesc.Streams[1], "/envoy.service.filter.v3.FilterConfigDiscoveryService/DeltaFilterConfigs", opts...)
	if err != nil {
		return nil, err
	}
	x := &filterConfigDiscoveryServiceDeltaFilterConfigsClient{stream}
	return x, nil
}

type FilterConfigDiscoveryService_DeltaFilterConfigsClient interface {
	Send(*v3.DeltaDiscoveryRequest) error
	Recv() (*v3.DeltaDiscoveryResponse, error)
	grpc.ClientStream
}

type filterConfigDiscoveryServiceDeltaFilterConfigsClient struct {
	grpc.ClientStream
}

func (x *filterConfigDiscoveryServiceDeltaFilterConfigsClient) Send(m *v3.DeltaDiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *filterConfigDiscoveryServiceDeltaFilterConfigsClient) Recv() (*v3.DeltaDiscoveryResponse, error) {
	m := new(v3.DeltaDiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *filterConfigDiscoveryServiceClient) FetchFilterConfigs(ctx context.Context, in *v3.DiscoveryRequest, opts ...grpc.CallOption) (*v3.DiscoveryResponse, error) {
	out := new(v3.DiscoveryResponse)
	err := c.cc.Invoke(ctx, "/envoy.service.filter.v3.FilterConfigDiscoveryService/FetchFilterConfigs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FilterConfigDiscoveryServiceServer is the server API for FilterConfigDiscoveryService service.
type FilterConfigDiscoveryServiceServer interface {
	StreamFilterConfigs(FilterConfigDiscoveryService_StreamFilterConfigsServer) error
	DeltaFilterConfigs(FilterConfigDiscoveryService_DeltaFilterConfigsServer) error
	FetchFilterConfigs(context.Context, *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error)
}

// UnimplementedFilterConfigDiscoveryServiceServer can be embedded to have forward compatible implementations.
type UnimplementedFilterConfigDiscoveryServiceServer struct {
}

func (*UnimplementedFilterConfigDiscoveryServiceServer) StreamFilterConfigs(FilterConfigDiscoveryService_StreamFilterConfigsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamFilterConfigs not implemented")
}
func (*UnimplementedFilterConfigDiscoveryServiceServer) DeltaFilterConfigs(FilterConfigDiscoveryService_DeltaFilterConfigsServer) error {
	return status.Errorf(codes.Unimplemented, "method DeltaFilterConfigs not implemented")
}
func (*UnimplementedFilterConfigDiscoveryServiceServer) FetchFilterConfigs(context.Context, *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchFilterConfigs not implemented")
}

func RegisterFilterConfigDiscoveryServiceServer(s *grpc.Server, srv FilterConfigDiscoveryServiceServer) {
	s.RegisterService(&_FilterConfigDiscoveryService_serviceDesc, srv)
}

func _FilterConfigDiscoveryService_StreamFilterConfigs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(FilterConfigDiscoveryServiceServer).StreamFilterConfigs(&filterConfigDiscoveryServiceStreamFilterConfigsServer{stream})
}

type FilterConfigDiscoveryService_StreamFilterConfigsServer interface {
	Send(*v3.DiscoveryResponse) error
	Recv() (*v3.DiscoveryRequest, error)
	grpc.ServerStream
}

type filterConfigDiscoveryServiceStreamFilterConfigsServer struct {
	grpc.ServerStream
}

func (x *filterConfigDiscoveryServiceStreamFilterConfigsServer) Send(m *v3.DiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *filterConfigDiscoveryServiceStreamFilterConfigsServer) Recv() (*v3.DiscoveryRequest, error) {
	m := new(v3.DiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _FilterConfigDiscoveryService_DeltaFilterConfigs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(FilterConfigDiscoveryServiceServer).DeltaFilterConfigs(&filterConfigDiscoveryServiceDeltaFilterConfigsServer{stream})
}

type FilterConfigDiscoveryService_DeltaFilterConfigsServer interface {
	Send(*v3.DeltaDiscoveryResponse) error
	Recv() (*v3.DeltaDiscoveryRequest, error)
	grpc.ServerStream
}

type filterConfigDiscoveryServiceDeltaFilterConfigsServer struct {
	grpc.ServerStream
}

func (x *filterConfigDiscoveryServiceDeltaFilterConfigsServer) Send(m *v3.DeltaDiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *filterConfigDiscoveryServiceDeltaFilterConfigsServer) Recv() (*v3.DeltaDiscoveryRequest, error) {
	m := new(v3.DeltaDiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _FilterConfigDiscoveryService_FetchFilterConfigs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(v3.DiscoveryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FilterConfigDiscoveryServiceServer).FetchFilterConfigs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/envoy.service.filter.v3.FilterConfigDiscoveryService/FetchFilterConfigs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FilterConfigDiscoveryServiceServer).FetchFilterConfigs(ctx, req.(*v3.DiscoveryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _FilterConfigDiscoveryService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "envoy.service.filter.v3.FilterConfigDiscoveryService",
	HandlerType: (*FilterConfigDiscoveryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchFilterConfigs",
			Handler:    _FilterConfigDiscoveryService_FetchFilterConfigs_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamFilterConfigs",
			Handler:       _FilterConfigDiscoveryService_StreamFilterConfigs_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "DeltaFilterConfigs",
			Handler:       _FilterConfigDiscoveryService_DeltaFilterConfigs_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "envoy/service/filter/v3/filter_config_discovery.proto",
}