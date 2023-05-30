package apiext

import (
	"context"
	"fmt"
	"os"

	"github.com/datawire/dlib/dlog"
	crdAll "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext"
)

// Main is a `github.com/emissary-ingress/emissary/v3/pkg/busy`-compatible wrapper around 'Run()', using
// values appropriate for the stock Emissary.
func Main(ctx context.Context, version string, args ...string) error {
	dlog.Infof(ctx, "Emissary Ingress apiext (version %q)", version)
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "%s: error: expected exactly one argument, got %d\n", os.Args[0], len(args))
		fmt.Fprintf(os.Stderr, "Usage: %s APIEXT_SVCNAME\n", os.Args[0])
		os.Exit(2)
	}

	serviceName := args[0]
	scheme := crdAll.BuildScheme()

	webhookServer := apiext.NewWebhookServer(apiext.WebhookServerConfig{
		ServiceName: serviceName,
		HTTPPort:    8080,
		HTTPSPort:   8443,
	})

	return webhookServer.Run(ctx, scheme)
}
