package entrypoint_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	v3bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	v3cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

func ClusterHasAltName(altName string) func(*v3cluster.Cluster) bool {
	return func(c *v3cluster.Cluster) bool {
		return c.AltStatName == altName
	}
}

func TestFakeCollision(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	assert.NoError(t, f.UpsertFile("testdata/Collision1.yaml"))
	f.Flush()

	snap, err := f.GetSnapshot(HasMapping("staging", "subway-staging-socket-stable-mapping"))
	require.NoError(t, err)

	// assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.NotNil(t, snap)

	// The first time we look at the Envoy config, we should find only three clusters:
	// one with an AltStatName of "127_0_0_1_8877", one with an AltStatName of
	// "subway_staging_stable_staging_3000", and one with an AltStatName of
	// "subway_staging_stable_staging_3001".

	envoyConfig, err := f.GetEnvoyConfig(func(config *v3bootstrap.Bootstrap) bool {
		// Three clusters...
		if len(config.StaticResources.Clusters) != 3 {
			return false
		}

		// ...each of which has an AltStatName that we like.
		foundClusters := map[string]bool{}

		for _, cluster := range config.StaticResources.Clusters {
			foundClusters[cluster.AltStatName] = true
		}

		// Make sure we found exactly the clusters we need.
		if !foundClusters["127_0_0_1_8877"] ||
			!foundClusters["subway_staging_stable_staging_3000"] ||
			!foundClusters["subway_staging_stable_staging_3001"] {
			return false
		}

		return true
	})

	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	LogJSON(t, envoyConfig)

	// Once here, we _must_ have found the two clusters we wanted. Grab them and take
	// a look inside to make sure all is well.

	for _, cluster := range envoyConfig.StaticResources.Clusters {
		LogJSON(t, cluster)
	}

	c0 := FindCluster(envoyConfig, ClusterHasAltName("subway_staging_stable_staging_3000"))
	assert.NotNil(t, c0)

	c1 := FindCluster(envoyConfig, ClusterHasAltName("subway_staging_stable_staging_3001"))
	assert.NotNil(t, c1)

	// OK. c0 should get its load assignments from EDS with a key of
	// k8s/staging/subway-staging-stable/3000, and it should have a name that starts with
	// cluster_subway_staging_stable_staging_30- and ends with -0.

	assert.Equal(t, "k8s/staging/subway-staging-stable/3000", c0.EdsClusterConfig.ServiceName)

	assert.True(t, strings.HasPrefix(c0.Name, "cluster_subway_staging_stable_staging_30-"))
	assert.True(t, strings.HasSuffix(c0.Name, "-0"))

	// For c1, we need load assignments from EDS with a key of
	// k8s/staging/subway-staging-stable/3001, and its name should have... the same
	// prefix and suffix as the first cluster did, actually!

	assert.Equal(t, "k8s/staging/subway-staging-stable/3001", c1.EdsClusterConfig.ServiceName)

	assert.True(t, strings.HasPrefix(c1.Name, "cluster_subway_staging_stable_staging_30-"))
	assert.True(t, strings.HasSuffix(c1.Name, "-0"))

	// Finally, no two clusters should have the same name -- even though c0 and c1 have names
	// with the same prefix and suffix, they must be different.

	nameMap := map[string]int{}

	for _, cluster := range envoyConfig.StaticResources.Clusters {
		nameMap[cluster.Name]++
		assert.Equal(t, 1, nameMap[cluster.Name])
	}

	// Next up: add another Mapping to subway-staging-stable.staging:3000.
	assert.NoError(t, f.UpsertFile("testdata/Collision2.yaml"))
	f.Flush()

	snap, err = f.GetSnapshot(HasMapping("staging", "subway-staging-socket-stable-mapping"))

	// assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.NotNil(t, snap)

	// Here, we need _four_ clusters: yes, we already have a Mapping that uses
	// subway-staging-stable.staging:3000, but it uses it differently so we should have
	// created a new cluster.
	//
	// We will still only three AltStatNames, though! The new cluster should have an
	// AltStatName of "subway_staging_stable_staging_3000", like one of our original
	// clusters.

	envoyConfig, err = f.GetEnvoyConfig(func(config *v3bootstrap.Bootstrap) bool {
		// Four clusters...
		if len(config.StaticResources.Clusters) != 4 {
			return false
		}

		// ...each of which has an AltStatName that we like.
		foundClusters := map[string]bool{}

		for _, cluster := range config.StaticResources.Clusters {
			foundClusters[cluster.AltStatName] = true
		}

		// Make sure we found exactly the clusters we need.
		if !foundClusters["127_0_0_1_8877"] ||
			!foundClusters["subway_staging_stable_staging_3000"] ||
			!foundClusters["subway_staging_stable_staging_3001"] {
			return false
		}

		return true
	})

	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	LogJSON(t, envoyConfig)

	// Once here, we _must_ have found the three clusters we're really interested in. Grab them
	// and take a look inside to make sure all is well.
	//
	// However, since we have two clusters with AltStatName "subway_staging_stable_staging_3000",
	// we can't just grab clusters by AltStatName like we did the first time. Instead, we'll
	// iterate over clusters and figure out what we have.

	nameMap = map[string]int{}
	altNameMap := map[string]int{}

	for _, cluster := range envoyConfig.StaticResources.Clusters {
		// No two clusters should have the same name.
		nameMap[cluster.Name]++
		assert.Equal(t, 1, nameMap[cluster.Name])

		// But! two clusters should have the same AltStatName. So. First, bump the altNameMap
		// for this cluster's AltStatName.
		altNameMap[cluster.AltStatName]++

		// Next, if this isn't the 127_0_0_1_8877 cluster, make sure its AltStatName has the
		// correct prefix and suffix (and yes, the suffixes should all be -0 now -- we should
		// honestly just get rid of those).

		if cluster.AltStatName != "127_0_0_1_8877" {
			assert.True(t, strings.HasPrefix(cluster.Name, "cluster_subway_staging_stable_staging_30-"))
			assert.True(t, strings.HasSuffix(cluster.Name, "-0"))
		}
	}

	// Once here, we know that we have four clusters, we know they have unique names, we've
	// counted the AltStatNames, and we know the AltStatNames have the correct prefix and suffix.
	// All that's left is to verify the counts of the AltStatNames.

	assert.Equal(t, 1, altNameMap["127_0_0_1_8877"])
	assert.Equal(t, 2, altNameMap["subway_staging_stable_staging_3000"])
	assert.Equal(t, 1, altNameMap["subway_staging_stable_staging_3001"])
}
