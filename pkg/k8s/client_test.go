package k8s_test

import (
	"context"
	"os"
	"testing"

	"github.com/datawire/ambassador/v2/pkg/dtest"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/dlog"
)

func TestMain(m *testing.M) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	dtest.WithMachineLock(context.TODO(), func(ctx context.Context) {
		dtest.K8sApply(ctx, dtest.Kube22, "00-custom-crd.yaml", "custom.yaml")

		os.Exit(m.Run())
	})
}

func TestList(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	c, err := k8s.NewClient(info(ctx))
	if err != nil {
		t.Error(err)
		return
	}
	svcs, err := c.List(ctx, "svc")
	if err != nil {
		t.Error(err)
	}
	found := false
	for _, svc := range svcs {
		if svc.Name() == "kubernetes" {
			found = true
		}
	}
	if !found {
		t.Errorf("did not find kubernetes service")
	}

	customs, err := c.List(ctx, "customs")
	if err != nil {
		t.Error(err)
	}
	found = false
	for _, cust := range customs {
		if cust.Name() == "xmas" {
			found = true
		}
	}

	if !found {
		t.Errorf("did not find xmas")
	}
}
