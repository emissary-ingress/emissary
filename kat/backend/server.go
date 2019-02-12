package main

import (
	"log"
	"os"

	srv "github.com/datawire/ambassador/kat/backend/services"
)

const (
	// Crt certificate file.
	Crt = "server.crt"
	// Key private key file.
	Key = "server.key"
	// Port non-secure port.
	Port = 8080
	// SSLPort secure port.
	SSLPort = 8443
)

func main() {
	var s srv.Service

	switch t := os.Getenv("KAT_BACKEND_TYPE"); t {
	case "grpc":
		log.Fatal("gRPC backend is not implemented")
		return
		// s = &srv.GRPC{
		// 	Port:       Port,
		// 	SecurePort: SSLPort,
		// 	Cert:       Crt,
		// 	Key:        Key,
		// }

	case "grpc_auth":
		s = &srv.GRPCAUTH{
			Port:       Port,
			SecurePort: SSLPort,
			Cert:       Crt,
			Key:        Key,
		}

	default:
		s = &srv.HTTP{
			Port:       Port,
			SecurePort: SSLPort,
			Cert:       Crt,
			Key:        Key,
		}
	}

	c := s.Start()
	log.Printf("listening on ports: %v, %v", Port, SSLPort)
	<-c
}
