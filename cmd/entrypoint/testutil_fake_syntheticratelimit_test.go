package entrypoint_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	v3bootstrap "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/bootstrap/v3"
	v3cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// This predicate is used to check k8s snapshots for an RateLimitService matching the provided name and
// namespace.
func HasRateLimitService(namespace, name string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.RateLimitServices {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}
		return false
	}
}

// Tests the synthetic rateLimit generation when a valid RateLimitService is created.  This RateLimitService has
// `protocol_version: v3` and should not be replaced by the synthetic RateLimitService.
func TestSyntheticRateLimitValid(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  protocol_version: "v3"
  service: 127.0.0.1:8500
`)
			assert.NoError(t, err)
			f.Flush()

			// Use the predicate above to check that the snapshot contains the
			// RateLimitService defined above.  The RateLimitService has `protocol_version: v3` so
			// it should not be removed/replaced by the synthetic RateLimitService injected
			// by syntheticratelimit.go
			snap, err := f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			// In edge-stack we should only ever have 1 RateLimitService.
			assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
			assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)

			// Check for a cluster name matching the provided RateLimitService
			isRateLimitCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
			}

			// Grab the next Envoy config that has an Edge Stack ratelimit cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isRateLimitCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a cluster for the
			// RateLimitService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// This tests with a provided RateLimitService that has no protocol_version (which defaults to v2).  It
// should get forcibly overridden to be v3.
func TestSyntheticRateLimitReplace(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
`)
			assert.NoError(t, err)
			f.Flush()

			// The RateLimitService does not have `protocol_version: v3` so it should be
			// forcibly edited to say `protocol_version: v3` by syntheticratelimit.go
			snap, err := f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			// In edge-stack we should only ever have 1 RateLimitService.
			assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
			// The snapshot should only have the one defined above.
			assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)
			// The protocol version should be forcibly set to v3.
			assert.Equal(t, "v3", snap.Kubernetes.RateLimitServices[0].Spec.ProtocolVersion)

			// Check for a cluster name matching the provided RateLimitService
			isRateLimitCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
			}

			// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isRateLimitCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a cluster for the
			// RateLimitService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// Tests the synthetic rateLimit generation when an invalid RateLimitService is created.  This RateLimitService has
// `protocol_version: v3` and should not be replaced by the synthetic RateLimitService even though it has
// a bogus value because the bogus field will be dropped when it is loaded, and we will be left with
// a valid RateLimitService.
func TestSyntheticRateLimitBogusField(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  bogus_field: "foo"
`)
			assert.NoError(t, err)
			f.Flush()

			// Use the predicate above to check that the snapshot contains the
			// RateLimitService defined above.  The RateLimitService has `protocol_version: v3` so
			// it should not be removed/replaced by the synthetic RateLimitService injected
			// by syntheticratelimit.go
			snap, err := f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			// In edge-stack we should only ever have 1 RateLimitService.
			assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
			assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)

			// Check for a cluster name matching the provided RateLimitService
			isRateLimitCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
			}

			// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isRateLimitCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a cluster for the
			// RateLimitService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// Tests the synthetic rateLimit generation when an invalid RateLimitService (because the protocol_version is
// invalid for the supported enums).  This RateLimitService should be tossed out and the synthetic
// RateLimitService should be injected.
func TestSyntheticRateLimitInvalidProtocolVer(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  protocol_version: "vBogus"
  bogus_field: "foo"
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the synthetic RateLimitService.
	// The RateLimitService has `protocol_version: v3`, but it has a bogus field, so it should not be
	// validated, and instead we inject the synthetic RateLimitService.
	snap, err := f.GetSnapshot(HasRateLimitService("default", "synthetic_edge_stack_rate_limit"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// In edge-stack we should only ever have 1 RateLimitService.
	assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
	// The snapshot should only have the synthetic RateLimitService and not the one defined above.
	assert.Equal(t, "synthetic_edge_stack_rate_limit", snap.Kubernetes.RateLimitServices[0].Name)

	// Check for a cluster name matching the provided RateLimitService
	isRateLimitCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
	// 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isRateLimitCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a cluster for the
	// RateLimitService that was defined.
	assert.NotNil(t, envoyConfig)
}

// Tests the synthetic rateLimit generation when an invalid RateLimitService is created and edited several
// times in succession.  After the config is edited several times, we should see that the final
// result is our provided valid RateLimitService.  There should not be any duplicate RateLimitService
// resources, and the synthetic RateLimitService that gets created when the first invalid RateLimitService is
// applied should be removed when the final edit makes it a valid RateLimitService.
func TestSyntheticRateLimitChurn(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.AutoFlush(true)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  protocol_version: "v3"
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  protocol_version: "v3"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the RateLimitService defined
	// above.  The RateLimitService has `protocol_version: v3` so it should not be removed/replaced
	// by the synthetic RateLimitService injected by syntheticratelimit.go
	snap, err := f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// In edge-stack we should only ever have 1 RateLimitService.
	assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
	// The snapshot should only have the synthetic RateLimitService and not the one defined above.
	assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)

	// Check for a cluster name matching the provided RateLimitService
	isRateLimitCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
	// 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isRateLimitCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a cluster for the
	// RateLimitService that was defined.
	assert.NotNil(t, envoyConfig)
}

// Tests the synthetic rateLimit generation by first creating an invalid RateLimitService and confirming that
// the synthetic RateLimitService gets injected.  Afterwards, a valid RateLimitService is applied and we
// expect the synthetic RateLimitService to be removed in favor of the new valid RateLimitService.
func TestSyntheticRateLimitInjectAndRemove(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.AutoFlush(true)

	// This will cause a synthethic RateLimitService to be injected.
	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  protocol_version: "vBogus"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the synthetic RateLimitService.
	// The user-provided RateLimitService is invalid and so it should be ignored and instead we
	// inject the synthetic RateLimitService.
	snap, err := f.GetSnapshot(HasRateLimitService("default", "synthetic_edge_stack_rate_limit"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// We should only have 1 RateLimitService.
	assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
	// The snapshot should only have the synthetic RateLimitService and not the one defined above.
	assert.Equal(t, "synthetic_edge_stack_rate_limit", snap.Kubernetes.RateLimitServices[0].Name)

	// Check for a cluster name matching the provided RateLimitService
	isRateLimitCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
	// 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isRateLimitCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a cluster for the
	// RateLimitService that was defined.
	assert.NotNil(t, envoyConfig)

	// Updating the yaml for that RateLimitService to include `protocol_version: v3` should make it
	// valid and then remove our synthetic RateLimitService and allow the now valid RateLimitService to be
	// used.
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  protocol_version: "v3"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the RateLimitService defined
	// above.  The RateLimitService has `protocol_version: v3` so it should not be removed/replaced
	// by the synthetic RateLimitService injected by syntheticratelimit.go
	snap, err = f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// In edge-stack we should only ever have 1 RateLimitService.
	assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
	assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)

	// Check for a cluster name matching the provided RateLimitService
	isRateLimitCluster = func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
	// 127.0.0.1:8500
	envoyConfig, err = f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isRateLimitCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a cluster for the
	// RateLimitService that was defined.
	assert.NotNil(t, envoyConfig)
}

// This RateLimitService points at 127.0.0.1:8500, but it does not have `protocol_version: v3`.  It also
// has additional fields set.  The correct action is to edit the RateLimitService to say
// `protocol_version: v3`.
func TestSyntheticRateLimitCopyFields(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: edge-stack-ratelimit-test
  namespace: foo
spec:
  service: 127.0.0.1:8500
  timeout_ms: 12345
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the RateLimitService.
	snap, err := f.GetSnapshot(HasRateLimitService("foo", "edge-stack-ratelimit-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// In edge-stack we should only ever have 1 RateLimitService.
	assert.Equal(t, 1, len(snap.Kubernetes.RateLimitServices))
	// It should be that user-provided RateLimitService...
	assert.Equal(t, "edge-stack-ratelimit-test", snap.Kubernetes.RateLimitServices[0].Name)
	assert.Equal(t, int64(12345), snap.Kubernetes.RateLimitServices[0].Spec.Timeout.Duration.Milliseconds())
	// ... but with `protocol_version: v3` set.
	assert.Equal(t, "v3", snap.Kubernetes.RateLimitServices[0].Spec.ProtocolVersion)

	// Check for a cluster name matching the provided RateLimitService
	isRateLimitCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack rateLimit cluster on
	// 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isRateLimitCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a cluster for the
	// RateLimitService that was defined.
	assert.NotNil(t, envoyConfig)
}
