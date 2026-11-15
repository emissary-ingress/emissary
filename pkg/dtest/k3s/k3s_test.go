package k3s_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/datawire/dlib/dlog"
	dtest_k3s "github.com/datawire/dtest"

	"github.com/emissary-ingress/emissary/v3/pkg/dtest/k3s"
)

// Every caller must get the same path, and loading that path must keep
// resolving to the cluster while other callers keep asking for it. A path that
// is rewritten in place on each call fails this with loads that resolve to
// client-go's http://localhost:8080 default.
func TestKubeVersionConfigStableUnderConcurrentCallers(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	path := k3s.KubeVersionConfig(ctx, dtest_k3s.Kube22)
	host, err := loadHost(path)
	require.NoError(t, err)
	require.NotEqual(t, "http://localhost:8080", host)

	stop := make(chan struct{})
	var callers sync.WaitGroup
	for i := 0; i < 8; i++ {
		callers.Add(1)
		go func() {
			defer callers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if got := k3s.KubeVersionConfig(ctx, dtest_k3s.Kube22); got != path {
					t.Errorf("KubeVersionConfig returned %q, want %q", got, path)
					return
				}
			}
		}()
	}

	loads, bad := 0, 0
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		got, err := loadHost(path)
		loads++
		if err != nil || got != host {
			bad++
		}
	}
	close(stop)
	callers.Wait()

	t.Logf("%d loads of %s, %d did not resolve to %s", loads, path, bad, host)
	require.Zero(t, bad)
}

func loadHost(path string) (string, error) {
	flags := genericclioptions.NewConfigFlags(false)
	flags.KubeConfig = &path
	cfg, err := flags.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return "", err
	}
	return cfg.Host, nil
}
