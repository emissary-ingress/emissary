package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/datawire/ambassador-agent/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

const APIKeyMetadataKey = "x-ambassador-api-key"

type RPCComm struct {
	conn                     io.Closer
	client                   agent.DirectorClient
	rptWake                  chan struct{}
	retCancel                context.CancelFunc
	agentID                  *agent.Identity
	directives               chan *agent.Directive
	metricsStreamWriterMutex sync.Mutex
	extraHeaders             []string
}

const (
	defaultHostname = "app.getambassador.io"
	defaultPort     = "443"
)

type ConnInfo struct {
	hostname string
	port     string
	secure   bool
}

func connInfoFromAddress(address string) (*ConnInfo, error) {
	endpoint, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	hostname := endpoint.Hostname()
	if hostname == "" {
		hostname = defaultHostname
	}

	port := endpoint.Port()
	if port == "" {
		port = defaultPort
	}

	secure := true
	if endpoint.Scheme == "http" {
		secure = false
	}

	return &ConnInfo{hostname, port, secure}, nil
}

func NewComm(
	ctx context.Context,
	connInfo *ConnInfo,
	agentID *agent.Identity,
	apiKey string,
	extraHeaders []string,
) (*RPCComm, error) {
	ctx = dlog.WithField(ctx, "agent", "comm")
	opts := make([]grpc.DialOption, 0, 1)
	address := connInfo.hostname + ":" + connInfo.port

	if connInfo.secure {
		config := &tls.Config{ServerName: connInfo.hostname}
		creds := credentials.NewTLS(config)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	dlog.Debugf(ctx, "Dialing server at %s (secure=%t)", address, connInfo.secure)

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}

	client := agent.NewDirectorClient(conn)
	retCtx, retCancel := context.WithCancel(ctx)

	c := &RPCComm{
		conn:         conn,
		client:       client,
		retCancel:    retCancel,
		agentID:      agentID,
		directives:   make(chan *agent.Directive, 1),
		rptWake:      make(chan struct{}, 1),
		extraHeaders: extraHeaders,
	}
	retCtx = metadata.AppendToOutgoingContext(ctx, c.getHeaders(apiKey)...)

	go c.retrieveLoop(retCtx)

	return c, nil
}

func (c *RPCComm) getHeaders(apiKey string) []string {
	return append([]string{
		APIKeyMetadataKey, apiKey}, c.extraHeaders...)
}

func (c *RPCComm) retrieveLoop(ctx context.Context) {
	ctx = dlog.WithField(ctx, "agent", "retriever")

	for {
		if err := c.retrieve(ctx); err != nil {
			dlog.Debugf(ctx, "exited: %+v", err)
		}

		select {
		case <-c.rptWake:
			dlog.Debug(ctx, "restarting")
		case <-ctx.Done():
			return
		}
	}
}

func (c *RPCComm) retrieve(ctx context.Context) error {
	stream, err := c.client.Retrieve(ctx, c.agentID)

	if err != nil {
		return err
	}

	for {
		directive, err := stream.Recv()
		if err != nil {
			return err
		}

		select {
		case c.directives <- directive:
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *RPCComm) Close() error {
	c.retCancel()
	return c.conn.Close()
}

func (c *RPCComm) ReportCommandResult(ctx context.Context, result *agent.CommandResult, apiKey string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, c.getHeaders(apiKey)...)
	_, err := c.client.ReportCommandResult(ctx, result, grpc.EmptyCallOption{})
	if err != nil {
		return fmt.Errorf("ReportCommandResult error: %w", err)
	}
	return nil
}

func (c *RPCComm) Report(ctx context.Context, report *agent.Snapshot, apiKey string) error {
	select {
	case c.rptWake <- struct{}{}:
	default:
	}
	ctx = metadata.AppendToOutgoingContext(ctx, c.getHeaders(apiKey)...)

	// marshal snapshot
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	const CHUNKSIZE = (64 * 1024) - 4 // 64KiB-4B; gRPC adds 4 bytes of overhead
	dlog.Debugf(ctx, "Report is %dB; will take %d chunks",
		len(data),
		(len(data)+CHUNKSIZE-1)/CHUNKSIZE)

	// make stream
	stream, err := c.client.ReportStream(ctx)
	if err != nil {
		return fmt.Errorf("ReportStream.Open: %w", err)
	}

	// send chunks
	msg := &agent.RawSnapshotChunk{}
	for i := 0; i < len(data); i += CHUNKSIZE {
		j := i + CHUNKSIZE

		if j < len(data) {
			msg.Chunk = data[i:j]
		} else {
			msg.Chunk = data[i:]
		}

		if err := stream.Send(msg); err != nil {
			return fmt.Errorf("ReportStream.Send: %w", err)
		}
	}

	if _, err = stream.CloseAndRecv(); err != nil {
		return fmt.Errorf("ReportStream.Close: %w", err)
	}

	return nil
}

func (c *RPCComm) StreamMetrics(ctx context.Context, metrics *agent.StreamMetricsMessage, apiKey string) error {
	ctx = dlog.WithField(ctx, "agent", "streammetrics")

	c.metricsStreamWriterMutex.Lock()
	defer c.metricsStreamWriterMutex.Unlock()
	ctx = metadata.AppendToOutgoingContext(ctx, c.getHeaders(apiKey)...)
	streamClient, err := c.client.StreamMetrics(ctx)

	if err != nil {
		return err
	}

	if err := streamClient.Send(metrics); err != nil {
		return err
	}

	return streamClient.CloseSend()
}

func (c *RPCComm) Directives() <-chan *agent.Directive {
	return c.directives
}

func (c *RPCComm) StreamDiagnostics(ctx context.Context, diagnosticsReport *agent.Diagnostics, apiKey string) error {
	select {
	case c.rptWake <- struct{}{}:
	default:
	}
	ctx = metadata.AppendToOutgoingContext(ctx, c.getHeaders(apiKey)...)

	// marshal diagnostics into bytes
	data, err := json.Marshal(diagnosticsReport)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	const CHUNKSIZE = (64 * 1024) - 4 // 64KiB-4B; gRPC adds 4 bytes of overhead
	dlog.Debugf(ctx, "Diagnostics Report is %dB; will take %d chunks",
		len(data),
		(len(data)+CHUNKSIZE-1)/CHUNKSIZE)

	// make stream
	stream, err := c.client.StreamDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("ReportDiagnosticsStream.Open: %w", err)
	}

	// send chunks
	msg := &agent.RawDiagnosticsChunk{}
	for i := 0; i < len(data); i += CHUNKSIZE {
		j := i + CHUNKSIZE

		if j < len(data) {
			msg.Chunk = data[i:j]
		} else {
			msg.Chunk = data[i:]
		}

		if err := stream.Send(msg); err != nil {
			return fmt.Errorf("ReportDiagnosticsStream.Send: %w", err)
		}
	}

	if _, err = stream.CloseAndRecv(); err != nil {
		return fmt.Errorf("ReportDiagnosticsStream.Close: %w", err)
	}

	return nil
}
