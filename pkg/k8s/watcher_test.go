package k8s_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/pkg/dtest"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/dlog"
)

const (
	delay = 10 * time.Second
)

func fetch(ctx context.Context, w *k8s.Watcher, resource, qname string) (result k8s.Resource, err error) {
	go func() {
		time.Sleep(delay)
		w.Stop()
	}()

	err = w.WatchQuery(k8s.Query{Kind: resource, Namespace: k8s.NamespaceAll}, func(w *k8s.Watcher) error {
		list, err := w.List(resource)
		if err != nil {
			return err
		}
		for _, r := range list {
			if r.QName() == qname {
				result = r
				w.Stop()
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := w.Wait(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func info(ctx context.Context) *k8s.KubeInfo {
	return k8s.NewKubeInfo(dtest.Kubeconfig(ctx), "", "")
}

func TestUpdateStatus(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w, err := k8s.NewWatcher(info(ctx))
	require.NoError(t, err)

	svc, err := fetch(ctx, w, "services", "kubernetes.default")
	require.NoError(t, err)
	svc.Status()["loadBalancer"].(map[string]interface{})["ingress"] = []map[string]interface{}{{"hostname": "foo", "ip": "1.2.3.4"}}
	result, err := w.UpdateStatus(ctx, svc)
	if err != nil {
		t.Error(err)
		return
	} else {
		t.Logf("updated %s status, result: %v\n", svc.QName(), result.ResourceVersion())
	}

	w2, err := k8s.NewWatcher(info(ctx))
	require.NoError(t, err)
	svc, err = fetch(ctx, w2, "services", "kubernetes.default")
	require.NoError(t, err)
	ingresses := svc.Status()["loadBalancer"].(map[string]interface{})["ingress"].([]interface{})
	ingress := ingresses[0].(map[string]interface{})
	if ingress["hostname"] != "foo" {
		t.Error("expected foo")
	}

	if ingress["ip"] != "1.2.3.4" {
		t.Error("expected 1.2.3.4")
	}
}

func TestWatchCustom(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w, err := k8s.NewWatcher(info(ctx))
	require.NoError(t, err)

	// XXX: we can only watch custom resources... k8s doesn't
	// support status for CRDs until 1.12
	xmas, err := fetch(ctx, w, "customs", "xmas.default")
	require.NoError(t, err)
	if xmas == nil {
		t.Error("couldn't find xmas")
	} else {
		spec := xmas.Spec()
		if spec["deck"] != "the halls" {
			t.Errorf("expected the halls, got %v", spec["deck"])
		}
	}
}

func TestWatchCustomCollision(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w, err := k8s.NewWatcher(info(ctx))
	require.NoError(t, err)

	easter, err := fetch(ctx, w, "csrv", "easter.default")
	require.NoError(t, err)
	require.NotNil(t, easter)
	t.Logf("easter: %#v", easter)
	require.Equal(t, "the lawn", easter.Spec()["deck"])
}

func TestWatchQuery(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w, err := k8s.NewWatcher(info(ctx))
	require.NoError(t, err)

	services := []string{}
	err = w.WatchQuery(k8s.Query{
		Kind:          "services",
		Namespace:     k8s.NamespaceAll,
		FieldSelector: "metadata.name=kubernetes",
	}, func(w *k8s.Watcher) error {
		list, err := w.List("services")
		if err != nil {
			return err
		}
		for _, r := range list {
			services = append(services, r.QName())
		}
		return nil
	})
	require.NoError(t, err)
	time.AfterFunc(1*time.Second, func() {
		w.Stop()
	})
	require.NoError(t, w.Wait(ctx))
	require.Equal(t, services, []string{"kubernetes.default"})
}
