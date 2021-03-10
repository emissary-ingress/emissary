package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	v2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
)

func TestAmbassadorMetaInfo(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})
	// Set some meta info we can check for.
	f.SetAmbassadorMeta(&snapshot.AmbassadorMetaInfo{ClusterID: "foo"})
	f.Upsert(&v2.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Name: "foo"},
		Spec:       v2.MappingSpec{Prefix: "/foo", Service: "1.2.3.4"},
	})
	f.Flush()
	snap := f.GetSnapshot(func(s *snapshot.Snapshot) bool { return true })
	assert.NotNil(t, snap)
	assert.Equal(t, "foo", snap.AmbassadorMeta.ClusterID)
}
