package agent

import (
	"github.com/emissary-ingress/emissary/v3/pkg/api/agent"
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
