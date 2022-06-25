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

// This predicate is used to check k8s snapshots for an AuthService matching the provided name and
// namespace.
func HasAuthService(namespace, name string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.AuthServices {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}
		return false
	}
}

// Tests the synthetic auth generation when a valid AuthService is created.  This AuthService has
// `protocol_version: v3` and should not be replaced by the synthetic AuthService.
func TestSyntheticAuthValid(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v3"
  proto: "grpc"
`)
			assert.NoError(t, err)
			f.Flush()

			// Use the predicate above to check that the snapshot contains the
			// AuthService defined above.  The AuthService has `protocol_version: v3` so
			// it should not be removed/replaced by the synthetic AuthService injected
			// by syntheticauth.go
			snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
			// In edge-stack we should only ever have 1 AuthService.
			assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

			// Check for an ext_authz cluster name matching the provided AuthService
			// (Http_Filters are harder to check since they always have the same name).
			// The namespace for this extauthz cluster should be foo (since that is the
			// namespace of the valid AuthService above).
			isAuthCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_foo")
			}

			// Grab the next Envoy config that has an Edge Stack auth cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isAuthCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a extauth cluster for the
			// AuthService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// This tests with a provided AuthService that has no protocol_version (which defaults to v2).  The
// synthetic AuthService should be created instead.
func TestSyntheticAuthReplace(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  proto: "grpc"
`)
			assert.NoError(t, err)
			f.Flush()

			// Use the predicate above to check that the snapshot contains the
			// AuthService defined above.  The AuthService does not have
			// `protocol_version: v3` so it should be removed and replaced by the
			// synthetic AuthService injected by syntheticauth.go
			snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			// The snapshot should only have the synthetic AuthService and not the one
			// defined above.
			assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
			// In edge-stack we should only ever have 1 AuthService.
			assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

			// Check for an ext_authz cluster name matching the provided AuthService
			// (Http_Filters are harder to check since they always have the same name).
			// The namespace for this extauthz cluster should be default (since that is
			// the namespace of the synthetic AuthService).
			isAuthCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
			}

			// Grab the next Envoy config that has an Edge Stack auth cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isAuthCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a extauth cluster for the
			// AuthService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// Tests the synthetic auth generation when an invalid AuthService is created.  This AuthService has
// `protocol_version: v3` and should not be replaced by the synthetic AuthService even though it has
// a bogus value because the bogus field will be dropped when it is loaded and we will be left with
// a valid AuthService.
func TestSyntheticAuthBogusField(t *testing.T) {
	for _, apiVersion := range []string{"v2", "v3alpha1"} {
		apiVersion := apiVersion // capture loop variable
		t.Run(apiVersion, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")

			f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

			err := f.UpsertYAML(`
---
apiVersion: getambassador.io/` + apiVersion + `
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v3"
  proto: "grpc"
  bogus_field: "foo"
`)
			assert.NoError(t, err)
			f.Flush()

			// Use the predicate above to check that the snapshot contains the
			// AuthService defined above.  The AuthService has `protocol_version: v3` so
			// it should not be removed/replaced by the synthetic AuthService injected
			// by syntheticauth.go
			snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
			assert.NoError(t, err)
			assert.NotNil(t, snap)

			assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
			// In edge-stack we should only ever have 1 AuthService.
			assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

			// Check for an ext_authz cluster name matching the provided AuthService
			// (Http_Filters are harder to check since they always have the same name).
			// The namespace for this extauthz cluster should be foo (since that is the
			// namespace of the valid AuthService above).
			isAuthCluster := func(c *v3cluster.Cluster) bool {
				return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_foo")
			}

			// Grab the next Envoy config that has an Edge Stack auth cluster on
			// 127.0.0.1:8500
			envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
				return FindCluster(envoy, isAuthCluster) != nil
			})
			require.NoError(t, err)

			// Make sure an Envoy Config containing a extauth cluster for the
			// AuthService that was defined.
			assert.NotNil(t, envoyConfig)
		})
	}
}

// Tests the synthetic auth generation when an invalid AuthService (because the protocol_version is
// invalid for the supported enums).  This AuthService should be tossed out an the synthetic
// AuthService should be injected.
func TestSyntheticAuthInvalidProtocolVer(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v4"
  proto: "grpc"
  bogus_field: "foo"
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the synthetic AuthService.
	// The AuthService has `protocol_version: v3`, but it has a bogus field so it should not be
	// validated and instead we inject the synthetic AuthService.
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above.
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the synthetic AuthService.  The namespace
	// for this extauthz cluster should be default (since that is the namespace of the synthetic
	// AuthService).
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was
	// defined.
	assert.NotNil(t, envoyConfig)
}

// Tests the synthetic auth generation when an invalid AuthService is created and edited several
// times in succession.  After the config is edited several times, we should see that the final
// result is our provided valid AuthService.  There should not be any duplicate AuthService
// resources, and the synthetic AuthService that gets created when the first invalid AuthService is
// applied should be removed when the final edit makes it a valid AuthService.
func TestSyntheticAuthChurn(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.AutoFlush(true)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  proto: "grpc"
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v3"
  proto: "grpc"
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  proto: "grpc"
`)
	assert.NoError(t, err)
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v3"
  proto: "grpc"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the AuthService defined
	// above.  The AuthService has `protocol_version: v3` so it should not be removed/replaced
	// by the synthetic AuthService injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above.
	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are
	// harder to check since they always have the same name).  The namespace for this extauthz
	// cluster should be foo (since that is the namespace of the valid AuthService above)
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)
}

// Tests the synthetic auth generation by first creating an invalid AuthService and confirming that
// the synthetic AuthService gets injected.  Afterwards, a valid AuthService is applied and we
// expect the synthetic AuthService to be removed in favor of the new valid AuthService.
func TestSyntheticAuthInjectAndRemove(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.AutoFlush(true)

	// This will cause a synthethic AuthService to be injected.
	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  proto: "grpc"
  bogus_field: "foo"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the synthetic AuthService.
	// The AuthService has `protocol_version: v3`, but it has a bogus field so it should not be
	// validated and instead we inject the synthetic AuthService.
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above.
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// We should only have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the synthetic AuthService.  The namespace
	// for this extauthz cluster should be default (since that is the namespace of the synthetic
	// AuthService).
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was
	// defined.
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")

	// Updating the yaml for that AuthService to include `protocol_version: v3` should make it
	// valid and then remove our synthetic AuthService and allow the now valid AuthService to be
	// used.
	err = f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  protocol_version: "v3"
  proto: "grpc"
`)
	assert.NoError(t, err)

	// Use the predicate above to check that the snapshot contains the AuthService defined
	// above.  The AuthService has `protocol_version: v3` so it should not be removed/replaced
	// by the synthetic AuthService injected by syntheticauth.go
	snap, err = f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are
	// harder to check since they always have the same name).  The namespace for this extauthz
	// cluster should be foo (since that is the namespace of the valid AuthService above).
	isAuthCluster = func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err = f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was
	// defined.
	assert.NotNil(t, envoyConfig)
}

// This AuthService points at 127.0.0.1:8500, but it does not have `protocol_version: v3`.  It also
// has additional fields set.  The correct action is to create a SyntheticAuth copy of this
// AuthService with the same fields but with `protocol_version: v3`.
func TestSyntheticAuthCopyFields(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: 127.0.0.1:8500
  proto: "grpc"
  timeout_ms: 12345

`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the synthetic AuthService.
	// The AuthService has `protocol_version: v3`, but it is missing the `protocol_version: v3`
	// field.  We expect the synthetic AuthService to be injected, but later we will check that
	// the synthetic AuthService has Our custom timeout_ms field.
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Even though it is the synthetic AuthService, we should have the custom timeout_ms and v3
	// protocol version.
	for _, authService := range snap.Kubernetes.AuthServices {
		assert.Equal(t, int64(12345), authService.Spec.Timeout.Duration.Milliseconds())
		assert.Equal(t, "v3", authService.Spec.ProtocolVersion)
	}

	// Check for an ext_authz cluster name matching the synthetic AuthService.  The namespace
	// for this extauthz cluster should be default (since that is the namespace of the synthetic
	// AuthService).
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was
	// defined.
	assert.NotNil(t, envoyConfig)
}

// This AuthService does not point at 127.0.0.1:8500, we leave it alone rather than adding a
// synthetic one.
func TestSyntheticAuthCustomAuthService(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: AuthService
metadata:
  name: edge-stack-auth-test
  namespace: foo
spec:
  auth_service: dummy-service
  proto: "grpc"
  protocol_version: "v3"
`)

	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined
	// above.  The AuthService has `protocol_version: v3` so it should not be removed/replaced
	// by the synthetic AuthService injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	for _, authService := range snap.Kubernetes.AuthServices {
		assert.Equal(t, "dummy-service", authService.Spec.AuthService)
	}

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are
	// harder to check since they always have the same name).  the namespace for this extauthz
	// cluster should be foo (since that is the namespace of the valid AuthService above).
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_dummy_service_foo")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was
	// defined.
	assert.NotNil(t, envoyConfig)
}

// When deciding if we need to inject a synthetic AuthService or not, we need to be able to reliably
// determine if that AuthService points at a localhost:8500 or not.
func TestIsLocalhost8500(t *testing.T) {
	t.Parallel()

	type subtest struct {
		inputAddr string
		expected  bool
	}

	subtests := []subtest{
		{inputAddr: "127.0.0.1:8500", expected: true},
		{inputAddr: "localhost:8500", expected: true},
		{inputAddr: "127.1.2.3:8500", expected: true},
		// IPv6:
		{inputAddr: "http://[0::1]:8500", expected: true},
		{inputAddr: "http://[0:0:0::1]:8500", expected: true},
		{inputAddr: "http://[::0:0:0:1]:8500", expected: true},

		{inputAddr: "127.0.0.1:850", expected: false},
		{inputAddr: "127.0.0.1:8080", expected: false},
		{inputAddr: "192.168.2.10:8500", expected: false},
		{inputAddr: "", expected: false},
		// IPv6:
		{inputAddr: "http://[0::1]:8400", expected: false},
		{inputAddr: "http://[0::2]:8500", expected: false},
		{inputAddr: "http://[0::2]:8080", expected: false},
		{inputAddr: "http://[0:0:0::2]:8500", expected: false},
		{inputAddr: "http://[0:0:0::1]:8080", expected: false},
		{inputAddr: "http://[::0:0:0:2]:8500", expected: false},
		{inputAddr: "http://[::0:0:0:2]:8080", expected: false},
	}

	for _, subtest := range subtests {
		subtest := subtest // capture loop variable
		t.Run(subtest.inputAddr, func(t *testing.T) {
			t.Parallel()
			res := entrypoint.IsLocalhost8500(subtest.inputAddr)
			assert.Equal(t, subtest.expected, res)
		})
	}
}
