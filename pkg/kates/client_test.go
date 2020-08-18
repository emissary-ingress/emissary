package kates

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCRUD(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-crud-configmap",
		},
	}

	assert.Equal(t, cm.GetResourceVersion(), "")

	err = cli.Get(ctx, cm, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	created := &ConfigMap{}
	err = cli.Create(ctx, cm, created)
	assert.NoError(t, err)
	assert.NotEqual(t, created.GetResourceVersion(), "")

	created.Labels = map[string]string{"foo": "bar"}
	updated := &ConfigMap{}
	err = cli.Update(ctx, created, updated)
	assert.NoError(t, err)

	gotten := &ConfigMap{}
	err = cli.Get(ctx, cm, gotten)
	assert.NoError(t, err)
	assert.Equal(t, gotten.GetName(), cm.GetName())
	assert.Equal(t, gotten.Labels["foo"], "bar")

	err = cli.Delete(ctx, cm, nil)
	assert.NoError(t, err)

	err = cli.Get(ctx, cm, nil)
	assert.Error(t, err, "expecting not found error")
	assert.True(t, IsNotFound(err), "expecting not found error")
}

func TestUpsert(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-upsert-configmap",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	defer func() {
		cli.Delete(ctx, cm, nil)
	}()

	err = cli.Upsert(ctx, cm, cm, cm)
	assert.NoError(t, err)
	assert.NotEqual(t, "", cm.GetResourceVersion())

	src := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-upsert-configmap",
			Labels: map[string]string{
				"foo": "baz",
			},
		},
	}

	err = cli.Upsert(ctx, cm, src, cm)
	assert.NoError(t, err)
	assert.Equal(t, "baz", cm.Labels["foo"])
}

func TestPatch(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-patch-configmap",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	err = cli.Create(ctx, cm, cm)
	assert.NoError(t, err)

	defer func() {
		cli.Delete(ctx, cm, nil)
	}()

	err = cli.Patch(ctx, cm, StrategicMergePatchType, []byte(`{"metadata": {"annotations": {"moo": "arf"}}}`), cm)
	assert.NoError(t, err)
	assert.Equal(t, "arf", cm.GetAnnotations()["moo"])
}

func TestList(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	namespaces := make([]*Namespace, 0)

	err = cli.List(ctx, Query{Kind: "namespaces"}, &namespaces)
	assert.NoError(t, err)

	// we know there should be at least the default namespace and
	// the kube-system namespace
	assert.True(t, len(namespaces) > 0)

	found := false
	for _, ns := range namespaces {
		if ns.GetName() == "default" {
			found = true
			break
		}
	}

	assert.True(t, found)
}

func TestListSelector(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	myns := &Namespace{
		TypeMeta: TypeMeta{
			Kind: "namespace",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-list-selector-namespace",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	err = cli.Create(ctx, myns, myns)
	assert.NoError(t, err)

	namespaces := make([]*Namespace, 0)

	err = cli.List(ctx, Query{Kind: "namespaces", LabelSelector: "foo=bar"}, &namespaces)
	assert.NoError(t, err)

	assert.Equal(t, len(namespaces), 1)

	if len(namespaces) == 1 {
		assert.Equal(t, namespaces[0].GetName(), myns.GetName())
	}

	err = cli.Delete(ctx, myns, myns)
	assert.NoError(t, err)
}

func TestShortcut(t *testing.T) {
	ctx := context.TODO()

	cli, err := NewClient(ClientOptions{})
	assert.NoError(t, err)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "cm",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-shortcut-configmap",
		},
	}

	created := &ConfigMap{}
	err = cli.Create(ctx, cm, created)
	assert.NoError(t, err)

	err = cli.Delete(ctx, created, nil)
	assert.NoError(t, err)
}

type TestSnapshot struct {
	ConfigMaps []*ConfigMap
	Secrets    []*Secret
}

// Currently this whole test is probabilistic and somewhat end-to-endy (it requires a kubernetes
// cluster, but makes very few assumptions about it). With a bit of a refactor to the kates
// implementation to allow for more mocks, this could be made into a pure unit test and not be
// probabilistic at all.
func TestCoherence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	cli, err := NewClient(ClientOptions{})
	require.NoError(t, err)

	// This simulates an api server that is very slow at notifying its watch clients of updates to
	// config maps, but notifies of other resources at normal speeds. This can really happen.
	cli.watchUpdated = func(_, obj interface{}) {
		un := obj.(*unstructured.Unstructured)
		if un.GetKind() == "ConfigMap" {
			time.Sleep(5 * time.Second)
		}
	}

	// Our snapshot will include both config maps and secrets. We will watch them from one thread
	// while simultaneously updating them both as fast as we can from another thread. While doing
	// this we will make assertions that the watching thread always sees the state as last updated
	// by the updating thread.
	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name:   "test-coherence",
			Labels: map[string]string{},
		},
	}

	// By updating a secret as well as a configmap, we force the accumulator to frequently report
	// that changes have occurred (since watches for secrets are not artificially slowed down),
	// thereby give the watch thread the opportunity see stale configmaps.
	secret := &Secret{
		TypeMeta: TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: ObjectMeta{
			Name:   "test-coherence",
			Labels: map[string]string{},
		},
	}

	defer func() {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err := cli.Delete(ctx, cm, nil)
		if err != nil {
			t.Log(err)
		}
		err = cli.Delete(ctx, secret, nil)
		if err != nil {
			t.Log(err)
		}
	}()

	err = cli.Get(ctx, cm, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	err = cli.Get(ctx, secret, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	acc := cli.Watch(ctx,
		Query{Name: "ConfigMaps", Kind: "ConfigMap"},
		Query{Name: "Secrets", Kind: "Secret"})
	snap := &TestSnapshot{}

	COUNT := 25

	// The mutex protects access to the shared counters lastSentByUpsert and lastSeenByWatch, as
	// well as allowing us to synchronize cli.Upsert() and acc.Update() invocations. The kates API
	// does not require those invocations to be synchronized, however the design of this test does
	// require that.
	mutex := &sync.Mutex{}
	lastSentByUpsert := 0
	lastSeenByWatch := 0

	done := make(chan struct{})
	go func() {
		defer cancel()
		defer close(done)

		for {
			select {
			case <-acc.Changed():
				mutex.Lock()
				if !acc.Update(snap) {
					mutex.Unlock()
					continue
				}
			case <-ctx.Done():
				return
			}

			func() {
				defer mutex.Unlock()

				var cmFromWatch *ConfigMap
				for _, c := range snap.ConfigMaps {
					if c.GetName() == "test-coherence" {
						cmFromWatch = c
						break
					}
				}

				if lastSentByUpsert > 0 {
					assert.NotNil(t, cmFromWatch)
					if cmFromWatch != nil {
						lbl := cmFromWatch.GetLabels()["counter"]
						parts := strings.Split(lbl, "-")
						require.Equal(t, 2, len(parts))
						i, err := strconv.Atoi(parts[1])
						require.NoError(t, err)
						lastSeenByWatch = i
						// This assertion is the core of this test. Despite the design of the test
						// artificially delaying the updates for all configmaps while
						// simultaneiously updating secrets to provide a very high probability the
						// configmaps returned by the watch are stale, we will still always have an
						// up-to-date view of the configmap that we have modified.
						assert.Equal(t, lastSentByUpsert, lastSeenByWatch)
					}
				}

				if lastSeenByWatch == COUNT {
					cancel()
				}
			}()
		}
	}()

	// Increment the counter label of the secret and configmap as quickly as we can.
	for counter := 0; counter <= COUNT; counter += 1 {
		mutex.Lock()
		func() {
			defer mutex.Unlock()
			lbl := fmt.Sprintf("upsert-%d", counter)
			t.Log(lbl)

			labels := cm.GetLabels()
			labels["counter"] = lbl
			cm.SetLabels(labels)

			err := cli.Upsert(ctx, cm, cm, nil)
			require.NoError(t, err)

			labels = secret.GetLabels()
			labels["counter"] = lbl
			secret.SetLabels(labels)
			err = cli.Upsert(ctx, secret, secret, nil)
			require.NoError(t, err)

			lastSentByUpsert = counter
		}()
	}

	<-done
}
