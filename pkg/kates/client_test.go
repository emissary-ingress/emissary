package kates

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	"github.com/datawire/dlib/dlog"
	dtest_k3s "github.com/datawire/dtest"
)

func testClient(t *testing.T, ctx context.Context) (context.Context, *Client) {
	if ctx == nil {
		ctx = dlog.NewTestContext(t, false)
	}
	cli, err := NewClient(ClientConfig{Kubeconfig: dtest_k3s.KubeVersionConfig(ctx, dtest_k3s.Kube22)})
	require.NoError(t, err)
	return ctx, cli
}

func TestCRUD(t *testing.T) {
	ctx, cli := testClient(t, nil)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-crud-configmap",
		},
	}

	assert.Equal(t, cm.GetResourceVersion(), "")

	err := cli.Get(ctx, cm, nil)
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
	ctx, cli := testClient(t, nil)

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
		assert.NoError(t, cli.Delete(ctx, cm, nil))
	}()

	err := cli.Upsert(ctx, cm, cm, cm)
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
	ctx, cli := testClient(t, nil)

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

	err := cli.Create(ctx, cm, cm)
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, cli.Delete(ctx, cm, nil))
	}()

	err = cli.Patch(ctx, cm, StrategicMergePatchType, []byte(`{"metadata": {"annotations": {"moo": "arf"}}}`), cm)
	assert.NoError(t, err)
	assert.Equal(t, "arf", cm.GetAnnotations()["moo"])
}

func TestList(t *testing.T) {
	ctx, cli := testClient(t, nil)

	namespaces := make([]*Namespace, 0)

	err := cli.List(ctx, Query{Kind: "namespaces"}, &namespaces)
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
	ctx, cli := testClient(t, nil)

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

	err := cli.Create(ctx, myns, myns)
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
	ctx, cli := testClient(t, nil)

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "cm",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-shortcut-configmap",
		},
	}

	created := &ConfigMap{}
	err := cli.Create(ctx, cm, created)
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
	ctx, cli := testClient(t, nil)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// This simulates an api server that is very slow at notifying its watch clients of updates to
	// config maps, but notifies of other resources at normal speeds. This can really happen.
	cli.watchUpdated = func(_, obj *Unstructured) {
		if obj.GetKind() == "ConfigMap" {
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
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err := cli.Delete(ctx, cm, nil)
		if err != nil {
			t.Log(err)
		}
		err = cli.Delete(ctx, secret, nil)
		if err != nil {
			t.Log(err)
		}
	}()

	err := cli.Get(ctx, cm, nil)
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

	acc, err := cli.Watch(ctx,
		Query{Name: "ConfigMaps", Kind: "ConfigMap"},
		Query{Name: "Secrets", Kind: "Secret"})
	require.NoError(t, err)
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
			var deltas []*Delta
			select {
			case <-acc.Changed():
				mutex.Lock()
				updated, err := acc.UpdateWithDeltas(ctx, snap, &deltas)
				assert.NoError(t, err)
				if !updated {
					mutex.Unlock()
					continue
				}
			case <-ctx.Done():
				return
			}

			for _, delta := range deltas {
				bytes, err := json.Marshal(delta)
				assert.NoError(t, err)
				t.Log(string(bytes))
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

func TestDeltas(t *testing.T) {
	doDeltaTest(t, 0, func(_, _ *Unstructured) {})
}

func TestDeltasWithLocalDelay(t *testing.T) {
	doDeltaTest(t, 3*time.Second, func(_, _ *Unstructured) {})
}

func TestDeltasWithRemoteDelay(t *testing.T) {
	doDeltaTest(t, 0, func(old, new *Unstructured) {
		// This will slow down updates to just the resources we are paying attention to in this test.
		obj := new
		if obj == nil {
			obj = old
		}

		if strings.HasPrefix(obj.GetName(), "test-deltas") {
			time.Sleep(3 * time.Second)
		}
	})
}

func doDeltaTest(t *testing.T, localDelay time.Duration, watchHook func(*Unstructured, *Unstructured)) {
	_ctx, cli := testClient(t, nil)
	var (
		_cm1 = &ConfigMap{
			TypeMeta: TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: ObjectMeta{
				Name:   "test-deltas-1",
				Labels: map[string]string{},
			},
		}
		_cm2 = &ConfigMap{
			TypeMeta: TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: ObjectMeta{
				Name:   "test-deltas-2",
				Labels: map[string]string{},
			},
		}
	)
	t.Cleanup(func() {
		if err := cli.Delete(_ctx, _cm1, nil); err != nil && !IsNotFound(err) {
			t.Error(err)
		}
		if err := cli.Delete(_ctx, _cm2, nil); err != nil && !IsNotFound(err) {
			t.Error(err)
		}
	})

	ctx, cancel := context.WithTimeout(_ctx, 30*time.Second)
	defer cancel()

	cli.watchAdded = watchHook
	cli.watchUpdated = watchHook
	cli.watchDeleted = watchHook

	cm1, cm2 := _cm1, _cm2

	defer func() {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if cm1 != nil {
			err := cli.Delete(ctx, cm1, nil)
			if err != nil {
				t.Log(err)
			}
		}
		err := cli.Delete(ctx, cm2, nil)
		if err != nil {
			t.Log(err)
		}
	}()

	err := cli.Get(ctx, cm1, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	err = cli.Get(ctx, cm2, nil)
	assert.Error(t, err, "expecting not found error")
	if !IsNotFound(err) {
		t.Error(err)
		return
	}

	acc, err := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap"})
	require.NoError(t, err)
	snap := &TestSnapshot{}

	err = cli.Upsert(ctx, cm1, cm1, nil)
	require.NoError(t, err)
	err = cli.Upsert(ctx, cm2, cm2, nil)
	require.NoError(t, err)

	time.Sleep(localDelay)

	for {
		<-acc.Changed()
		var deltas []*Delta
		updated, err := acc.UpdateWithDeltas(ctx, snap, &deltas)
		require.NoError(t, err)
		if !updated {
			continue
		}

		checkForDelta(t, ObjectAdd, "test-deltas-1", deltas)
		checkForDelta(t, ObjectAdd, "test-deltas-2", deltas)
		break
	}

	cm1.SetLabels(map[string]string{"foo": "bar"})
	err = cli.Upsert(ctx, cm1, cm1, nil)
	require.NoError(t, err)

	for {
		<-acc.Changed()
		var deltas []*Delta
		updated, err := acc.UpdateWithDeltas(ctx, snap, &deltas)
		require.NoError(t, err)
		if !updated {
			continue
		}

		checkForDelta(t, ObjectUpdate, "test-deltas-1", deltas)
		checkNoDelta(t, "test-deltas-2", deltas)
		break
	}

	err = cli.Delete(ctx, cm1, nil)
	require.NoError(t, err)
	cm1 = nil

	time.Sleep(localDelay)

	for {
		<-acc.Changed()
		var deltas []*Delta
		updated, err := acc.UpdateWithDeltas(ctx, snap, &deltas)
		require.NoError(t, err)
		if !updated {
			continue
		}

		checkForDelta(t, ObjectDelete, "test-deltas-1", deltas)
		checkNoDelta(t, "test-deltas-2", deltas)
		break
	}

	cancel()
}

func checkForDelta(t *testing.T, dt DeltaType, name string, deltas []*Delta) {
	for _, delta := range deltas {
		if delta.DeltaType == dt && delta.GetName() == name {
			return
		}
	}

	assert.Fail(t, fmt.Sprintf("could not find delta %d %s", dt, name))
}

func checkNoDelta(t *testing.T, name string, deltas []*Delta) {
	for _, delta := range deltas {
		if delta.GetName() == name {
			assert.Fail(t, fmt.Sprintf("found delta %s: %d", name, delta.DeltaType))
			return
		}
	}
}

// This is a unit test for the patchWatch method of client. When you are watching resources and also
// modifying the same set that you are watching (as is the case with a read/write controller), the
// client has two sources of information for any given resource: (1) the version of the resource
// reported by the watch, and (2) the version of the resource returned whenever a
// Create/Update/Delete is performed. The patchWatch method updates the results of a watch to ensure
// we always report back the newest version for any given resource.
func TestPatchWatch(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	ctx := context.Background()

	cli, err := NewClient(ClientConfig{})
	require.NoError(err)

	// Make a field the same way newAccumulator does.
	field, err := cli.newField(Query{Name: "Pods", Kind: "pods"})
	require.NoError(err)

	// Convenience function for making multiple versions of a given pod.
	makePod := func(namespace, name string, version int) *Unstructured {
		un := &Unstructured{}
		un.SetGroupVersionKind(field.mapping.GroupVersionKind)
		un.SetNamespace(namespace)
		un.SetName(name)
		un.SetUID(types.UID(fmt.Sprintf("UID:%s.%s", namespace, name)))
		un.SetResourceVersion(fmt.Sprintf("%d", version))
		return un
	}

	// The field.values map holds the version of a resource reported by watch.
	//
	// The cli.canonical map stores any resource that we Get/List/Create/Update/Delete.
	//
	// We can exercise all logic in patchWatch by populating these two maps in various permutations
	// as is done below:

	// Make a pod to take through the CRUD cycle.
	p1 := makePod("default", "foo", 1)
	p1Key := unKey(p1)

	p1Newer := makePod("default", "foo", 2)
	require.Equal(p1Key, unKey(p1Newer))

	// Create: something in cli.canonical, nothing in field.values
	cli.canonical[p1Key] = p1
	delete(field.values, p1Key)
	err = cli.patchWatch(ctx, field)
	require.NoError(err)
	assert.Equal(p1, field.values[p1Key])

	// Local Update: something newer in cli.canonical, older version in field.values
	cli.canonical[p1Key] = p1Newer
	field.values[p1Key] = p1
	err = cli.patchWatch(ctx, field)
	require.NoError(err)
	assert.Equal(p1Newer, field.values[p1Key])

	// Remote Update: something older in cli.canonical, something newer in field.values
	cli.canonical[p1Key] = p1
	field.values[p1Key] = p1Newer
	err = cli.patchWatch(ctx, field)
	require.NoError(err)
	assert.Equal(p1Newer, field.values[p1Key])
	assert.NotContains(cli.canonical, p1Key)

	// Delete: nil value in cli.canonical, something in field.values
	cli.canonical[p1Key] = nil
	field.values[p1Key] = p1Newer
	err = cli.patchWatch(ctx, field)
	require.NoError(err)
	assert.NotContains(field.values, p1Key)
}
