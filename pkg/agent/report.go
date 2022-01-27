package agent

import (
	"github.com/datawire/ambassador/v2/pkg/api/agent"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

// GetIdentity returns the Agent's CEPC Identity, if present, enabled, and
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
