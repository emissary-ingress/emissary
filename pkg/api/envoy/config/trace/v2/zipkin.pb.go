// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.7
// source: envoy/config/trace/v2/zipkin.proto

package tracev2

import (
	_ "github.com/cncf/xds/go/udpa/annotations"
	_ "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/annotations"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Available Zipkin collector endpoint versions.
type ZipkinConfig_CollectorEndpointVersion int32

const (
	// Zipkin API v1, JSON over HTTP.
	// [#comment: The default implementation of Zipkin client before this field is added was only v1
	// and the way user configure this was by not explicitly specifying the version. Consequently,
	// before this is added, the corresponding Zipkin collector expected to receive v1 payload.
	// Hence the motivation of adding HTTP_JSON_V1 as the default is to avoid a breaking change when
	// user upgrading Envoy with this change. Furthermore, we also immediately deprecate this field,
	// since in Zipkin realm this v1 version is considered to be not preferable anymore.]
	//
	// Deprecated: Do not use.
	ZipkinConfig_HTTP_JSON_V1 ZipkinConfig_CollectorEndpointVersion = 0
	// Zipkin API v2, JSON over HTTP.
	ZipkinConfig_HTTP_JSON ZipkinConfig_CollectorEndpointVersion = 1
	// Zipkin API v2, protobuf over HTTP.
	ZipkinConfig_HTTP_PROTO ZipkinConfig_CollectorEndpointVersion = 2
	// [#not-implemented-hide:]
	ZipkinConfig_GRPC ZipkinConfig_CollectorEndpointVersion = 3
)

// Enum value maps for ZipkinConfig_CollectorEndpointVersion.
var (
	ZipkinConfig_CollectorEndpointVersion_name = map[int32]string{
		0: "HTTP_JSON_V1",
		1: "HTTP_JSON",
		2: "HTTP_PROTO",
		3: "GRPC",
	}
	ZipkinConfig_CollectorEndpointVersion_value = map[string]int32{
		"HTTP_JSON_V1": 0,
		"HTTP_JSON":    1,
		"HTTP_PROTO":   2,
		"GRPC":         3,
	}
)

func (x ZipkinConfig_CollectorEndpointVersion) Enum() *ZipkinConfig_CollectorEndpointVersion {
	p := new(ZipkinConfig_CollectorEndpointVersion)
	*p = x
	return p
}

func (x ZipkinConfig_CollectorEndpointVersion) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ZipkinConfig_CollectorEndpointVersion) Descriptor() protoreflect.EnumDescriptor {
	return file_envoy_config_trace_v2_zipkin_proto_enumTypes[0].Descriptor()
}

func (ZipkinConfig_CollectorEndpointVersion) Type() protoreflect.EnumType {
	return &file_envoy_config_trace_v2_zipkin_proto_enumTypes[0]
}

func (x ZipkinConfig_CollectorEndpointVersion) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ZipkinConfig_CollectorEndpointVersion.Descriptor instead.
func (ZipkinConfig_CollectorEndpointVersion) EnumDescriptor() ([]byte, []int) {
	return file_envoy_config_trace_v2_zipkin_proto_rawDescGZIP(), []int{0, 0}
}

// Configuration for the Zipkin tracer.
// [#extension: envoy.tracers.zipkin]
// [#next-free-field: 6]
type ZipkinConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The cluster manager cluster that hosts the Zipkin collectors. Note that the
	// Zipkin cluster must be defined in the :ref:`Bootstrap static cluster
	// resources <envoy_api_field_config.bootstrap.v2.Bootstrap.StaticResources.clusters>`.
	CollectorCluster string `protobuf:"bytes,1,opt,name=collector_cluster,json=collectorCluster,proto3" json:"collector_cluster,omitempty"`
	// The API endpoint of the Zipkin service where the spans will be sent. When
	// using a standard Zipkin installation, the API endpoint is typically
	// /api/v1/spans, which is the default value.
	CollectorEndpoint string `protobuf:"bytes,2,opt,name=collector_endpoint,json=collectorEndpoint,proto3" json:"collector_endpoint,omitempty"`
	// Determines whether a 128bit trace id will be used when creating a new
	// trace instance. The default value is false, which will result in a 64 bit trace id being used.
	TraceId_128Bit bool `protobuf:"varint,3,opt,name=trace_id_128bit,json=traceId128bit,proto3" json:"trace_id_128bit,omitempty"`
	// Determines whether client and server spans will share the same span context.
	// The default value is true.
	SharedSpanContext *wrappers.BoolValue `protobuf:"bytes,4,opt,name=shared_span_context,json=sharedSpanContext,proto3" json:"shared_span_context,omitempty"`
	// Determines the selected collector endpoint version. By default, the ``HTTP_JSON_V1`` will be
	// used.
	CollectorEndpointVersion ZipkinConfig_CollectorEndpointVersion `protobuf:"varint,5,opt,name=collector_endpoint_version,json=collectorEndpointVersion,proto3,enum=envoy.config.trace.v2.ZipkinConfig_CollectorEndpointVersion" json:"collector_endpoint_version,omitempty"`
}

func (x *ZipkinConfig) Reset() {
	*x = ZipkinConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_config_trace_v2_zipkin_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ZipkinConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ZipkinConfig) ProtoMessage() {}

func (x *ZipkinConfig) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_config_trace_v2_zipkin_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ZipkinConfig.ProtoReflect.Descriptor instead.
func (*ZipkinConfig) Descriptor() ([]byte, []int) {
	return file_envoy_config_trace_v2_zipkin_proto_rawDescGZIP(), []int{0}
}

func (x *ZipkinConfig) GetCollectorCluster() string {
	if x != nil {
		return x.CollectorCluster
	}
	return ""
}

func (x *ZipkinConfig) GetCollectorEndpoint() string {
	if x != nil {
		return x.CollectorEndpoint
	}
	return ""
}

func (x *ZipkinConfig) GetTraceId_128Bit() bool {
	if x != nil {
		return x.TraceId_128Bit
	}
	return false
}

func (x *ZipkinConfig) GetSharedSpanContext() *wrappers.BoolValue {
	if x != nil {
		return x.SharedSpanContext
	}
	return nil
}

func (x *ZipkinConfig) GetCollectorEndpointVersion() ZipkinConfig_CollectorEndpointVersion {
	if x != nil {
		return x.CollectorEndpointVersion
	}
	return ZipkinConfig_HTTP_JSON_V1
}

var File_envoy_config_trace_v2_zipkin_proto protoreflect.FileDescriptor

var file_envoy_config_trace_v2_zipkin_proto_rawDesc = []byte{
	0x0a, 0x22, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x74,
	0x72, 0x61, 0x63, 0x65, 0x2f, 0x76, 0x32, 0x2f, 0x7a, 0x69, 0x70, 0x6b, 0x69, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x76, 0x32, 0x1a, 0x1e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61,
	0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x23, 0x65, 0x6e, 0x76,
	0x6f, 0x79, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x64,
	0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1d, 0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xcd, 0x03, 0x0a, 0x0c, 0x5a, 0x69, 0x70,
	0x6b, 0x69, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x34, 0x0a, 0x11, 0x63, 0x6f, 0x6c,
	0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x72, 0x02, 0x20, 0x01, 0x52, 0x10, 0x63,
	0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12,
	0x36, 0x0a, 0x12, 0x63, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x5f, 0x65, 0x6e, 0x64,
	0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04,
	0x72, 0x02, 0x20, 0x01, 0x52, 0x11, 0x63, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x45,
	0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x12, 0x26, 0x0a, 0x0f, 0x74, 0x72, 0x61, 0x63, 0x65,
	0x5f, 0x69, 0x64, 0x5f, 0x31, 0x32, 0x38, 0x62, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x0d, 0x74, 0x72, 0x61, 0x63, 0x65, 0x49, 0x64, 0x31, 0x32, 0x38, 0x62, 0x69, 0x74, 0x12,
	0x4a, 0x0a, 0x13, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x73, 0x70, 0x61, 0x6e, 0x5f, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42,
	0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x11, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64,
	0x53, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x7a, 0x0a, 0x1a, 0x63,
	0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x5f, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x3c, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x74,
	0x72, 0x61, 0x63, 0x65, 0x2e, 0x76, 0x32, 0x2e, 0x5a, 0x69, 0x70, 0x6b, 0x69, 0x6e, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x45, 0x6e,
	0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x52, 0x18, 0x63,
	0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74,
	0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x5f, 0x0a, 0x18, 0x43, 0x6f, 0x6c, 0x6c, 0x65,
	0x63, 0x74, 0x6f, 0x72, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x0c, 0x48, 0x54, 0x54, 0x50, 0x5f, 0x4a, 0x53, 0x4f, 0x4e,
	0x5f, 0x56, 0x31, 0x10, 0x00, 0x1a, 0x08, 0x08, 0x01, 0xa8, 0xf7, 0xb4, 0x8b, 0x02, 0x01, 0x12,
	0x0d, 0x0a, 0x09, 0x48, 0x54, 0x54, 0x50, 0x5f, 0x4a, 0x53, 0x4f, 0x4e, 0x10, 0x01, 0x12, 0x0e,
	0x0a, 0x0a, 0x48, 0x54, 0x54, 0x50, 0x5f, 0x50, 0x52, 0x4f, 0x54, 0x4f, 0x10, 0x02, 0x12, 0x08,
	0x0a, 0x04, 0x47, 0x52, 0x50, 0x43, 0x10, 0x03, 0x42, 0x82, 0x01, 0x0a, 0x23, 0x69, 0x6f, 0x2e,
	0x65, 0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79,
	0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x76, 0x32,
	0x42, 0x0b, 0x5a, 0x69, 0x70, 0x6b, 0x69, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a,
	0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2f, 0x67, 0x6f, 0x2d, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f,
	0x6c, 0x2d, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2f, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2f, 0x76, 0x32, 0x3b, 0x74, 0x72,
	0x61, 0x63, 0x65, 0x76, 0x32, 0xba, 0x80, 0xc8, 0xd1, 0x06, 0x02, 0x10, 0x01, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_envoy_config_trace_v2_zipkin_proto_rawDescOnce sync.Once
	file_envoy_config_trace_v2_zipkin_proto_rawDescData = file_envoy_config_trace_v2_zipkin_proto_rawDesc
)

func file_envoy_config_trace_v2_zipkin_proto_rawDescGZIP() []byte {
	file_envoy_config_trace_v2_zipkin_proto_rawDescOnce.Do(func() {
		file_envoy_config_trace_v2_zipkin_proto_rawDescData = protoimpl.X.CompressGZIP(file_envoy_config_trace_v2_zipkin_proto_rawDescData)
	})
	return file_envoy_config_trace_v2_zipkin_proto_rawDescData
}

var file_envoy_config_trace_v2_zipkin_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_envoy_config_trace_v2_zipkin_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_envoy_config_trace_v2_zipkin_proto_goTypes = []interface{}{
	(ZipkinConfig_CollectorEndpointVersion)(0), // 0: envoy.config.trace.v2.ZipkinConfig.CollectorEndpointVersion
	(*ZipkinConfig)(nil),                       // 1: envoy.config.trace.v2.ZipkinConfig
	(*wrappers.BoolValue)(nil),                 // 2: google.protobuf.BoolValue
}
var file_envoy_config_trace_v2_zipkin_proto_depIdxs = []int32{
	2, // 0: envoy.config.trace.v2.ZipkinConfig.shared_span_context:type_name -> google.protobuf.BoolValue
	0, // 1: envoy.config.trace.v2.ZipkinConfig.collector_endpoint_version:type_name -> envoy.config.trace.v2.ZipkinConfig.CollectorEndpointVersion
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_envoy_config_trace_v2_zipkin_proto_init() }
func file_envoy_config_trace_v2_zipkin_proto_init() {
	if File_envoy_config_trace_v2_zipkin_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_envoy_config_trace_v2_zipkin_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ZipkinConfig); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_envoy_config_trace_v2_zipkin_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_envoy_config_trace_v2_zipkin_proto_goTypes,
		DependencyIndexes: file_envoy_config_trace_v2_zipkin_proto_depIdxs,
		EnumInfos:         file_envoy_config_trace_v2_zipkin_proto_enumTypes,
		MessageInfos:      file_envoy_config_trace_v2_zipkin_proto_msgTypes,
	}.Build()
	File_envoy_config_trace_v2_zipkin_proto = out.File
	file_envoy_config_trace_v2_zipkin_proto_rawDesc = nil
	file_envoy_config_trace_v2_zipkin_proto_goTypes = nil
	file_envoy_config_trace_v2_zipkin_proto_depIdxs = nil
}
