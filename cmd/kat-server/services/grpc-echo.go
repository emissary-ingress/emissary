package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	// "os"
	"strconv"
	"strings"

	pb "github.com/datawire/ambassador/pkg/api/kat"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

// GRPC server object (all fields are required).
type GRPC struct {
	Port          int16
	Backend       string
	SecurePort    int16
	SecureBackend string
	Cert          string
	Key           string
}

// DefaultOpts sets gRPC service options.
func DefaultOpts() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 5),
		grpc.MaxSendMsgSize(1024 * 1024 * 5),
	}
}

// Start initializes the gRPC server.
func (g *GRPC) Start() <-chan bool {
	log.Printf("GRPC: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%d", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			log.Fatal()
		}

		s := grpc.NewServer(DefaultOpts()...)
		pb.RegisterEchoServiceServer(s, g)
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	go func() {
		cer, err := tls.LoadX509KeyPair(g.Cert, g.Key)
		if err != nil {
			log.Fatal(err)
			return
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		port := fmt.Sprintf(":%d", g.SecurePort)
		ln, err := tls.Listen(proto, port, config)
		if err != nil {
			log.Fatal(err)
			return
		}

		s := grpc.NewServer(DefaultOpts()...)
		pb.RegisterEchoServiceServer(s, g)
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	log.Print("starting gRPC echo service")
	return exited
}

// Echo returns the an object with the HTTP context of the request.
func (g *GRPC) Echo(ctx context.Context, r *pb.EchoRequest) (*pb.EchoResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Code(13), "request has not valid context metadata")
	}

	buf := bytes.Buffer{}
	buf.WriteString("metadata received: \n")
	for k, v := range md {
		buf.WriteString(fmt.Sprintf("%v : %s\n", k, strings.Join(v, ",")))
	}
	log.Println(buf.String())

	request := &pb.Request{
		Headers: make(map[string]string),
	}

	response := &pb.Response{
		Headers: make(map[string]string),
	}

	// Sets request headers.
	for k, v := range md {
		request.Headers[k] = strings.Join(v, ",")
		response.Headers[k] = strings.Join(v, ",")
	}

	// Set default backend and assume we're the clear side of the world.
	backend := g.Backend

	// Checks scheme and set TLS info.
	if len(md["x-forwarded-proto"]) > 0 && md["x-forwarded-proto"][0] == "https" {
		// We're the secure side of the world, I guess.
		backend = g.SecureBackend
		request.Tls = &pb.TLS{
			Enabled: true,
		}
	}

	// Check header and delay response.
	if h, ok := md["Requested-Backend-Delay"]; ok {
		if v, err := strconv.Atoi(h[0]); err == nil {
			log.Printf("Delaying response by %v ms", v)
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}

	// Set response date header.
	response.Headers["date"] = time.Now().Format(time.RFC1123)

	// Sets client requested metadata.
	if len(md["requested-headers"]) > 0 {
		for _, v := range md["requested-headers"] {
			if len(md[v]) > 0 {
				s := strings.Join(md[v], ",")
				response.Headers[v] = s
				p := metadata.Pairs(v, s)
				grpc.SendHeader(ctx, p)
			}
		}
	}

	// Sets grpc response.
	echoRES := &pb.EchoResponse{
		Backend:  backend,
		Request:  request,
		Response: response,
	}

	// Set a log message.
	if data, err := json.MarshalIndent(echoRES, "", "  "); err == nil {
		log.Printf("setting response: %s\n", string(data))
	}

	// Checks if requested-status is a valid and not OK gRPC status.
	if len(md["requested-status"]) > 0 {
		val, err := strconv.Atoi(md["requested-status"][0])
		if err == nil {
			if val < 18 || val > 0 {
				// Return response and the not OK status.
				return echoRES, status.Error(codes.Code(val), "requested-error")
			}
		}
	}

	// Returns response and the OK status.
	return echoRES, nil
}
