package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	pb "github.com/datawire/ambassador/kat/backend/echo"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

// GRPC server object (all fields are required).
type GRPC struct {
	Port       int16
	SecurePort int16
	Cert       string
	Key        string
}

// Start initializes the gRPC server.
func (g *GRPC) Start() <-chan bool {
	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%v", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			log.Fatal()
		}

		s := grpc.NewServer()
		pb.RegisterEchoServiceServer(s, &EchoService{})
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
		port := fmt.Sprintf(":%v", g.SecurePort)
		ln, err := tls.Listen(proto, port, config)
		if err != nil {
			log.Fatal(err)
			return
		}

		s := grpc.NewServer()
		pb.RegisterEchoServiceServer(s, &EchoService{})
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	log.Print("starting gRPC echo service")
	return exited
}

// EchoService implements envoy.service.auth.external_auth.
type EchoService struct{}

// Echo returns the an object with the HTTP context of the request.
func (s *EchoService) Echo(ctx context.Context, r *pb.EchoRequest) (*pb.EchoResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Code(13), "request has not valid context metadata")
	}

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

	// Checks scheme and set TLS info.
	if len(md[":scheme"]) > 0 && md[":scheme"][0] == "https" {
		request.Tls = &pb.TLS{
			Enabled: true,
		}
	}

	// Sets client requested metadata.
	if len(md["requested-headers"]) > 0 {
		for _, v := range md["requested-headers"] {
			if len(md[v]) > 0 {
				strval := strings.Join(md[v], ",")
				response.Headers[v] = strval
				header := metadata.Pairs(v, strval)
				grpc.SendHeader(ctx, header)
			}
		}
	}

	// Sets grpc response.
	echoRES := &pb.EchoResponse{
		Backend:  os.Getenv("BACKEND"),
		Request:  request,
		Response: response,
	}

	// Set a log message.
	if body, err := json.MarshalIndent(echoRES, "", "  "); err == nil {
		log.Printf("setting response: %s", string(body))
	}

	// Checks if requested-status is a valid and not OK gRPC status.
	if len(md["requested-status"]) == 1 {
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
