package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/firstboot"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func main() {
	ctx := middleware.WithLogger(context.Background(), types.WrapLogrus(logrus.New()))
	server := &http.Server{
		Addr:        ":8080",
		Handler:     firstboot.NewFirstBootWizard(),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	log.Fatal(server.ListenAndServe())
}
