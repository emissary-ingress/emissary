package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	srv "github.com/datawire/ambassador/v2/cmd/kat-server/services"
	"github.com/datawire/dlib/dlog"
	"github.com/datawire/envconfig"
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

type EnvConfig struct {
	Type             string `env:"KAT_BACKEND_TYPE  ,parser=nonempty-string        ,default=http"`
	Backend          string `env:"BACKEND           ,parser=possibly-empty-string  ,default="`

	// services/http.go
	AddExtAuth bool `env:"INCLUDE_EXTAUTH_HEADER ,parser=empty-nonempty ,default="`

	// services/grpc-agent.go
	GRPCMaxRecvMsgSize int `env:"KAT_GRPC_MAX_RECV_MESSAGE_SIZE  ,parser=strconv.ParseInt  ,default=0"`

	AuthProtocolVersion string `env:"GRPC_AUTH_PROTOCOL_VERSION  ,parser=nonempty-string  ,default=v2"`
	RLSProtocolVersion  string `env:"GRPC_RLS_PROTOCOL_VERSION   ,parser=nonempty-string  ,default=v2"`
}

func ConfigFromEnv() (cfg EnvConfig, warn []error, fatal []error) {
	parser, err := envconfig.GenerateParser(reflect.TypeOf(EnvConfig{}), nil)
	if err != nil {
		// panic, because it means that the definition of
		// 'Config' is invalid, which is a bug, not a
		// runtime error.
		panic(err)
	}
	warn, fatal = parser.ParseFromEnv(&cfg)
	return
}

func main() {
	ctx := context.Background() // first line in main()

	cfg, warn, fatal := ConfigFromEnv()
	for _, err := range warn {
		dlog.Warnln(ctx, "config error:", err)
	}
	for _, err := range fatal {
		dlog.Errorln(ctx, "config error:", err)
	}
	if len(fatal) > 0 {
		os.Exit(1)
	}

	listeners := make([]srv.Service, 0)
	var s srv.Service

	dlog.Printf(ctx, "Running as type %s", cfg.Type)

	switch cfg.Type {
	case "grpc_echo":
		s = &srv.GRPC{
			Port:          Port,
			Backend:       cfg.Backend,
			SecurePort:    SSLPort,
			SecureBackend: cfg.Backend,
			Cert:          Crt,
			Key:           Key,
		}

		listeners = append(listeners, s)

	case "grpc_auth":
		switch cfg.AuthProtocolVersion {
		case "v3":
			s = &srv.GRPCAUTHV3{
				Port:            Port,
				Backend:         cfg.Backend,
				SecurePort:      SSLPort,
				SecureBackend:   cfg.Backend,
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: cfg.AuthProtocolVersion,
			}
		case "v2":
			s = &srv.GRPCAUTH{
				Port:            Port,
				Backend:         cfg.Backend,
				SecurePort:      SSLPort,
				SecureBackend:   cfg.Backend,
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: cfg.AuthProtocolVersion,
			}
		default:
			dlog.Errorf(ctx, "invalid auth protocol version: %q", cfg.AuthProtocolVersion)
			os.Exit(1)
		}

		listeners = append(listeners, s)

	case "grpc_rls":
		switch cfg.RLSProtocolVersion {
		case "v3":
			s = &srv.GRPCRLSV3{
				Port:            Port,
				Backend:         cfg.Backend,
				SecurePort:      SSLPort,
				SecureBackend:   cfg.Backend,
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: cfg.RLSProtocolVersion,
			}
		case "v2":
			s = &srv.GRPCRLS{
				Port:            Port,
				Backend:         cfg.Backend,
				SecurePort:      SSLPort,
				SecureBackend:   cfg.Backend,
				Cert:            Crt,
				Key:             Key,
				ProtocolVersion: cfg.RLSProtocolVersion,
			}
		default:
			dlog.Errorf(ctx, "invalid rls protocol version: %q", cfg.RLSProtocolVersion)
			os.Exit(1)
		}

		listeners = append(listeners, s)
	case "grpc_agent":
		s = &srv.GRPCAgent{
			Port:               Port,
			GRPCMaxRecvMsgSize: cfg.GRPCMaxRecvMsgSize,
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
					clearBackend = cfg.Backend

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
					secureBackend = cfg.Backend

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
					AddExtAuth:    cfg.AddExtAuth,
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
