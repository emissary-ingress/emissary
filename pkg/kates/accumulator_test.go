package kates

import (
	"fmt"
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
