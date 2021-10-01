package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
)

func TestAmbassadorMetaInfo(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, &snapshot.AmbassadorMetaInfo{ClusterID: "foo"})
	// Set some meta info we can check for.
	f.Upsert(&amb.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Name: "foo"},
		Spec:       amb.MappingSpec{Prefix: "/foo", Service: "1.2.3.4"},
	})
	f.Flush()
	snap := f.GetSnapshot(func(s *snapshot.Snapshot) bool { return true })
	assert.NotNil(t, snap)
	assert.Equal(t, "foo", snap.AmbassadorMeta.ClusterID)
}
