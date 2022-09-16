package kates

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Snap struct {
	ConfigMaps []*ConfigMap
}

// Make sure that we don't prematurely signal the first changed event. The first notification should
// wait until we have done a complete List()ing of all existing resources.
func TestBootstrapNoNotifyBeforeSync(t *testing.T) {
	// Create a set of 10 configmaps to give us some resources to watch.
	ctx, cli := testClient(t, nil)
	var cms [10]*ConfigMap
	for i := 0; i < 10; i++ {
		cm := &ConfigMap{
			TypeMeta: TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: ObjectMeta{
				Name: fmt.Sprintf("test-bootstrap-%d", i),
				Labels: map[string]string{
					"test": "test-bootstrap",
				},
			},
		}
		err := cli.Upsert(ctx, cm, cm, &cm)
		require.NoError(t, err)
		cms[i] = cm
	}

	// Use a separate client for watching so we can bypass any caching.
	_, cli2 := testClient(t, nil)
	// Configure this to slow down dispatch add events. This will dramatically increase the chance
	// of the edge case we are trying to test.
	cli2.watchAdded = func(old *Unstructured, new *Unstructured) {
		time.Sleep(1 * time.Second)
	}
	acc, err := cli2.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "test=test-bootstrap"})
	require.NoError(t, err)

	snap := &Snap{}
	for {
		<-acc.Changed()
		updated, err := acc.Update(ctx, snap)
		require.NoError(t, err)
		if updated {
			break
		}
	}

	// When we are here the first notification will have happened, and since there were 10
	// ConfigMaps prior to starting te Watch, all 10 of those ConfigMaps should be present in the
	// first update.
	assert.Equal(t, 10, len(snap.ConfigMaps))

	t.Cleanup(func() {
		for _, cm := range cms {
			if err := cli.Delete(ctx, cm, nil); err != nil && !IsNotFound(err) {
				t.Error(err)
			}
		}
	})
}

// Make sure we still notify on bootstrap if there are no resources that satisfy a Watch.
func TestBootstrapNotifyEvenOnEmptyWatch(t *testing.T) {
	ctx, cli := testClient(t, nil)

	// Create a watch with a nonexistent label filter to gaurantee no resources will satisfy the watch.
	acc, err := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "nonexistent-label"})
	require.NoError(t, err)

	snap := &Snap{}
	for {
		<-acc.Changed()
		updated, err := acc.Update(ctx, snap)
		require.NoError(t, err)
		if updated {
			break
		}
	}

	// When we are here the first notification will have happened, and since there were no resources
	// that satisfy the selector, the ConfigMaps field should be empty.
	assert.Equal(t, 0, len(snap.ConfigMaps))
}

// Make sure we coalesce raw changes before sending an update when a batch of resources
// are created/modified in quick succession.
func TestBatchChangesBeforeNotify(t *testing.T) {
	ctx, cli := testClient(t, nil)
	// Set a long enough interval to make sure all changes are batched before sending.
	err := cli.MaxAccumulatorInterval(10 * time.Second)
	require.NoError(t, err)
	acc, err := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "test=test-batch"})
	require.NoError(t, err)

	snap := &Snap{}

	// Listen for changes from the Accumulator. Here it will listen for only 2 updates
	// The first update should be the one sent during bootstrap. No resources should have changed
	// in this update. The second update should contain resource changes.
	<-acc.Changed()
	updated, err := acc.Update(ctx, snap)
	require.NoError(t, err)
	if !updated {
		t.Error("Expected snapshot to be successfully updated after receiving first change event")
	}
	assert.Equal(t, 0, len(snap.ConfigMaps))

	// Use a separate client to create resources to avoid any potential uses of the cache
	_, cli2 := testClient(t, nil)

	// Create a set of 10 Configmaps after the Accumulator is watching to simulate getting
	// a bunch of resources at once mid-watch.
	var cms [10]*ConfigMap
	for i := 0; i < 10; i++ {
		cm := &ConfigMap{
			TypeMeta: TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: ObjectMeta{
				Name: fmt.Sprintf("test-batch-%d", i),
				Labels: map[string]string{
					"test": "test-batch",
				},
			},
		}
		err := cli2.Upsert(ctx, cm, cm, &cm)
		require.NoError(t, err)
		cms[i] = cm
	}

	<-acc.Changed()
	updated, err = acc.Update(ctx, snap)
	require.NoError(t, err)
	if !updated {
		t.Error("Expected snapshot to be successfully updated after receiving second change event")
	}

	// After receiving 2 updates from the Accumulator, we should have 10 ConfigMaps
	// in our Snapshot due to the Accumulator coalescing changes before sending an update.
	assert.Equal(t, 10, len(snap.ConfigMaps))

	t.Cleanup(func() {
		for _, cm := range cms {
			if err := cli.Delete(ctx, cm, nil); err != nil && !IsNotFound(err) {
				t.Error(err)
			}
		}
	})
}

// Make sure we send an update after the window period expires when we keep
// sending changes less than the batch interval. This is to test against an edge case where a
// a change event is never triggered due to constant changes.
func TestNotifyNotInfinitelyBlocked(t *testing.T) {
	ctx, cli := testClient(t, nil)
	err := cli.MaxAccumulatorInterval(5 * time.Second)
	require.NoError(t, err)
	acc, err := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "test=test-batch-max"})
	require.NoError(t, err)

	snap := &Snap{}

	<-acc.Changed()
	updated, err := acc.Update(ctx, snap)
	require.NoError(t, err)
	if !updated {
		t.Error("Expected snapshot to be successfully updated after receiving first change event")
	}
	assert.Equal(t, 0, len(snap.ConfigMaps))

	var cms []*ConfigMap
	ctx2, cli2 := testClient(t, nil)
	ctx2, cancel := context.WithCancel(ctx2)
	var wg sync.WaitGroup
	// Create a new Configmap every 2 seconds < 5 second interval to simulate a constant changes
	go func() {
		wg.Add(1)
		defer wg.Done()
		var i int
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				cm := &ConfigMap{
					TypeMeta: TypeMeta{
						Kind: "ConfigMap",
					},
					ObjectMeta: ObjectMeta{
						Name: fmt.Sprintf("test-batch-%d", i),
						Labels: map[string]string{
							"test": "test-batch-max",
						},
					},
				}
				err := cli2.Upsert(ctx, cm, cm, &cm)
				require.NoError(t, err)
				cms = append(cms, cm)
				i++
			case <-ctx2.Done():
				return
			}
		}
	}()

	// Watch for second change. Actually validating this is tricky. Idiosyncratic timing differences
	// can cause the number of Configmaps in the change event to change across test runs resulting in a
	// flakey test. We're just concerned that we got _a_ change when constant updates are being made
	// less than the batch window interval so that we're not infinitely blocked. So we're just going to
	// check that the snapshot is non-empty after we get the change. If we don't
	// get a change after some time then we fail the test.
	select {
	case <-acc.Changed():
		updated, err = acc.Update(ctx, snap)
		require.NoError(t, err)
		if !updated {
			t.Error("Expected snapshot to be successfully updated after receiving second change event")
		}
		assert.Greater(t, len(snap.ConfigMaps), 0)
		cancel()
		wg.Wait()
	case <-time.After(10 * time.Second):
		cancel()
		wg.Wait()
		t.Error("Timeout after 10s listening for second change. It's possible it's infinitely blocked")
	}

	t.Cleanup(func() {
		for _, cm := range cms {
			if err := cli.Delete(ctx, cm, nil); err != nil && !IsNotFound(err) {
				t.Error(err)
			}
		}
	})
}

// Make sure we get single updates when changes are submitted after the batch interval has expired.
func TestNotifyOnUpdate(t *testing.T) {
	ctx, cli := testClient(t, nil)
	err := cli.MaxAccumulatorInterval(2 * time.Second)
	require.NoError(t, err)
	acc, err := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "test=test-isolated"})
	require.NoError(t, err)

	snap := &Snap{}

	waitForChange := func() {
		<-acc.Changed()
		updated, err := acc.Update(ctx, snap)
		require.NoError(t, err)
		if !updated {
			t.Error("Expected snapshot to be successfully updated after receiving change event")
		}
	}

	waitForChange()
	assert.Equal(t, 0, len(snap.ConfigMaps))

	var cms [2]*ConfigMap

	cm := &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-isolated-1",
			Labels: map[string]string{
				"test": "test-isolated",
			},
		},
	}
	err = cli.Upsert(ctx, cm, cm, &cm)
	require.NoError(t, err)
	cms[0] = cm

	waitForChange()
	assert.Equal(t, 1, len(snap.ConfigMaps))

	// Send the next change after the 2 second batch interval
	time.Sleep(3)

	cm = &ConfigMap{
		TypeMeta: TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: ObjectMeta{
			Name: "test-isolated-2",
			Labels: map[string]string{
				"test": "test-isolated",
			},
		},
	}
	err = cli.Upsert(ctx, cm, cm, &cm)
	require.NoError(t, err)
	cms[1] = cm

	waitForChange()
	assert.Equal(t, 2, len(snap.ConfigMaps))

	t.Cleanup(func() {
		for _, cm := range cms {
			if err := cli.Delete(ctx, cm, nil); err != nil && !IsNotFound(err) {
				t.Error(err)
			}
		}
	})
}
