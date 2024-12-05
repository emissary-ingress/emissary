// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
// source: envoy/api/v2/cds.proto

package apiv2

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ClusterDiscoveryService_StreamClusters_FullMethodName = "/envoy.api.v2.ClusterDiscoveryService/StreamClusters"
	ClusterDiscoveryService_DeltaClusters_FullMethodName  = "/envoy.api.v2.ClusterDiscoveryService/DeltaClusters"
	ClusterDiscoveryService_FetchClusters_FullMethodName  = "/envoy.api.v2.ClusterDiscoveryService/FetchClusters"
)

// ClusterDiscoveryServiceClient is the client API for ClusterDiscoveryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClusterDiscoveryServiceClient interface {
	StreamClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_StreamClustersClient, error)
	DeltaClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_DeltaClustersClient, error)
	FetchClusters(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
}

type clusterDiscoveryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterDiscoveryServiceClient(cc grpc.ClientConnInterface) ClusterDiscoveryServiceClient {
	return &clusterDiscoveryServiceClient{cc}
}

func (c *clusterDiscoveryServiceClient) StreamClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_StreamClustersClient, error) {
	stream, err := c.cc.NewStream(ctx, &ClusterDiscoveryService_ServiceDesc.Streams[0], ClusterDiscoveryService_StreamClusters_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &clusterDiscoveryServiceStreamClustersClient{stream}
	return x, nil
}

type ClusterDiscoveryService_StreamClustersClient interface {
	Send(*DiscoveryRequest) error
	Recv() (*DiscoveryResponse, error)
	grpc.ClientStream
}

type clusterDiscoveryServiceStreamClustersClient struct {
	grpc.ClientStream
}

func (x *clusterDiscoveryServiceStreamClustersClient) Send(m *DiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceStreamClustersClient) Recv() (*DiscoveryResponse, error) {
	m := new(DiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *clusterDiscoveryServiceClient) DeltaClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_DeltaClustersClient, error) {
	stream, err := c.cc.NewStream(ctx, &ClusterDiscoveryService_ServiceDesc.Streams[1], ClusterDiscoveryService_DeltaClusters_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &clusterDiscoveryServiceDeltaClustersClient{stream}
	return x, nil
}

type ClusterDiscoveryService_DeltaClustersClient interface {
	Send(*DeltaDiscoveryRequest) error
	Recv() (*DeltaDiscoveryResponse, error)
	grpc.ClientStream
}

type clusterDiscoveryServiceDeltaClustersClient struct {
	grpc.ClientStream
}

func (x *clusterDiscoveryServiceDeltaClustersClient) Send(m *DeltaDiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceDeltaClustersClient) Recv() (*DeltaDiscoveryResponse, error) {
	m := new(DeltaDiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *clusterDiscoveryServiceClient) FetchClusters(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error) {
	out := new(DiscoveryResponse)
	err := c.cc.Invoke(ctx, ClusterDiscoveryService_FetchClusters_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterDiscoveryServiceServer is the server API for ClusterDiscoveryService service.
// All implementations should embed UnimplementedClusterDiscoveryServiceServer
// for forward compatibility
type ClusterDiscoveryServiceServer interface {
	StreamClusters(ClusterDiscoveryService_StreamClustersServer) error
	DeltaClusters(ClusterDiscoveryService_DeltaClustersServer) error
	FetchClusters(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
}

// UnimplementedClusterDiscoveryServiceServer should be embedded to have forward compatible implementations.
type UnimplementedClusterDiscoveryServiceServer struct {
}

func (UnimplementedClusterDiscoveryServiceServer) StreamClusters(ClusterDiscoveryService_StreamClustersServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamClusters not implemented")
}
func (UnimplementedClusterDiscoveryServiceServer) DeltaClusters(ClusterDiscoveryService_DeltaClustersServer) error {
	return status.Errorf(codes.Unimplemented, "method DeltaClusters not implemented")
}
func (UnimplementedClusterDiscoveryServiceServer) FetchClusters(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchClusters not implemented")
}

// UnsafeClusterDiscoveryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ClusterDiscoveryServiceServer will
// result in compilation errors.
type UnsafeClusterDiscoveryServiceServer interface {
	mustEmbedUnimplementedClusterDiscoveryServiceServer()
}

func RegisterClusterDiscoveryServiceServer(s grpc.ServiceRegistrar, srv ClusterDiscoveryServiceServer) {
	s.RegisterService(&ClusterDiscoveryService_ServiceDesc, srv)
}

func _ClusterDiscoveryService_StreamClusters_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClusterDiscoveryServiceServer).StreamClusters(&clusterDiscoveryServiceStreamClustersServer{stream})
}

type ClusterDiscoveryService_StreamClustersServer interface {
	Send(*DiscoveryResponse) error
	Recv() (*DiscoveryRequest, error)
	grpc.ServerStream
}

type clusterDiscoveryServiceStreamClustersServer struct {
	grpc.ServerStream
}

func (x *clusterDiscoveryServiceStreamClustersServer) Send(m *DiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceStreamClustersServer) Recv() (*DiscoveryRequest, error) {
	m := new(DiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ClusterDiscoveryService_DeltaClusters_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClusterDiscoveryServiceServer).DeltaClusters(&clusterDiscoveryServiceDeltaClustersServer{stream})
}

type ClusterDiscoveryService_DeltaClustersServer interface {
	Send(*DeltaDiscoveryResponse) error
	Recv() (*DeltaDiscoveryRequest, error)
	grpc.ServerStream
}

type clusterDiscoveryServiceDeltaClustersServer struct {
	grpc.ServerStream
}

func (x *clusterDiscoveryServiceDeltaClustersServer) Send(m *DeltaDiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceDeltaClustersServer) Recv() (*DeltaDiscoveryRequest, error) {
	m := new(DeltaDiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ClusterDiscoveryService_FetchClusters_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiscoveryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterDiscoveryServiceServer).FetchClusters(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterDiscoveryService_FetchClusters_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterDiscoveryServiceServer).FetchClusters(ctx, req.(*DiscoveryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ClusterDiscoveryService_ServiceDesc is the grpc.ServiceDesc for ClusterDiscoveryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ClusterDiscoveryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "envoy.api.v2.ClusterDiscoveryService",
	HandlerType: (*ClusterDiscoveryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchClusters",
			Handler:    _ClusterDiscoveryService_FetchClusters_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamClusters",
			Handler:       _ClusterDiscoveryService_StreamClusters_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "DeltaClusters",
			Handler:       _ClusterDiscoveryService_DeltaClusters_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "envoy/api/v2/cds.proto",
}
