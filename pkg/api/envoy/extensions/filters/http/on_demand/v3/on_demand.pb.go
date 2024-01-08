// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.1
// source: envoy/extensions/filters/http/on_demand/v3/on_demand.proto

package on_demandv3

import (
	_ "github.com/cncf/xds/go/udpa/annotations"
	v3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	duration "github.com/golang/protobuf/ptypes/duration"
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

// Configuration of on-demand CDS.
type OnDemandCds struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A configuration source for the service that will be used for
	// on-demand cluster discovery.
	Source *v3.ConfigSource `protobuf:"bytes,1,opt,name=source,proto3" json:"source,omitempty"`
	// xdstp:// resource locator for on-demand cluster collection.
	ResourcesLocator string `protobuf:"bytes,2,opt,name=resources_locator,json=resourcesLocator,proto3" json:"resources_locator,omitempty"`
	// The timeout for on demand cluster lookup. If not set, defaults to 5 seconds.
	Timeout *duration.Duration `protobuf:"bytes,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
}

func (x *OnDemandCds) Reset() {
	*x = OnDemandCds{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OnDemandCds) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OnDemandCds) ProtoMessage() {}

func (x *OnDemandCds) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OnDemandCds.ProtoReflect.Descriptor instead.
func (*OnDemandCds) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescGZIP(), []int{0}
}

func (x *OnDemandCds) GetSource() *v3.ConfigSource {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *OnDemandCds) GetResourcesLocator() string {
	if x != nil {
		return x.ResourcesLocator
	}
	return ""
}

func (x *OnDemandCds) GetTimeout() *duration.Duration {
	if x != nil {
		return x.Timeout
	}
	return nil
}

// On Demand Discovery filter config.
type OnDemand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// An optional configuration for on-demand cluster discovery
	// service. If not specified, the on-demand cluster discovery will
	// be disabled. When it's specified, the filter will pause the
	// request to an unknown cluster and will begin a cluster discovery
	// process. When the discovery is finished (successfully or not), the
	// request will be resumed for further processing.
	Odcds *OnDemandCds `protobuf:"bytes,1,opt,name=odcds,proto3" json:"odcds,omitempty"`
}

func (x *OnDemand) Reset() {
	*x = OnDemand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OnDemand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OnDemand) ProtoMessage() {}

func (x *OnDemand) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OnDemand.ProtoReflect.Descriptor instead.
func (*OnDemand) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescGZIP(), []int{1}
}

func (x *OnDemand) GetOdcds() *OnDemandCds {
	if x != nil {
		return x.Odcds
	}
	return nil
}

// Per-route configuration for On Demand Discovery.
type PerRouteConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// An optional configuration for on-demand cluster discovery
	// service. If not specified, the on-demand cluster discovery will
	// be disabled. When it's specified, the filter will pause the
	// request to an unknown cluster and will begin a cluster discovery
	// process. When the discovery is finished (successfully or not), the
	// request will be resumed for further processing.
	Odcds *OnDemandCds `protobuf:"bytes,1,opt,name=odcds,proto3" json:"odcds,omitempty"`
}

func (x *PerRouteConfig) Reset() {
	*x = PerRouteConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PerRouteConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PerRouteConfig) ProtoMessage() {}

func (x *PerRouteConfig) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PerRouteConfig.ProtoReflect.Descriptor instead.
func (*PerRouteConfig) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescGZIP(), []int{2}
}

func (x *PerRouteConfig) GetOdcds() *OnDemandCds {
	if x != nil {
		return x.Odcds
	}
	return nil
}

var File_envoy_extensions_filters_http_on_demand_v3_on_demand_proto protoreflect.FileDescriptor

var file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDesc = []byte{
	0x0a, 0x3a, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f,
	0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2f, 0x76, 0x33, 0x2f, 0x6f, 0x6e, 0x5f,
	0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x2a, 0x65, 0x6e,
	0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x6f, 0x6e, 0x5f, 0x64,
	0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2e, 0x76, 0x33, 0x1a, 0x28, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x76, 0x33, 0x2f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1d, 0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x21, 0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb5, 0x01,
	0x0a, 0x0b, 0x4f, 0x6e, 0x44, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x43, 0x64, 0x73, 0x12, 0x44, 0x0a,
	0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e,
	0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x63, 0x6f, 0x72,
	0x65, 0x2e, 0x76, 0x33, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x53, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x06, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x12, 0x2b, 0x0a, 0x11, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x5f, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10,
	0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x6f, 0x72,
	0x12, 0x33, 0x0a, 0x07, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x07, 0x74, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x22, 0x90, 0x01, 0x0a, 0x08, 0x4f, 0x6e, 0x44, 0x65, 0x6d, 0x61,
	0x6e, 0x64, 0x12, 0x4d, 0x0a, 0x05, 0x6f, 0x64, 0x63, 0x64, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x37, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74,
	0x70, 0x2e, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2e, 0x76, 0x33, 0x2e, 0x4f,
	0x6e, 0x44, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x43, 0x64, 0x73, 0x52, 0x05, 0x6f, 0x64, 0x63, 0x64,
	0x73, 0x3a, 0x35, 0x9a, 0xc5, 0x88, 0x1e, 0x30, 0x0a, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x2e, 0x68, 0x74,
	0x74, 0x70, 0x2e, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2e, 0x76, 0x32, 0x2e,
	0x4f, 0x6e, 0x44, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x22, 0x5f, 0x0a, 0x0e, 0x50, 0x65, 0x72, 0x52,
	0x6f, 0x75, 0x74, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x4d, 0x0a, 0x05, 0x6f, 0x64,
	0x63, 0x64, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x37, 0x2e, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d,
	0x61, 0x6e, 0x64, 0x2e, 0x76, 0x33, 0x2e, 0x4f, 0x6e, 0x44, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x43,
	0x64, 0x73, 0x52, 0x05, 0x6f, 0x64, 0x63, 0x64, 0x73, 0x42, 0xb2, 0x01, 0xba, 0x80, 0xc8, 0xd1,
	0x06, 0x02, 0x10, 0x02, 0x0a, 0x38, 0x69, 0x6f, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72,
	0x6f, 0x78, 0x79, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74,
	0x70, 0x2e, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2e, 0x76, 0x33, 0x42, 0x0d,
	0x4f, 0x6e, 0x44, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a,
	0x5d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2f, 0x67, 0x6f, 0x2d, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f,
	0x6c, 0x2d, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2f, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x65, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73,
	0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x2f,
	0x76, 0x33, 0x3b, 0x6f, 0x6e, 0x5f, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x64, 0x76, 0x33, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescOnce sync.Once
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescData = file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDesc
)

func file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescGZIP() []byte {
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescOnce.Do(func() {
		file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescData = protoimpl.X.CompressGZIP(file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescData)
	})
	return file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDescData
}

var file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_goTypes = []interface{}{
	(*OnDemandCds)(nil),       // 0: envoy.extensions.filters.http.on_demand.v3.OnDemandCds
	(*OnDemand)(nil),          // 1: envoy.extensions.filters.http.on_demand.v3.OnDemand
	(*PerRouteConfig)(nil),    // 2: envoy.extensions.filters.http.on_demand.v3.PerRouteConfig
	(*v3.ConfigSource)(nil),   // 3: envoy.config.core.v3.ConfigSource
	(*duration.Duration)(nil), // 4: google.protobuf.Duration
}
var file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_depIdxs = []int32{
	3, // 0: envoy.extensions.filters.http.on_demand.v3.OnDemandCds.source:type_name -> envoy.config.core.v3.ConfigSource
	4, // 1: envoy.extensions.filters.http.on_demand.v3.OnDemandCds.timeout:type_name -> google.protobuf.Duration
	0, // 2: envoy.extensions.filters.http.on_demand.v3.OnDemand.odcds:type_name -> envoy.extensions.filters.http.on_demand.v3.OnDemandCds
	0, // 3: envoy.extensions.filters.http.on_demand.v3.PerRouteConfig.odcds:type_name -> envoy.extensions.filters.http.on_demand.v3.OnDemandCds
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_init() }
func file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_init() {
	if File_envoy_extensions_filters_http_on_demand_v3_on_demand_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OnDemandCds); i {
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
		file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OnDemand); i {
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
		file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PerRouteConfig); i {
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
			RawDescriptor: file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_goTypes,
		DependencyIndexes: file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_depIdxs,
		MessageInfos:      file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_msgTypes,
	}.Build()
	File_envoy_extensions_filters_http_on_demand_v3_on_demand_proto = out.File
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_rawDesc = nil
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_goTypes = nil
	file_envoy_extensions_filters_http_on_demand_v3_on_demand_proto_depIdxs = nil
}
