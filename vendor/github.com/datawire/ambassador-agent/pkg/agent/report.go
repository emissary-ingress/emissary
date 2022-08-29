package agent

import (
	"github.com/datawire/ambassador-agent/pkg/api/agent"
	diagnosticsTypes "github.com/emissary-ingress/emissary/v3/pkg/diagnostics/v1"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// GetIdentity returns the Agent's DCP Identity, if present, enabled, and
// configured by the user.
func GetIdentity(ambassadorMeta *snapshotTypes.AmbassadorMetaInfo, ambHost string) *agent.Identity {
	if ambassadorMeta == nil || ambassadorMeta.ClusterID == "" {
		// No Ambassador module -> no identity -> no reporting
		return nil
	}

	return &agent.Identity{
		ClusterId: ambassadorMeta.ClusterID,
		Hostname:  ambHost,
	}
}

// GetIdentityFromDiagnostics returns the Agent's DCP Identity, if present
func GetIdentityFromDiagnostics(ambSystem *diagnosticsTypes.System, ambHost string) *agent.Identity {
	if ambSystem == nil || ambSystem.ClusterID == "" {
		// No Ambassador module -> no identity -> no reporting
		return nil
	}

	return &agent.Identity{
		ClusterId: ambSystem.ClusterID,
		Hostname:  ambHost,
	}
}
