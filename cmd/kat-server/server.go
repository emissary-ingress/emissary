package main

import (
	"context"
	"fmt"
	"os"

	srv "github.com/datawire/ambassador/v2/cmd/kat-server/services"
	"github.com/datawire/dlib/dlog"
)

const (
	// Crt certificate file.
	Crt = "server.crt"
	// Key private key file.
	Key = "server.key"
	// Port non-secure port.
	Port int16 = 8080
	// SSLPort secure port.
	SSLPort int16 = 8443
)

func main() {
	ctx := context.Background() // first line in main()
	listeners := make([]srv.Service, 0)
	var s srv.Service

	t := os.Getenv("KAT_BACKEND_TYPE")

	if len(t) <= 0 {
		t = "http"
	}

	dlog.Printf(ctx, "Running as type %s", t)

	switch t {
	case "grpc_echo":
		s = &srv.GRPC{
			Port:          Port,
			Backend:       os.Getenv("BACKEND"),
			SecurePort:    SSLPort,
			SecureBackend: os.Getenv("BACKEND"),
			Cert:          Crt,
			Key:           Key,
		}

		listeners = append(listeners, s)

	case "grpc_auth":
		protocolVersion := os.Getenv("GRPC_AUTH_PROTOCOL_VERSION")
		if protocolVersion == "v2" {
			s = &srv.GRPCAUTH{
				Port:            Port,
				Backend:         os.Getenv("BACKEND"),
				SecurePort:      SSLPort,
				SecureBackend:   os.Getenv("BACKEND"),
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: protocolVersion,
			}
		} else {
			s = &srv.GRPCAUTHV3{
				Port:            Port,
				Backend:         os.Getenv("BACKEND"),
				SecurePort:      SSLPort,
				SecureBackend:   os.Getenv("BACKEND"),
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: protocolVersion,
			}
		}

		listeners = append(listeners, s)

	case "grpc_rls":
		protocolVersion := os.Getenv("GRPC_RLS_PROTOCOL_VERSION")
		if protocolVersion == "v2" {
			s = &srv.GRPCRLS{
				Port:            Port,
				Backend:         os.Getenv("BACKEND"),
				SecurePort:      SSLPort,
				SecureBackend:   os.Getenv("BACKEND"),
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: protocolVersion,
			}
		} else {
			s = &srv.GRPCRLSV3{
				Port:            Port,
				Backend:         os.Getenv("BACKEND"),
				SecurePort:      SSLPort,
				SecureBackend:   os.Getenv("BACKEND"),
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: protocolVersion,
			}
		}

		listeners = append(listeners, s)
	case "grpc_agent":
		s = &srv.GRPCAgent{
			Port: Port,
		}
		listeners = append(listeners, s)

	default:
		port := Port
		securePort := SSLPort

		for {
			eName := fmt.Sprintf("BACKEND_%d", port)
			clearBackend := os.Getenv(eName)

			dlog.Printf(ctx, "clear: checking %s -- %s", eName, clearBackend)

			if len(clearBackend) <= 0 {
				if port == 8080 {
					// Default for backwards compatibility.
					clearBackend = os.Getenv("BACKEND")

					dlog.Printf(ctx, "clear: fallback to BACKEND -- %s", clearBackend)
				}
			}

			if len(clearBackend) <= 0 {
				dlog.Printf(ctx, "clear: bailing, no backend")
				break
			}

			eName = fmt.Sprintf("BACKEND_%d", securePort)
			secureBackend := os.Getenv(eName)

			dlog.Printf(ctx, "secure: checking %s -- %s", eName, secureBackend)

			if len(secureBackend) <= 0 {
				if securePort == 8443 {
					// Default for backwards compatibility.
					secureBackend = os.Getenv("BACKEND")

					dlog.Printf(ctx, "secure: fallback to BACKEND -- %s", clearBackend)
				}
			}

			if len(secureBackend) <= 0 {
				dlog.Printf(ctx, "secure: bailing, no backend")
				break
			}

			if clearBackend != secureBackend {
				dlog.Printf(ctx, "BACKEND_%d and BACKEND_%d do not match", port, securePort)
			} else {
				dlog.Printf(ctx, "creating HTTP listener for %s on ports %d/%d", clearBackend, port, securePort)

				s = &srv.HTTP{
					Port:          port,
					Backend:       clearBackend,
					SecurePort:    securePort,
					SecureBackend: secureBackend,
					Cert:          Crt,
					Key:           Key,
				}

				listeners = append(listeners, s)
			}

			port++
			securePort++
		}
	}

	if len(listeners) > 0 {
		var waitFor <-chan bool
		first := true

		for _, s := range listeners {
			c := s.Start(ctx)

			if first {
				waitFor = c
				first = false
			}
		}

		<-waitFor
	} else {
		dlog.Error(ctx, "no listeners, exiting")
		os.Exit(1)
	}
}
