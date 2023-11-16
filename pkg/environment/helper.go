package environment

import (
	"context"
	"os"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
)

const (
	ambassadorRootEnvVar          = "APPDIR"
	ambassadorConfigBaseDirEnvVar = "AMBASSADOR_CONFIG_BASE_DIR"
	ambassadorClusterIdEnvVar     = "AMBASSADOR_CLUSTER_ID"
)

// EnvironmentSetupEntrypoint replicates the entrypoint.sh environment bootstrapping if the docker entrypoint was changed
func EnvironmentSetupEntrypoint(ctx context.Context) {
	clusterID := entrypoint.GetClusterID(ctx)
	os.Setenv(ambassadorClusterIdEnvVar, clusterID)
	dlog.Printf(ctx, "%s=%s", ambassadorClusterIdEnvVar, os.Getenv(ambassadorClusterIdEnvVar))
}
