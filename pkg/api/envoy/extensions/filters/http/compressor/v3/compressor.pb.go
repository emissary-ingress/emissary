// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.4
// source: envoy/extensions/filters/http/compressor/v3/compressor.proto

package compressorv3

import (
	_ "github.com/cncf/xds/go/udpa/annotations"
	_ "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/annotations"
	v3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
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

// [#next-free-field: 9]
type Compressor struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Minimum response length, in bytes, which will trigger compression. The default value is 30.
	//
	// Deprecated: Do not use.
	ContentLength *wrappers.UInt32Value `protobuf:"bytes,1,opt,name=content_length,json=contentLength,proto3" json:"content_length,omitempty"`
	// Set of strings that allows specifying which mime-types yield compression; e.g.,
	// application/json, text/html, etc. When this field is not defined, compression will be applied
	// to the following mime-types: "application/javascript", "application/json",
	// "application/xhtml+xml", "image/svg+xml", "text/css", "text/html", "text/plain", "text/xml"
	// and their synonyms.
	//
	// Deprecated: Do not use.
	ContentType []string `protobuf:"bytes,2,rep,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	// If true, disables compression when the response contains an etag header. When it is false, the
	// filter will preserve weak etags and remove the ones that require strong validation.
	//
	// Deprecated: Do not use.
	DisableOnEtagHeader bool `protobuf:"varint,3,opt,name=disable_on_etag_header,json=disableOnEtagHeader,proto3" json:"disable_on_etag_header,omitempty"`
	// If true, removes accept-encoding from the request headers before dispatching it to the upstream
	// so that responses do not get compressed before reaching the filter.
	//
	// .. attention::
	//
	//    To avoid interfering with other compression filters in the same chain use this option in
	//    the filter closest to the upstream.
	//
	// Deprecated: Do not use.
	RemoveAcceptEncodingHeader bool `protobuf:"varint,4,opt,name=remove_accept_encoding_header,json=removeAcceptEncodingHeader,proto3" json:"remove_accept_encoding_header,omitempty"`
	// Runtime flag that controls whether the filter is enabled or not. If set to false, the
	// filter will operate as a pass-through filter. If not specified, defaults to enabled.
	//
	// Deprecated: Do not use.
	RuntimeEnabled *v3.RuntimeFeatureFlag `protobuf:"bytes,5,opt,name=runtime_enabled,json=runtimeEnabled,proto3" json:"runtime_enabled,omitempty"`
	// A compressor library to use for compression. Currently only
	// :ref:`envoy.compression.gzip.compressor<envoy_v3_api_msg_extensions.compression.gzip.compressor.v3.Gzip>`
	// is included in Envoy.
	// [#extension-category: envoy.compression.compressor]
	CompressorLibrary *v3.TypedExtensionConfig `protobuf:"bytes,6,opt,name=compressor_library,json=compressorLibrary,proto3" json:"compressor_library,omitempty"`
	// Configuration for request compression. Compression is disabled by default if left empty.
	RequestDirectionConfig *Compressor_RequestDirectionConfig `protobuf:"bytes,7,opt,name=request_direction_config,json=requestDirectionConfig,proto3" json:"request_direction_config,omitempty"`
	// Configuration for response compression. Compression is enabled by default if left empty.
	//
	// .. attention::
	//
	//    If the field is not empty then the duplicate deprecated fields of the `Compressor` message,
	//    such as `content_length`, `content_type`, `disable_on_etag_header`,
	//    `remove_accept_encoding_header` and `runtime_enabled`, are ignored.
	//
	//    Also all the statistics related to response compression will be rooted in
	//    `<stat_prefix>.compressor.<compressor_library.name>.<compressor_library_stat_prefix>.response.*`
	//    instead of
	//    `<stat_prefix>.compressor.<compressor_library.name>.<compressor_library_stat_prefix>.*`.
	ResponseDirectionConfig *Compressor_ResponseDirectionConfig `protobuf:"bytes,8,opt,name=response_direction_config,json=responseDirectionConfig,proto3" json:"response_direction_config,omitempty"`
}

func (x *Compressor) Reset() {
	*x = Compressor{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Compressor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Compressor) ProtoMessage() {}

func (x *Compressor) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Compressor.ProtoReflect.Descriptor instead.
func (*Compressor) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescGZIP(), []int{0}
}

// Deprecated: Do not use.
func (x *Compressor) GetContentLength() *wrappers.UInt32Value {
	if x != nil {
		return x.ContentLength
	}
	return nil
}

// Deprecated: Do not use.
func (x *Compressor) GetContentType() []string {
	if x != nil {
		return x.ContentType
	}
	return nil
}

// Deprecated: Do not use.
func (x *Compressor) GetDisableOnEtagHeader() bool {
	if x != nil {
		return x.DisableOnEtagHeader
	}
	return false
}

// Deprecated: Do not use.
func (x *Compressor) GetRemoveAcceptEncodingHeader() bool {
	if x != nil {
		return x.RemoveAcceptEncodingHeader
	}
	return false
}

// Deprecated: Do not use.
func (x *Compressor) GetRuntimeEnabled() *v3.RuntimeFeatureFlag {
	if x != nil {
		return x.RuntimeEnabled
	}
	return nil
}

func (x *Compressor) GetCompressorLibrary() *v3.TypedExtensionConfig {
	if x != nil {
		return x.CompressorLibrary
	}
	return nil
}

func (x *Compressor) GetRequestDirectionConfig() *Compressor_RequestDirectionConfig {
	if x != nil {
		return x.RequestDirectionConfig
	}
	return nil
}

func (x *Compressor) GetResponseDirectionConfig() *Compressor_ResponseDirectionConfig {
	if x != nil {
		return x.ResponseDirectionConfig
	}
	return nil
}

type Compressor_CommonDirectionConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Runtime flag that controls whether compression is enabled or not for the direction this
	// common config is put in. If set to false, the filter will operate as a pass-through filter
	// in the chosen direction. If the field is omitted, the filter will be enabled.
	Enabled *v3.RuntimeFeatureFlag `protobuf:"bytes,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	// Minimum value of Content-Length header of request or response messages (depending on the direction
	// this common config is put in), in bytes, which will trigger compression. The default value is 30.
	MinContentLength *wrappers.UInt32Value `protobuf:"bytes,2,opt,name=min_content_length,json=minContentLength,proto3" json:"min_content_length,omitempty"`
	// Set of strings that allows specifying which mime-types yield compression; e.g.,
	// application/json, text/html, etc. When this field is not defined, compression will be applied
	// to the following mime-types: "application/javascript", "application/json",
	// "application/xhtml+xml", "image/svg+xml", "text/css", "text/html", "text/plain", "text/xml"
	// and their synonyms.
	ContentType []string `protobuf:"bytes,3,rep,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
}

func (x *Compressor_CommonDirectionConfig) Reset() {
	*x = Compressor_CommonDirectionConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Compressor_CommonDirectionConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Compressor_CommonDirectionConfig) ProtoMessage() {}

func (x *Compressor_CommonDirectionConfig) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Compressor_CommonDirectionConfig.ProtoReflect.Descriptor instead.
func (*Compressor_CommonDirectionConfig) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Compressor_CommonDirectionConfig) GetEnabled() *v3.RuntimeFeatureFlag {
	if x != nil {
		return x.Enabled
	}
	return nil
}

func (x *Compressor_CommonDirectionConfig) GetMinContentLength() *wrappers.UInt32Value {
	if x != nil {
		return x.MinContentLength
	}
	return nil
}

func (x *Compressor_CommonDirectionConfig) GetContentType() []string {
	if x != nil {
		return x.ContentType
	}
	return nil
}

// Configuration for filter behavior on the request direction.
type Compressor_RequestDirectionConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommonConfig *Compressor_CommonDirectionConfig `protobuf:"bytes,1,opt,name=common_config,json=commonConfig,proto3" json:"common_config,omitempty"`
}

func (x *Compressor_RequestDirectionConfig) Reset() {
	*x = Compressor_RequestDirectionConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Compressor_RequestDirectionConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Compressor_RequestDirectionConfig) ProtoMessage() {}

func (x *Compressor_RequestDirectionConfig) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Compressor_RequestDirectionConfig.ProtoReflect.Descriptor instead.
func (*Compressor_RequestDirectionConfig) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Compressor_RequestDirectionConfig) GetCommonConfig() *Compressor_CommonDirectionConfig {
	if x != nil {
		return x.CommonConfig
	}
	return nil
}

// Configuration for filter behavior on the response direction.
type Compressor_ResponseDirectionConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommonConfig *Compressor_CommonDirectionConfig `protobuf:"bytes,1,opt,name=common_config,json=commonConfig,proto3" json:"common_config,omitempty"`
	// If true, disables compression when the response contains an etag header. When it is false, the
	// filter will preserve weak etags and remove the ones that require strong validation.
	DisableOnEtagHeader bool `protobuf:"varint,2,opt,name=disable_on_etag_header,json=disableOnEtagHeader,proto3" json:"disable_on_etag_header,omitempty"`
	// If true, removes accept-encoding from the request headers before dispatching it to the upstream
	// so that responses do not get compressed before reaching the filter.
	//
	// .. attention::
	//
	//    To avoid interfering with other compression filters in the same chain use this option in
	//    the filter closest to the upstream.
	RemoveAcceptEncodingHeader bool `protobuf:"varint,3,opt,name=remove_accept_encoding_header,json=removeAcceptEncodingHeader,proto3" json:"remove_accept_encoding_header,omitempty"`
}

func (x *Compressor_ResponseDirectionConfig) Reset() {
	*x = Compressor_ResponseDirectionConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Compressor_ResponseDirectionConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Compressor_ResponseDirectionConfig) ProtoMessage() {}

func (x *Compressor_ResponseDirectionConfig) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Compressor_ResponseDirectionConfig.ProtoReflect.Descriptor instead.
func (*Compressor_ResponseDirectionConfig) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescGZIP(), []int{0, 2}
}

func (x *Compressor_ResponseDirectionConfig) GetCommonConfig() *Compressor_CommonDirectionConfig {
	if x != nil {
		return x.CommonConfig
	}
	return nil
}

func (x *Compressor_ResponseDirectionConfig) GetDisableOnEtagHeader() bool {
	if x != nil {
		return x.DisableOnEtagHeader
	}
	return false
}

func (x *Compressor_ResponseDirectionConfig) GetRemoveAcceptEncodingHeader() bool {
	if x != nil {
		return x.RemoveAcceptEncodingHeader
	}
	return false
}

var File_envoy_extensions_filters_http_compressor_v3_compressor_proto protoreflect.FileDescriptor

var file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f,
	0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2f, 0x76, 0x33, 0x2f, 0x63, 0x6f,
	0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x2b,
	0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x63, 0x6f,
	0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33, 0x1a, 0x1f, 0x65, 0x6e, 0x76,
	0x6f, 0x79, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x76,
	0x33, 0x2f, 0x62, 0x61, 0x73, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x24, 0x65, 0x6e,
	0x76, 0x6f, 0x79, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f,
	0x76, 0x33, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x23, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1d, 0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e,
	0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e,
	0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x9c, 0x0b, 0x0a, 0x0a, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f,
	0x72, 0x12, 0x50, 0x0a, 0x0e, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x6c, 0x65, 0x6e,
	0x67, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74,
	0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x0b, 0x18, 0x01, 0x92, 0xc7, 0x86, 0xd8, 0x04,
	0x03, 0x33, 0x2e, 0x30, 0x52, 0x0d, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x4c, 0x65, 0x6e,
	0x67, 0x74, 0x68, 0x12, 0x2e, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x42, 0x0b, 0x18, 0x01, 0x92, 0xc7, 0x86,
	0xd8, 0x04, 0x03, 0x33, 0x2e, 0x30, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x40, 0x0a, 0x16, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6f,
	0x6e, 0x5f, 0x65, 0x74, 0x61, 0x67, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x08, 0x42, 0x0b, 0x18, 0x01, 0x92, 0xc7, 0x86, 0xd8, 0x04, 0x03, 0x33, 0x2e, 0x30,
	0x52, 0x13, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x4f, 0x6e, 0x45, 0x74, 0x61, 0x67, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x4e, 0x0a, 0x1d, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x5f,
	0x61, 0x63, 0x63, 0x65, 0x70, 0x74, 0x5f, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f,
	0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x42, 0x0b, 0x18, 0x01,
	0x92, 0xc7, 0x86, 0xd8, 0x04, 0x03, 0x33, 0x2e, 0x30, 0x52, 0x1a, 0x72, 0x65, 0x6d, 0x6f, 0x76,
	0x65, 0x41, 0x63, 0x63, 0x65, 0x70, 0x74, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x5e, 0x0a, 0x0f, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65,
	0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28,
	0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x63, 0x6f,
	0x72, 0x65, 0x2e, 0x76, 0x33, 0x2e, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x46, 0x65, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x46, 0x6c, 0x61, 0x67, 0x42, 0x0b, 0x18, 0x01, 0x92, 0xc7, 0x86, 0xd8,
	0x04, 0x03, 0x33, 0x2e, 0x30, 0x52, 0x0e, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x45, 0x6e,
	0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x63, 0x0a, 0x12, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73,
	0x73, 0x6f, 0x72, 0x5f, 0x6c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x2a, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x76, 0x33, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x64, 0x45, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x08, 0xfa,
	0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x11, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73,
	0x73, 0x6f, 0x72, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x12, 0x88, 0x01, 0x0a, 0x18, 0x72,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x4e, 0x2e,
	0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x63, 0x6f,
	0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33, 0x2e, 0x43, 0x6f, 0x6d, 0x70,
	0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x44, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x16, 0x72,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x8b, 0x01, 0x0a, 0x19, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x5f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x4f, 0x2e, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65,
	0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73,
	0x6f, 0x72, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x17, 0x72, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x1a, 0xca, 0x01, 0x0a, 0x15, 0x43, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x44, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x42, 0x0a,
	0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28,
	0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x63, 0x6f,
	0x72, 0x65, 0x2e, 0x76, 0x33, 0x2e, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x46, 0x65, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x46, 0x6c, 0x61, 0x67, 0x52, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65,
	0x64, 0x12, 0x4a, 0x0a, 0x12, 0x6d, 0x69, 0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x5f, 0x6c, 0x65, 0x6e, 0x67, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x55, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x10, 0x6d, 0x69, 0x6e,
	0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x4c, 0x65, 0x6e, 0x67, 0x74, 0x68, 0x12, 0x21, 0x0a,
	0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65,
	0x1a, 0x8c, 0x01, 0x0a, 0x16, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x44, 0x69, 0x72, 0x65,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x72, 0x0a, 0x0d, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x4d, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74,
	0x74, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33,
	0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x43, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x52, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x1a,
	0x85, 0x02, 0x0a, 0x17, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x44, 0x69, 0x72, 0x65,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x72, 0x0a, 0x0d, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x4d, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74,
	0x74, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33,
	0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x43, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x52, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12,
	0x33, 0x0a, 0x16, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6f, 0x6e, 0x5f, 0x65, 0x74,
	0x61, 0x67, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x13, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x4f, 0x6e, 0x45, 0x74, 0x61, 0x67, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x12, 0x41, 0x0a, 0x1d, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x5f, 0x61,
	0x63, 0x63, 0x65, 0x70, 0x74, 0x5f, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x68,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x1a, 0x72, 0x65, 0x6d,
	0x6f, 0x76, 0x65, 0x41, 0x63, 0x63, 0x65, 0x70, 0x74, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e,
	0x67, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x3a, 0x38, 0x9a, 0xc5, 0x88, 0x1e, 0x33, 0x0a, 0x31,
	0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73,
	0x73, 0x6f, 0x72, 0x2e, 0x76, 0x32, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f,
	0x72, 0x42, 0xb7, 0x01, 0x0a, 0x39, 0x69, 0x6f, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72,
	0x6f, 0x78, 0x79, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74,
	0x70, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x2e, 0x76, 0x33, 0x42,
	0x0f, 0x43, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x5f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65,
	0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2f, 0x67, 0x6f, 0x2d, 0x63, 0x6f, 0x6e,
	0x74, 0x72, 0x6f, 0x6c, 0x2d, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x2f, 0x65, 0x6e, 0x76, 0x6f, 0x79,
	0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73,
	0x73, 0x6f, 0x72, 0x2f, 0x76, 0x33, 0x3b, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x6f,
	0x72, 0x76, 0x33, 0xba, 0x80, 0xc8, 0xd1, 0x06, 0x02, 0x10, 0x02, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescOnce sync.Once
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescData = file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDesc
)

func file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescGZIP() []byte {
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescOnce.Do(func() {
		file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescData = protoimpl.X.CompressGZIP(file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescData)
	})
	return file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDescData
}

var file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_envoy_extensions_filters_http_compressor_v3_compressor_proto_goTypes = []interface{}{
	(*Compressor)(nil),                         // 0: envoy.extensions.filters.http.compressor.v3.Compressor
	(*Compressor_CommonDirectionConfig)(nil),   // 1: envoy.extensions.filters.http.compressor.v3.Compressor.CommonDirectionConfig
	(*Compressor_RequestDirectionConfig)(nil),  // 2: envoy.extensions.filters.http.compressor.v3.Compressor.RequestDirectionConfig
	(*Compressor_ResponseDirectionConfig)(nil), // 3: envoy.extensions.filters.http.compressor.v3.Compressor.ResponseDirectionConfig
	(*wrappers.UInt32Value)(nil),               // 4: google.protobuf.UInt32Value
	(*v3.RuntimeFeatureFlag)(nil),              // 5: envoy.config.core.v3.RuntimeFeatureFlag
	(*v3.TypedExtensionConfig)(nil),            // 6: envoy.config.core.v3.TypedExtensionConfig
}
var file_envoy_extensions_filters_http_compressor_v3_compressor_proto_depIdxs = []int32{
	4, // 0: envoy.extensions.filters.http.compressor.v3.Compressor.content_length:type_name -> google.protobuf.UInt32Value
	5, // 1: envoy.extensions.filters.http.compressor.v3.Compressor.runtime_enabled:type_name -> envoy.config.core.v3.RuntimeFeatureFlag
	6, // 2: envoy.extensions.filters.http.compressor.v3.Compressor.compressor_library:type_name -> envoy.config.core.v3.TypedExtensionConfig
	2, // 3: envoy.extensions.filters.http.compressor.v3.Compressor.request_direction_config:type_name -> envoy.extensions.filters.http.compressor.v3.Compressor.RequestDirectionConfig
	3, // 4: envoy.extensions.filters.http.compressor.v3.Compressor.response_direction_config:type_name -> envoy.extensions.filters.http.compressor.v3.Compressor.ResponseDirectionConfig
	5, // 5: envoy.extensions.filters.http.compressor.v3.Compressor.CommonDirectionConfig.enabled:type_name -> envoy.config.core.v3.RuntimeFeatureFlag
	4, // 6: envoy.extensions.filters.http.compressor.v3.Compressor.CommonDirectionConfig.min_content_length:type_name -> google.protobuf.UInt32Value
	1, // 7: envoy.extensions.filters.http.compressor.v3.Compressor.RequestDirectionConfig.common_config:type_name -> envoy.extensions.filters.http.compressor.v3.Compressor.CommonDirectionConfig
	1, // 8: envoy.extensions.filters.http.compressor.v3.Compressor.ResponseDirectionConfig.common_config:type_name -> envoy.extensions.filters.http.compressor.v3.Compressor.CommonDirectionConfig
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_envoy_extensions_filters_http_compressor_v3_compressor_proto_init() }
func file_envoy_extensions_filters_http_compressor_v3_compressor_proto_init() {
	if File_envoy_extensions_filters_http_compressor_v3_compressor_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Compressor); i {
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
		file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Compressor_CommonDirectionConfig); i {
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
		file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Compressor_RequestDirectionConfig); i {
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
		file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Compressor_ResponseDirectionConfig); i {
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
			RawDescriptor: file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_envoy_extensions_filters_http_compressor_v3_compressor_proto_goTypes,
		DependencyIndexes: file_envoy_extensions_filters_http_compressor_v3_compressor_proto_depIdxs,
		MessageInfos:      file_envoy_extensions_filters_http_compressor_v3_compressor_proto_msgTypes,
	}.Build()
	File_envoy_extensions_filters_http_compressor_v3_compressor_proto = out.File
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_rawDesc = nil
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_goTypes = nil
	file_envoy_extensions_filters_http_compressor_v3_compressor_proto_depIdxs = nil
}
