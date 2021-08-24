// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: envoy/extensions/filters/http/ext_proc/v3alpha/ext_proc.proto

package envoy_extensions_filters_http_ext_proc_v3alpha

import (
	_ "github.com/cncf/udpa/go/udpa/annotations"
	v3 "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// [#next-free-field: 9]
type ExternalProcessor struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Configuration for the gRPC service that the filter will communicate with.
	// The filter supports both the "Envoy" and "Google" gRPC clients.
	GrpcService *v3.GrpcService `protobuf:"bytes,1,opt,name=grpc_service,json=grpcService,proto3" json:"grpc_service,omitempty"`
	// [#not-implemented-hide:]
	// By default, if the gRPC stream cannot be established, or if it is closed
	// prematurely with an error, the filter will fail. Specifically, if the
	// response headers have not yet been delivered, then it will return a 500
	// error downstream. If they have been delivered, then instead the HTTP stream to the
	// downstream client will be reset.
	// With this parameter set to true, however, then if the gRPC stream is prematurely closed
	// or could not be opened, processing continues without error.
	FailureModeAllow bool `protobuf:"varint,2,opt,name=failure_mode_allow,json=failureModeAllow,proto3" json:"failure_mode_allow,omitempty"`
	// [#not-implemented-hide:]
	// Specifies default options for how HTTP headers, trailers, and bodies are
	// sent. See ProcessingMode for details.
	ProcessingMode *ProcessingMode `protobuf:"bytes,3,opt,name=processing_mode,json=processingMode,proto3" json:"processing_mode,omitempty"`
	// [#not-implemented-hide:]
	// If true, send each part of the HTTP request or response specified by ProcessingMode
	// asynchronously -- in other words, send the message on the gRPC stream and then continue
	// filter processing. If false, which is the default, suspend filter execution after
	// each message is sent to the remote service and wait up to "message_timeout"
	// for a reply.
	AsyncMode bool `protobuf:"varint,4,opt,name=async_mode,json=asyncMode,proto3" json:"async_mode,omitempty"`
	// [#not-implemented-hide:]
	// Envoy provides a number of :ref:`attributes <arch_overview_attributes>`
	// for expressive policies. Each attribute name provided in this field will be
	// matched against that list and populated in the request_headers message.
	// See the :ref:`attribute documentation <arch_overview_request_attributes>`
	// for the list of supported attributes and their types.
	RequestAttributes []string `protobuf:"bytes,5,rep,name=request_attributes,json=requestAttributes,proto3" json:"request_attributes,omitempty"`
	// [#not-implemented-hide:]
	// Envoy provides a number of :ref:`attributes <arch_overview_attributes>`
	// for expressive policies. Each attribute name provided in this field will be
	// matched against that list and populated in the response_headers message.
	// See the :ref:`attribute documentation <arch_overview_attributes>`
	// for the list of supported attributes and their types.
	ResponseAttributes []string `protobuf:"bytes,6,rep,name=response_attributes,json=responseAttributes,proto3" json:"response_attributes,omitempty"`
	// [#not-implemented-hide:]
	// Specifies the timeout for each individual message sent on the stream and
	// when the filter is running in synchronous mode. Whenever
	// the proxy sends a message on the stream that requires a response, it will
	// reset this timer, and will stop processing and return an error (subject
	// to the processing mode) if the timer expires. There is no timeout when
	// the filter is running in asynchronous mode. Default is 200 ms.
	MessageTimeout *duration.Duration `protobuf:"bytes,7,opt,name=message_timeout,json=messageTimeout,proto3" json:"message_timeout,omitempty"`
	// [#not-implemented-hide:]
	// Optional additional prefix to use when emitting statistics. This allows to distinguish
	// emitted statistics between configured *ext_proc* filters in an HTTP filter chain.
	StatPrefix string `protobuf:"bytes,8,opt,name=stat_prefix,json=statPrefix,proto3" json:"stat_prefix,omitempty"`
}

func (x *ExternalProcessor) Reset() {
	*x = ExternalProcessor{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExternalProcessor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExternalProcessor) ProtoMessage() {}

func (x *ExternalProcessor) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExternalProcessor.ProtoReflect.Descriptor instead.
func (*ExternalProcessor) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescGZIP(), []int{0}
}

func (x *ExternalProcessor) GetGrpcService() *v3.GrpcService {
	if x != nil {
		return x.GrpcService
	}
	return nil
}

func (x *ExternalProcessor) GetFailureModeAllow() bool {
	if x != nil {
		return x.FailureModeAllow
	}
	return false
}

func (x *ExternalProcessor) GetProcessingMode() *ProcessingMode {
	if x != nil {
		return x.ProcessingMode
	}
	return nil
}

func (x *ExternalProcessor) GetAsyncMode() bool {
	if x != nil {
		return x.AsyncMode
	}
	return false
}

func (x *ExternalProcessor) GetRequestAttributes() []string {
	if x != nil {
		return x.RequestAttributes
	}
	return nil
}

func (x *ExternalProcessor) GetResponseAttributes() []string {
	if x != nil {
		return x.ResponseAttributes
	}
	return nil
}

func (x *ExternalProcessor) GetMessageTimeout() *duration.Duration {
	if x != nil {
		return x.MessageTimeout
	}
	return nil
}

func (x *ExternalProcessor) GetStatPrefix() string {
	if x != nil {
		return x.StatPrefix
	}
	return ""
}

// [#not-implemented-hide:]
// Extra settings that may be added to per-route configuration for a
// virtual host or cluster.
type ExtProcPerRoute struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Override:
	//	*ExtProcPerRoute_Disabled
	//	*ExtProcPerRoute_Overrides
	Override isExtProcPerRoute_Override `protobuf_oneof:"override"`
}

func (x *ExtProcPerRoute) Reset() {
	*x = ExtProcPerRoute{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExtProcPerRoute) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExtProcPerRoute) ProtoMessage() {}

func (x *ExtProcPerRoute) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExtProcPerRoute.ProtoReflect.Descriptor instead.
func (*ExtProcPerRoute) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescGZIP(), []int{1}
}

func (m *ExtProcPerRoute) GetOverride() isExtProcPerRoute_Override {
	if m != nil {
		return m.Override
	}
	return nil
}

func (x *ExtProcPerRoute) GetDisabled() bool {
	if x, ok := x.GetOverride().(*ExtProcPerRoute_Disabled); ok {
		return x.Disabled
	}
	return false
}

func (x *ExtProcPerRoute) GetOverrides() *ExtProcOverrides {
	if x, ok := x.GetOverride().(*ExtProcPerRoute_Overrides); ok {
		return x.Overrides
	}
	return nil
}

type isExtProcPerRoute_Override interface {
	isExtProcPerRoute_Override()
}

type ExtProcPerRoute_Disabled struct {
	// Disable the filter for this particular vhost or route.
	// If disabled is specified in multiple per-filter-configs, the most specific one will be used.
	Disabled bool `protobuf:"varint,1,opt,name=disabled,proto3,oneof"`
}

type ExtProcPerRoute_Overrides struct {
	// Override aspects of the configuration for this route
	Overrides *ExtProcOverrides `protobuf:"bytes,2,opt,name=overrides,proto3,oneof"`
}

func (*ExtProcPerRoute_Disabled) isExtProcPerRoute_Override() {}

func (*ExtProcPerRoute_Overrides) isExtProcPerRoute_Override() {}

// [#not-implemented-hide:]
// Overrides that may be set on a per-route basis
type ExtProcOverrides struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Set a different processing mode for this route than the default.
	ProcessingMode *ProcessingMode `protobuf:"bytes,1,opt,name=processing_mode,json=processingMode,proto3" json:"processing_mode,omitempty"`
	// Set a different asynchronous processing option than the default.
	AsyncMode bool `protobuf:"varint,2,opt,name=async_mode,json=asyncMode,proto3" json:"async_mode,omitempty"`
	// Set different optional properties than the default.
	RequestProperties []string `protobuf:"bytes,3,rep,name=request_properties,json=requestProperties,proto3" json:"request_properties,omitempty"`
	// Set different optional properties than the default.
	ResponseProperties []string `protobuf:"bytes,4,rep,name=response_properties,json=responseProperties,proto3" json:"response_properties,omitempty"`
}

func (x *ExtProcOverrides) Reset() {
	*x = ExtProcOverrides{}
	if protoimpl.UnsafeEnabled {
		mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExtProcOverrides) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExtProcOverrides) ProtoMessage() {}

func (x *ExtProcOverrides) ProtoReflect() protoreflect.Message {
	mi := &file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExtProcOverrides.ProtoReflect.Descriptor instead.
func (*ExtProcOverrides) Descriptor() ([]byte, []int) {
	return file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescGZIP(), []int{2}
}

func (x *ExtProcOverrides) GetProcessingMode() *ProcessingMode {
	if x != nil {
		return x.ProcessingMode
	}
	return nil
}

func (x *ExtProcOverrides) GetAsyncMode() bool {
	if x != nil {
		return x.AsyncMode
	}
	return false
}

func (x *ExtProcOverrides) GetRequestProperties() []string {
	if x != nil {
		return x.RequestProperties
	}
	return nil
}

func (x *ExtProcOverrides) GetResponseProperties() []string {
	if x != nil {
		return x.ResponseProperties
	}
	return nil
}

var File_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto protoreflect.FileDescriptor

var file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDesc = []byte{
	0x0a, 0x3d, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f,
	0x65, 0x78, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x63, 0x2f, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x65,
	0x78, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x63, 0x2e, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x1a,
	0x27, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x63, 0x6f,
	0x72, 0x65, 0x2f, 0x76, 0x33, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x44, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f,
	0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x74, 0x65,
	0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x63,
	0x2f, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73,
	0x69, 0x6e, 0x67, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1d,
	0x75, 0x64, 0x70, 0x61, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd4, 0x03, 0x0a, 0x11, 0x45, 0x78, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x12, 0x44, 0x0a, 0x0c,
	0x67, 0x72, 0x70, 0x63, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x21, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x76, 0x33, 0x2e, 0x47, 0x72, 0x70, 0x63, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x52, 0x0b, 0x67, 0x72, 0x70, 0x63, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x12, 0x2c, 0x0a, 0x12, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x5f, 0x6d, 0x6f,
	0x64, 0x65, 0x5f, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10,
	0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x41, 0x6c, 0x6c, 0x6f, 0x77,
	0x12, 0x67, 0x0a, 0x0f, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x6d,
	0x6f, 0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3e, 0x2e, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x72,
	0x6f, 0x63, 0x2e, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65,
	0x73, 0x73, 0x69, 0x6e, 0x67, 0x4d, 0x6f, 0x64, 0x65, 0x52, 0x0e, 0x70, 0x72, 0x6f, 0x63, 0x65,
	0x73, 0x73, 0x69, 0x6e, 0x67, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x73, 0x79,
	0x6e, 0x63, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x61,
	0x73, 0x79, 0x6e, 0x63, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x2d, 0x0a, 0x12, 0x72, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x5f, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x41, 0x74, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x12, 0x2f, 0x0a, 0x13, 0x72, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x5f, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x18, 0x06,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x12, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x41, 0x74,
	0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x12, 0x42, 0x0a, 0x0f, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0e, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x12, 0x1f, 0x0a, 0x0b,
	0x73, 0x74, 0x61, 0x74, 0x5f, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x22, 0xab, 0x01,
	0x0a, 0x0f, 0x45, 0x78, 0x74, 0x50, 0x72, 0x6f, 0x63, 0x50, 0x65, 0x72, 0x52, 0x6f, 0x75, 0x74,
	0x65, 0x12, 0x25, 0x0a, 0x08, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x6a, 0x02, 0x08, 0x01, 0x48, 0x00, 0x52, 0x08,
	0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x60, 0x0a, 0x09, 0x6f, 0x76, 0x65, 0x72,
	0x72, 0x69, 0x64, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x40, 0x2e, 0x65, 0x6e,
	0x76, 0x6f, 0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x65, 0x78, 0x74, 0x5f,
	0x70, 0x72, 0x6f, 0x63, 0x2e, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x45, 0x78, 0x74,
	0x50, 0x72, 0x6f, 0x63, 0x4f, 0x76, 0x65, 0x72, 0x72, 0x69, 0x64, 0x65, 0x73, 0x48, 0x00, 0x52,
	0x09, 0x6f, 0x76, 0x65, 0x72, 0x72, 0x69, 0x64, 0x65, 0x73, 0x42, 0x0f, 0x0a, 0x08, 0x6f, 0x76,
	0x65, 0x72, 0x72, 0x69, 0x64, 0x65, 0x12, 0x03, 0xf8, 0x42, 0x01, 0x22, 0xfa, 0x01, 0x0a, 0x10,
	0x45, 0x78, 0x74, 0x50, 0x72, 0x6f, 0x63, 0x4f, 0x76, 0x65, 0x72, 0x72, 0x69, 0x64, 0x65, 0x73,
	0x12, 0x67, 0x0a, 0x0f, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x6d,
	0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3e, 0x2e, 0x65, 0x6e, 0x76, 0x6f,
	0x79, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x72,
	0x6f, 0x63, 0x2e, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65,
	0x73, 0x73, 0x69, 0x6e, 0x67, 0x4d, 0x6f, 0x64, 0x65, 0x52, 0x0e, 0x70, 0x72, 0x6f, 0x63, 0x65,
	0x73, 0x73, 0x69, 0x6e, 0x67, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x73, 0x79,
	0x6e, 0x63, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x61,
	0x73, 0x79, 0x6e, 0x63, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x2d, 0x0a, 0x12, 0x72, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f,
	0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x2f, 0x0a, 0x13, 0x72, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x5f, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x04,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x12, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72,
	0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x42, 0x5e, 0x0a, 0x3c, 0x69, 0x6f, 0x2e, 0x65,
	0x6e, 0x76, 0x6f, 0x79, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2e, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2e,
	0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65,
	0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x72, 0x6f, 0x63,
	0x2e, 0x76, 0x33, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x42, 0x0c, 0x45, 0x78, 0x74, 0x50, 0x72, 0x6f,
	0x63, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0xba, 0x80, 0xc8, 0xd1, 0x06, 0x02, 0x08, 0x01,
	0xba, 0x80, 0xc8, 0xd1, 0x06, 0x02, 0x10, 0x02, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescOnce sync.Once
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescData = file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDesc
)

func file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescGZIP() []byte {
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescOnce.Do(func() {
		file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescData = protoimpl.X.CompressGZIP(file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescData)
	})
	return file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDescData
}

var file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_goTypes = []interface{}{
	(*ExternalProcessor)(nil), // 0: envoy.extensions.filters.http.ext_proc.v3alpha.ExternalProcessor
	(*ExtProcPerRoute)(nil),   // 1: envoy.extensions.filters.http.ext_proc.v3alpha.ExtProcPerRoute
	(*ExtProcOverrides)(nil),  // 2: envoy.extensions.filters.http.ext_proc.v3alpha.ExtProcOverrides
	(*v3.GrpcService)(nil),    // 3: envoy.config.core.v3.GrpcService
	(*ProcessingMode)(nil),    // 4: envoy.extensions.filters.http.ext_proc.v3alpha.ProcessingMode
	(*duration.Duration)(nil), // 5: google.protobuf.Duration
}
var file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_depIdxs = []int32{
	3, // 0: envoy.extensions.filters.http.ext_proc.v3alpha.ExternalProcessor.grpc_service:type_name -> envoy.config.core.v3.GrpcService
	4, // 1: envoy.extensions.filters.http.ext_proc.v3alpha.ExternalProcessor.processing_mode:type_name -> envoy.extensions.filters.http.ext_proc.v3alpha.ProcessingMode
	5, // 2: envoy.extensions.filters.http.ext_proc.v3alpha.ExternalProcessor.message_timeout:type_name -> google.protobuf.Duration
	2, // 3: envoy.extensions.filters.http.ext_proc.v3alpha.ExtProcPerRoute.overrides:type_name -> envoy.extensions.filters.http.ext_proc.v3alpha.ExtProcOverrides
	4, // 4: envoy.extensions.filters.http.ext_proc.v3alpha.ExtProcOverrides.processing_mode:type_name -> envoy.extensions.filters.http.ext_proc.v3alpha.ProcessingMode
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_init() }
func file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_init() {
	if File_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto != nil {
		return
	}
	file_envoy_extensions_filters_http_ext_proc_v3alpha_processing_mode_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExternalProcessor); i {
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
		file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExtProcPerRoute); i {
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
		file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExtProcOverrides); i {
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
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*ExtProcPerRoute_Disabled)(nil),
		(*ExtProcPerRoute_Overrides)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_goTypes,
		DependencyIndexes: file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_depIdxs,
		MessageInfos:      file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_msgTypes,
	}.Build()
	File_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto = out.File
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_rawDesc = nil
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_goTypes = nil
	file_envoy_extensions_filters_http_ext_proc_v3alpha_ext_proc_proto_depIdxs = nil
}
