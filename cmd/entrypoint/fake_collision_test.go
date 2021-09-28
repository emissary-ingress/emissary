package entrypoint_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v2"
	"github.com/stretchr/testify/assert"
)

func TestFakeCollision(t *testing.T) {
	// This test will not pass in legacy mode because diagd will not emit EDS clusters in legacy mode.
	if legacy, err := strconv.ParseBool(os.Getenv("AMBASSADOR_LEGACY_MODE")); err == nil && legacy {
		return
	}

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	f.UpsertFile("testdata/Collision1.yaml")
	f.Flush()

	snap := f.GetSnapshot(HasMapping("staging", "subway-staging-socket-stable-mapping"))

	// assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.NotNil(t, snap)

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		// The first time we look at the Envoy config, we should find only two clusters.
		//
		// First up, a cluster named cluster_subway_staging_stable_staging_30-0,
		// which should get its load assignments from EDS with a key of
		// k8s/staging/subway-staging-stable/3000.
		c0 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-0"))

		if c0 == nil {
			return false
		}

		if c0.EdsClusterConfig.ServiceName != "k8s/staging/subway-staging-stable/3000" {
			return false
		}

		// We also need a cluster named cluster_subway_staging_stable_staging_30-1,
		// which should get its load assignments from EDS with a key of
		// k8s/staging/subway-staging-stable/3001.
		c1 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-1"))

		if c1 == nil {
			return false
		}

		if c1.EdsClusterConfig.ServiceName != "k8s/staging/subway-staging-stable/3001" {
			return false
		}

		// We need to _not_ have a cluster named cluster_subway_staging_stable_staging_30-2.

		c2 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-2"))

		if c2 != nil {
			return false
		}

		return true
	})

	assert.NotNil(t, envoyConfig)

	fmt.Println(Jsonify(envoyConfig))

	f.UpsertFile("testdata/Collision2.yaml")
	f.Flush()

	snap = f.GetSnapshot(HasMapping("staging", "subway-staging-socket-stable-mapping"))

	// assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.NotNil(t, snap)

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig = f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		// The second time we look at the Envoy config, we need to see three
		// clusters, but note that some of the contents of the clusters will have
		// changed.
		//
		// We still need a cluster named cluster_subway_staging_stable_staging_30-0,
		// and it should still get its load assignments from EDS with a key of
		// k8s/staging/subway-staging-stable/3000.
		c0 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-0"))

		if c0 == nil {
			return false
		}

		if c0.EdsClusterConfig.ServiceName != "k8s/staging/subway-staging-stable/3000" {
			return false
		}

		// We still need a cluster named cluster_subway_staging_stable_staging_30-1,
		// but its load assignments should now also come from EDS with a key of
		// k8s/staging/subway-staging-stable/3000.
		c1 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-1"))

		if c1 == nil {
			return false
		}

		if c1.EdsClusterConfig.ServiceName != "k8s/staging/subway-staging-stable/3000" {
			return false
		}

		// Finally, we need a cluster named cluster_subway_staging_stable_staging_30-2,
		// with load assignments coming from EDS with a key of
		// k8s/staging/subway-staging-stable/3001.
		c2 := FindCluster(config, ClusterNameContains("cluster_subway_staging_stable_staging_30-2"))

		if c2 == nil {
			return false
		}

		if c2.EdsClusterConfig.ServiceName != "k8s/staging/subway-staging-stable/3001" {
			return false
		}

		return true
	})

	assert.NotNil(t, envoyConfig)

	fmt.Println(Jsonify(envoyConfig))
}
