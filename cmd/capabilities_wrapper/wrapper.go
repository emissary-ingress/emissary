package main

import (
	"context"
	"os"
	"syscall"

	"github.com/datawire/dlib/dlog"
)

func main() {
	ctx := context.Background()
	dlog.Println(ctx, "Starting Envoy with CAP_NET_BIND_SERVICE capability")

	if err := capset(); err != nil {
		dlog.Error(ctx, err)
		os.Exit(126)
	}

	dlog.Println(ctx, "Succeeded in setting capabilities")

	if err := syscall.Exec("/usr/local/bin/envoy", os.Args, os.Environ()); err != nil {
		dlog.Error(ctx, err)
		os.Exit(127)
	}
}
