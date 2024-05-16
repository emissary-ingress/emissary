// Copyright 2022 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

// Package sotw provides an implementation of GRPC SoTW (State of The World) part of XDS client
package sotw

import (
	"context"
	"errors"
	"io"
	"sync"

	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
)

var (
	ErrInit    = errors.New("ads client: grpc connection is not initialized (use InitConnect() method to initialize connection)")
	ErrNilResp = errors.New("ads client: nil response from xds management server")
)

// ADSClient is a SoTW and ADS based generic gRPC xDS client which can be used to
// implement an xDS client which fetches resources from an xDS server and responds
// the server back with ack or nack the resources.
type ADSClient interface {
	// Initialize the gRPC connection with management server and send the initial Discovery Request.
	InitConnect(clientConn grpc.ClientConnInterface, opts ...grpc.CallOption) error
	// Fetch waits for a response from management server and returns response or error.
	Fetch() (*Response, error)
	// Ack acknowledge the validity of the last received response to management server.
	Ack() error
	// Nack acknowledge the invalidity of the last received response to management server.
	Nack(message string) error
}

// Response wraps the latest Resources from the xDS server.
// For the time being it only contains the Resources from server. This can be extended with
// other response details. For example some metadata from DiscoveryResponse.
type Response struct {
	Resources []*anypb.Any
}

type adsClient struct {
	ctx     context.Context
	mu      sync.Mutex
	node    *core.Node
	typeURL string

	// streamClient is the ADS discovery client
	streamClient discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	// lastAckedResponse is the last response acked by the ADS client
	lastAckedResponse *discovery.DiscoveryResponse
	// lastReceivedResponse is the last response received from management server
	lastReceivedResponse *discovery.DiscoveryResponse
}

// NewADSClient returns a new ADSClient
func NewADSClient(ctx context.Context, node *core.Node, typeURL string) ADSClient {
	return &adsClient{
		ctx:     ctx,
		node:    node,
		typeURL: typeURL,
	}
}

// Initialize the gRPC connection with management server and send the initial Discovery Request.
func (c *adsClient) InitConnect(clientConn grpc.ClientConnInterface, opts ...grpc.CallOption) error {
	streamClient, err := discovery.NewAggregatedDiscoveryServiceClient(clientConn).StreamAggregatedResources(c.ctx, opts...)
	if err != nil {
		return err
	}
	c.streamClient = streamClient
	return c.Ack()
}

// Fetch waits for a response from management server and returns response or error.
func (c *adsClient) Fetch() (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamClient == nil {
		return nil, ErrInit
	}
	resp, err := c.streamClient.Recv()
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, ErrNilResp
	}
	c.lastReceivedResponse = resp
	return &Response{
		Resources: resp.GetResources(),
	}, err
}

// Ack acknowledge the validity of the last received response to management server.
func (c *adsClient) Ack() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastAckedResponse = c.lastReceivedResponse
	return c.send(nil)
}

// Nack acknowledge the invalidity of the last received response to management server.
func (c *adsClient) Nack(message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	errorDetail := &status.Status{
		Message: message,
	}
	return c.send(errorDetail)
}

// IsConnError checks the provided error is due to the gRPC connection
// and returns true if it is due to the gRPC connection.
//
// In this case the gRPC connection with the server should be re initialized with the
// ADSClient.InitConnect method.
func IsConnError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	errStatus, ok := grpcStatus.FromError(err)
	if !ok {
		return false
	}
	return errStatus.Code() == codes.Unavailable || errStatus.Code() == codes.Canceled
}

func (c *adsClient) send(errorDetail *status.Status) error {
	if c.streamClient == nil {
		return ErrInit
	}

	req := &discovery.DiscoveryRequest{
		Node:          c.node,
		VersionInfo:   c.lastAckedResponse.GetVersionInfo(),
		TypeUrl:       c.typeURL,
		ResponseNonce: c.lastReceivedResponse.GetNonce(),
		ErrorDetail:   errorDetail,
	}
	return c.streamClient.Send(req)
}
