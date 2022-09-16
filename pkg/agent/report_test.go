package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/agent"
	agentTypes "github.com/datawire/ambassador/v2/pkg/api/agent"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func TestGetIdentity(t *testing.T) {
	defaultHost := "defaulthost"
	getIdentityTests := []struct {
		testName         string
		cfg              *snapshotTypes.AmbassadorMetaInfo
		expectedIdentity *agentTypes.Identity
		hostname         *string
	}{
		{
			// GetIdentity should copy info from the ambassador meta to populate the
			// Identity
			testName: "basic",
			expectedIdentity: &agentTypes.Identity{
				Label:     "",
				ClusterId: "reallyspecialclusterid",
				Version:   "",
				Hostname:  defaultHost,
			},
			cfg: &snapshotTypes.AmbassadorMetaInfo{
				ClusterID:         "reallyspecialclusterid",
				AmbassadorVersion: "v1.0",
			},
		},
		{
			// if our config is nil, GetIdentity should return nil
			testName:         "nil-config",
			expectedIdentity: nil,
			cfg:              nil,
		},
		{
			// If the meta info doesn't contain any identifying info, GetIdenity should
			// return nil
			testName:         "empty-cepc",
			expectedIdentity: nil,
			cfg: &snapshotTypes.AmbassadorMetaInfo{
				ClusterID:         "",
				AmbassadorVersion: "v1.0",
			},
		},
	}
	for _, testcase := range getIdentityTests {
		t.Run(testcase.testName, func(innerT *testing.T) {
			var host string
			if testcase.hostname == nil {
				host = defaultHost
			} else {
				host = *testcase.hostname
			}
			identity := agent.GetIdentity(testcase.cfg, host)

			assert.Equal(innerT, testcase.expectedIdentity, identity)
		})
	}
}
