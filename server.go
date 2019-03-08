package main

import (
	"fmt"
	"log"
	"os"

	srv "github.com/datawire/kat-backend/services"
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
	listeners := make([]srv.Service, 0)
	var s srv.Service

	t := os.Getenv("KAT_BACKEND_TYPE")

	if len(t) <= 0 {
		t = "http"
	}

	log.Printf("Running as type %s", t)

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
		s = &srv.GRPCAUTH{
			Port:          Port,
			Backend:       os.Getenv("BACKEND"),
			SecurePort:    SSLPort,
			SecureBackend: os.Getenv("BACKEND"),
			Cert:          Crt,
			Key:           Key,
		}

		listeners = append(listeners, s)

	default:
		port := Port
		secure_port := SSLPort

		for {
			ename := fmt.Sprintf("BACKEND_%d", port)
			clear_backend := os.Getenv(ename)

			log.Printf("clear: checking %s -- %s", ename, clear_backend)

			if len(clear_backend) <= 0 {
				if port == 8080 {
					// Default for backwards compatibility.
					clear_backend = os.Getenv("BACKEND")

					log.Printf("clear: fallback to BACKEND -- %s", clear_backend)
				}
			}

			if len(clear_backend) <= 0 {
				log.Printf("clear: bailing, no backend")
				break
			}

			ename = fmt.Sprintf("BACKEND_%d", secure_port)
			secure_backend := os.Getenv(ename)

			log.Printf("secure: checking %s -- %s", ename, secure_backend)

			if len(secure_backend) <= 0 {
				if secure_port == 8443 {
					// Default for backwards compatibility.
					secure_backend = os.Getenv("BACKEND")

					log.Printf("secure: fallback to BACKEND -- %s", clear_backend)
				}
			}

			if len(secure_backend) <= 0 {
				log.Printf("secure: bailing, no backend")
				break
			}

			if clear_backend != secure_backend {
				log.Printf("BACKEND_%d and BACKEND_%d do not match", port, secure_port)
			} else {
				log.Printf("creating HTTP listener for %s on ports %d/%d", clear_backend, port, secure_port)

				s = &srv.HTTP{
					Port:          port,
					Backend:       clear_backend,
					SecurePort:    secure_port,
					SecureBackend: secure_backend,
					Cert:          Crt,
					Key:           Key,
				}
		
				listeners = append(listeners, s)
			}

			port++
			secure_port++
		}
	}

	if len(listeners) > 0 {
		var wait_for <-chan bool
		first := true
		
		for _, s := range listeners {
			// log.Printf("listening on ports: %v, %v", s.Port, s.SecurePort)
			c := s.Start()	
			
			if first {
				wait_for = c
				first = false
			}
		}

		<- wait_for
	} else {
		log.Fatal("no listeners, exiting")
	}
}
