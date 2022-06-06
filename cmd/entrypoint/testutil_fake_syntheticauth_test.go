package entrypoint_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

// This predicate is used to check k8s snapshots for an AuthService matching the provided name and namespace
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

// Tests the synthetic auth generation when a valid AuthService is created
// This authservice has protocol_Version: v3 and should not be replaced by the synthetic AuthService
func TestSyntheticAuthValid(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
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
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
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

	t.Setenv("EDGE_STACK", "")
}

// Tests the synthetic auth generation when a valid AuthService is created as a getambassador.io/v2 resource
// This authservice has protocol_Version: v3 and should not be replaced by the synthetic AuthService
func TestSyntheticAuthValidV2(t *testing.T) {
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
  protocol_version: "v3"
  proto: "grpc"
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
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

	t.Setenv("EDGE_STACK", "")
}

// This tests with a provided AuthService that has no protocol_version (which defaults to v2)
// The synthetic AuthService should be created instead
func TestSyntheticAuthReplace(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

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
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService does not have protocol_Version: v3 so it should be removed and replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be default (since that is the namespace of the synthetic AuthService)
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")
}

// This tests with a provided AuthService that has no protocol_version (which defaults to v2)
// The synthetic AuthService should be created instead
func TestSyntheticAuthReplaceV2(t *testing.T) {
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
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService does not have protocol_Version: v3 so it should be removed and replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be default (since that is the namespace of the synthetic AuthService)
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")
}

// Tests the synthetic auth generation when an invalid AuthService is created as a getambassador.io/v2 resource
// This authservice has protocol_Version: v3 and should not be replaced by the synthetic AuthService even though it has a bogus value
// because the bogus field will be dropped when it is loaded and we will be left with a Valid AuthService
func TestSyntheticAuthBogusField(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	err := f.UpsertYAML(`
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
  bogus_field: "foo"
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
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

	t.Setenv("EDGE_STACK", "")
}

// Tests the synthetic auth generation when an invalid AuthService is created as a getambassador.io/v2 resource
// This authservice has protocol_Version: v3 and should be replaced by the synthetic AuthService because it contains a bogus field and is not valid.
func TestSyntheticAuthBogusFieldV2(t *testing.T) {
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
  protocol_version: "v3"
  proto: "grpc"
  bogus_field: "foo"
`)
	assert.NoError(t, err)
	f.Flush()

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
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

	t.Setenv("EDGE_STACK", "")

}

// Tests the synthetic auth generation when an invalid AuthService (because the protocol_version is invalid for the supported enums)
// This AuthService should be tossed out an the synthetic AuthService should be injected
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

	// Use the predicate above to check that the snapshot contains the Synthetic AuthService
	// The AuthService has protocol_Version: v3, but it has a bogus field so it should not be validated and instead we inject the synthetic authservice
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the synthetic AuthService.
	// the namespace for this extauthz cluster should be default (since that is the namespace of the synthetic AuthService)
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")
}

// Tests the synthetic auth generation when an invalid AuthService is created and edited several times in succession.
// After the config is edited several times, we should see that the final result is our provided valid AuthService.
// There should not be any duplicate AuthService resources, and the synthetic AuthService that gets created when the first
// Invalid AuthService is applied should be removed when the final edit makes it a valid AuthService.
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

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err := f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above
	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
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

	t.Setenv("EDGE_STACK", "")
}

// Tests the synthetic auth generation by first creating an invalid AuthService and confirming that the synthetic AuthService gets injected.
// Afterwards, a valid AuthService is applied and we expect the synthetic AuthService to be removed in favor of the new valid AuthService.
func TestSyntheticAuthInjectAndRemove(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.AutoFlush(true)

	// This will cause a synthethic authservice to be injected
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

	// Use the predicate above to check that the snapshot contains the Synthetic AuthService
	// The AuthService has protocol_Version: v3, but it has a bogus field so it should not be validated and instead we inject the synthetic authservice
	snap, err := f.GetSnapshot(HasAuthService("default", "synthetic-edge-stack-auth"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// The snapshot should only have the synthetic AuthService and not the one defined above
	assert.Equal(t, "synthetic-edge-stack-auth", snap.Kubernetes.AuthServices[0].Name)
	// We should only have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the synthetic AuthService.
	// the namespace for this extauthz cluster should be default (since that is the namespace of the synthetic AuthService)
	isAuthCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_default")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")

	// Updating the yaml for that AuthService to include protocol_version: v3 should make it valid and then
	// Remove our synthetic AuthService and allow the now valid AuthService to be used.
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

	// Use the predicate above to check that the snapshot contains the AuthService defined above
	// The AuthService has protocol_Version: v3 so it should not be removed/replaced by the synthetic AuthService
	// injected by syntheticauth.go
	snap, err = f.GetSnapshot(HasAuthService("foo", "edge-stack-auth-test"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	assert.Equal(t, "edge-stack-auth-test", snap.Kubernetes.AuthServices[0].Name)
	// In edge-stack we should only ever have 1 AuthService.
	assert.Equal(t, 1, len(snap.Kubernetes.AuthServices))

	// Check for an ext_authz cluster name matching the provided AuthService (Http_Filters are harder to check since they always have the same name)
	// the namespace for this extauthz cluster should be foo (since that is the namespace of the valid AuthService above)
	isAuthCluster = func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "cluster_extauth_127_0_0_1_8500_foo")
	}

	// Grab the next Envoy config that has an Edge Stack auth cluster on 127.0.0.1:8500
	envoyConfig, err = f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isAuthCluster) != nil
	})
	require.NoError(t, err)

	// Make sure an Envoy Config containing a extauth cluster for the AuthService that was defined
	assert.NotNil(t, envoyConfig)

	t.Setenv("EDGE_STACK", "")

}
