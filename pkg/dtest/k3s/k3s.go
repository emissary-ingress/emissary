// Package k3s hands tests a kubeconfig for the dtest k3s cluster that is safe
// to load while other tests, in this process or in other test binaries, are
// using the same cluster.
package k3s

import (
	"context"
	"os"
	"sync"

	dtest_k3s "github.com/datawire/dtest"
)

var (
	once sync.Once
	path string
)

// KubeVersionConfig brings up the dtest k3s cluster on first use and returns
// the path of a kubeconfig that only this process writes, exactly once.
//
// dtest rewrites its own kubeconfig path in place on every call, from every
// process that uses the cluster, so a client loading that path concurrently can
// read a truncated file, which client-go silently resolves to
// http://localhost:8080. Only the first call's version and flags matter; dtest
// reuses a running cluster regardless.
//
// An explicit DTEST_KUBECONFIG is the caller's own file and is returned as is.
func KubeVersionConfig(ctx context.Context, ver dtest_k3s.KubeVersion, k3sExtraFlags ...string) string {
	once.Do(func() {
		path = dtest_k3s.KubeVersionConfig(ctx, ver, k3sExtraFlags...)
		if os.Getenv("DTEST_KUBECONFIG") != "" {
			return
		}
		path = writePrivate(dtest_k3s.GetKubeconfig(ctx))
	})
	return path
}

func writePrivate(kubeconfig string) string {
	file, err := os.CreateTemp("", "dtest-kubeconfig-*.yaml")
	if err != nil {
		panic(err)
	}
	if _, err := file.WriteString(kubeconfig); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	return file.Name()
}
