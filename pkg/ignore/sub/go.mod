// This module is a trick to more explicitly tell the Go toolchain
// what the minimum version of certain dependencies are.
//
// The problem starts with the Kubernetes library.  It has a bazillion
// dependencies, and it's kinda picky about those versions.  So, we
// have a `go run github.com/datawire/libk8s/cmd/fix-go.mod` command
// to downgrade all of the shared dependencies in our go.mod to the
// version needed the Kubernetes library version we're using.
//
// That works well, except for when there's a shared dependency that
// we really do want a newer version of.  In that case, `go run
// github.com/datawire/libk8s/cmd/fix-go.mod` would break the build.
//
// The solution is to have a separate go.mod (in a nested Go module)
// where fix-go.mod can't edit it.
//
// Also, be sure to add a package from the dependency module to
// pin.go, so that `go mod tidy` won't remove it from the go.mod.
module github.com/datawire/ambassador/pkg/ignore/sub

go 1.13

// We upgrade from 1.2 to 1.3 for Envoy.
require github.com/gogo/protobuf v1.3.0

// The older version used by Kubernetes doesn't have the Linux
// capabilities stuff we need in cmd/capabilities_wrapper.
require golang.org/x/sys v0.0.0-20191024073052-e66fe6eb8e0c
