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
	apiv2_accesslog "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/data/accesslog/v2"
	apiv3_accesslog "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/data/accesslog/v3"
	apiv2_svc_accesslog "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/accesslog/v2"
	apiv3_svc_accesslog "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/accesslog/v3"
)

type GRPCALS struct {
	HTTPListener

	mu     sync.Mutex
	v2http []*apiv2_accesslog.HTTPAccessLogEntry
	v2tcp  []*apiv2_accesslog.TCPAccessLogEntry
	v3http []*apiv3_accesslog.HTTPAccessLogEntry
	v3tcp  []*apiv3_accesslog.TCPAccessLogEntry
}

func (als *GRPCALS) Start(ctx context.Context) <-chan bool {
	httpHandler := http.NewServeMux()
	httpHandler.HandleFunc("/logs", als.ServeLogs)

	grpcHandler := grpc.NewServer()
	apiv2_svc_accesslog.RegisterAccessLogServiceServer(grpcHandler, ALSv2{als})
	apiv3_svc_accesslog.RegisterAccessLogServiceServer(grpcHandler, ALSv3{als})

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

func (als ALSv2) StreamAccessLogs(srv apiv2_svc_accesslog.AccessLogService_StreamAccessLogsServer) error {
	for {
		msg, err := srv.Recv()
		if msg != nil {
			switch logEntries := msg.LogEntries.(type) {
			case *apiv2_svc_accesslog.StreamAccessLogsMessage_HttpLogs:
				if logEntries.HttpLogs != nil {
					als.mu.Lock()
					als.v2http = append(als.v2http, logEntries.HttpLogs.LogEntry...)
					als.mu.Unlock()
				}
			case *apiv2_svc_accesslog.StreamAccessLogsMessage_TcpLogs:
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

func (als ALSv3) StreamAccessLogs(srv apiv3_svc_accesslog.AccessLogService_StreamAccessLogsServer) error {
	for {
		msg, err := srv.Recv()
		if msg != nil {
			switch logEntries := msg.LogEntries.(type) {
			case *apiv3_svc_accesslog.StreamAccessLogsMessage_HttpLogs:
				if logEntries.HttpLogs != nil {
					als.mu.Lock()
					als.v3http = append(als.v3http, logEntries.HttpLogs.LogEntry...)
					als.mu.Unlock()
				}
			case *apiv3_svc_accesslog.StreamAccessLogsMessage_TcpLogs:
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
