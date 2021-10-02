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

func fetch(ctx context.Context, w *k8s.Watcher, resource, qname string) (result k8s.Resource) {
	go func() {
		time.Sleep(delay)
		w.Stop()
	}()

	err := w.WatchQuery(k8s.Query{Kind: resource, Namespace: k8s.NamespaceAll}, func(w *k8s.Watcher) {
		for _, r := range w.List(resource) {
			if r.QName() == qname {
				result = r
				w.Stop()
			}
		}
	})

	if err != nil {
		panic(err)
	}

	w.Wait(ctx)
	return result
}

func info(ctx context.Context) *k8s.KubeInfo {
	return k8s.NewKubeInfo(dtest.Kubeconfig(ctx), "", "")
}

func TestUpdateStatus(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w := k8s.MustNewWatcher(info(ctx))

	svc := fetch(ctx, w, "services", "kubernetes.default")
	svc.Status()["loadBalancer"].(map[string]interface{})["ingress"] = []map[string]interface{}{{"hostname": "foo", "ip": "1.2.3.4"}}
	result, err := w.UpdateStatus(ctx, svc)
	if err != nil {
		t.Error(err)
		return
	} else {
		t.Logf("updated %s status, result: %v\n", svc.QName(), result.ResourceVersion())
	}

	svc = fetch(ctx, k8s.MustNewWatcher(info(ctx)), "services", "kubernetes.default")
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
	w := k8s.MustNewWatcher(info(ctx))

	// XXX: we can only watch custom resources... k8s doesn't
	// support status for CRDs until 1.12
	xmas := fetch(ctx, w, "customs", "xmas.default")
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
	w := k8s.MustNewWatcher(info(ctx))

	easter := fetch(ctx, w, "csrv", "easter.default")
	if easter == nil {
		t.Error("couln't find easter")
	} else {
		spec := easter.Spec()
		deck := spec["deck"]
		if deck != "the lawn" {
			t.Errorf("expected the lawn, got %v", deck)
		}
	}
}

func TestWatchQuery(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	w := k8s.MustNewWatcher(info(ctx))

	services := []string{}
	err := w.WatchQuery(k8s.Query{
		Kind:          "services",
		Namespace:     k8s.NamespaceAll,
		FieldSelector: "metadata.name=kubernetes",
	}, func(w *k8s.Watcher) {
		for _, r := range w.List("services") {
			services = append(services, r.QName())
		}
	})
	if err != nil {
		panic(err)
	}
	time.AfterFunc(1*time.Second, func() {
		w.Stop()
	})
	w.Wait(ctx)
	require.Equal(t, services, []string{"kubernetes.default"})
}
