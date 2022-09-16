package services

import (
	// stdlib
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"

	// third party
	"google.golang.org/grpc"

	// first party
	logdatav2 "github.com/datawire/ambassador/v2/pkg/api/envoy/data/accesslog/v2"
	logdatav3 "github.com/datawire/ambassador/v2/pkg/api/envoy/data/accesslog/v3"
	alsv2 "github.com/datawire/ambassador/v2/pkg/api/envoy/service/accesslog/v2"
	alsv3 "github.com/datawire/ambassador/v2/pkg/api/envoy/service/accesslog/v3"
)

type GRPCALS struct {
	HTTPListener

	mu     sync.Mutex
	v2http []*logdatav2.HTTPAccessLogEntry
	v2tcp  []*logdatav2.TCPAccessLogEntry
	v3http []*logdatav3.HTTPAccessLogEntry
	v3tcp  []*logdatav3.TCPAccessLogEntry
}

func (als *GRPCALS) Start(ctx context.Context) <-chan bool {
	httpHandler := http.NewServeMux()
	httpHandler.HandleFunc("/logs", als.ServeLogs)

	grpcHandler := grpc.NewServer()
	alsv2.RegisterAccessLogServiceServer(grpcHandler, ALSv2{als})
	alsv3.RegisterAccessLogServiceServer(grpcHandler, ALSv3{als})

	return als.HTTPListener.Run(ctx, "gRPC ALS", httpHandler, grpcHandler)
}

func (als *GRPCALS) ServeLogs(w http.ResponseWriter, r *http.Request) {
	als.mu.Lock()
	defer als.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		bs, err := json.Marshal(map[string]interface{}{
			"alsv2-http": als.v2http,
			"alsv2-tcp":  als.v2tcp,
			"alsv3-http": als.v3http,
			"alsv3-tcp":  als.v3tcp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(bs)
	case http.MethodDelete:
		als.v2http = nil
		als.v2tcp = nil
		als.v3http = nil
		als.v3tcp = nil
	default:
		http.Error(w, "only responds to GET and DELETE", http.StatusMethodNotAllowed)
	}
}

type ALSv2 struct {
	*GRPCALS
}

func (als ALSv2) StreamAccessLogs(srv alsv2.AccessLogService_StreamAccessLogsServer) error {
	for {
		msg, err := srv.Recv()
		if msg != nil {
			switch logEntries := msg.LogEntries.(type) {
			case *alsv2.StreamAccessLogsMessage_HttpLogs:
				if logEntries.HttpLogs != nil {
					als.mu.Lock()
					als.v2http = append(als.v2http, logEntries.HttpLogs.LogEntry...)
					als.mu.Unlock()
				}
			case *alsv2.StreamAccessLogsMessage_TcpLogs:
				if logEntries.TcpLogs != nil {
					als.mu.Lock()
					als.v2tcp = append(als.v2tcp, logEntries.TcpLogs.LogEntry...)
					als.mu.Unlock()
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

type ALSv3 struct {
	*GRPCALS
}

func (als ALSv3) StreamAccessLogs(srv alsv3.AccessLogService_StreamAccessLogsServer) error {
	for {
		msg, err := srv.Recv()
		if msg != nil {
			switch logEntries := msg.LogEntries.(type) {
			case *alsv3.StreamAccessLogsMessage_HttpLogs:
				if logEntries.HttpLogs != nil {
					als.mu.Lock()
					als.v3http = append(als.v3http, logEntries.HttpLogs.LogEntry...)
					als.mu.Unlock()
				}
			case *alsv3.StreamAccessLogsMessage_TcpLogs:
				if logEntries.TcpLogs != nil {
					als.mu.Lock()
					als.v3tcp = append(als.v3tcp, logEntries.TcpLogs.LogEntry...)
					als.mu.Unlock()
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
