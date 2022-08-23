package services

import (
	// stdlib
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	// third party
	"google.golang.org/grpc"

	// first party
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

// Service defines a KAT backend service interface.
type Service interface {
	Start(context.Context) <-chan bool
}

type HTTPListener struct {
	CleartextPort int16
	TLSPort       int16
	TLSCert       string
	TLSKey        string
}

func (hl HTTPListener) Run(ctx context.Context, name string, httpHandler *http.ServeMux, grpcHandler *grpc.Server) <-chan bool {
	dlog.Printf(ctx, "GRPCRLS: %s listening on cleartext :%d and tls :%d", name, hl.CleartextPort, hl.TLSPort)

	cer, err := tls.LoadX509KeyPair(hl.TLSCert, hl.TLSKey)
	if err != nil {
		dlog.Error(ctx, err)
		panic(err) // TODO: do something better
	}

	sc := &dhttp.ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dlog.Infof(r.Context(), "handling request to %q...", r.RequestURI)
			if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
				grpcHandler.ServeHTTP(w, r)
			} else {
				httpHandler.ServeHTTP(w, r)
			}
		}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cer},
		},
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
	grp.Go("cleartext", func(ctx context.Context) error {
		return sc.ListenAndServe(ctx, fmt.Sprintf(":%v", hl.CleartextPort))
	})
	grp.Go("tls", func(ctx context.Context) error {
		return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%v", hl.TLSPort), "", "")
	})

	dlog.Printf(ctx, "starting %s", name)

	exited := make(chan bool)
	go func() {
		if err := grp.Wait(); err != nil {
			dlog.Error(ctx, err)
			panic(err) // TODO: do something better
		}
		close(exited)
	}()
	return exited
}
