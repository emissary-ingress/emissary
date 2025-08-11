package entrypoint_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/emissary-ingress.dev/v4alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

func TestAmbassadorMetaInfo(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, &snapshot.AmbassadorMetaInfo{ClusterID: "foo"})
	// Set some meta info we can check for.
	require.NoError(t, f.Upsert(&amb.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Name: "foo"},
		Spec:       amb.MappingSpec{Prefix: "/foo", Service: "1.2.3.4"},
	}))
	f.Flush()
	snap, err := f.GetSnapshot(func(s *snapshot.Snapshot) bool { return true })
	require.NoError(t, err)
	assert.NotNil(t, snap)
	assert.Equal(t, "foo", snap.AmbassadorMeta.ClusterID)
}
