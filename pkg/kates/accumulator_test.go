package kates

import (
	"context"
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
	ctx := context.Background()

	// Create a set of 10 configmaps to give us some resources to watch.
	cli := testClient(t)
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
	}

	// Use a separate client for watching so we can bypass any caching.
	cli2 := testClient(t)
	// Configure this to slow down dispatch add events. This will dramatically increase the chance
	// of the edge case we are trying to test.
	cli2.watchAdded = func(old *Unstructured, new *Unstructured) {
		time.Sleep(1 * time.Second)
	}
	acc := cli2.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "test=test-bootstrap"})

	snap := &Snap{}
	for {
		<-acc.Changed()
		if acc.Update(snap) {
			break
		}
	}

	// When we are here the first notification will have happened, and since there were 10
	// ConfigMaps prior to starting te Watch, all 10 of those ConfigMaps should be present in the
	// first update.
	assert.Equal(t, 10, len(snap.ConfigMaps))
}

// Make sure we still notify on bootstrap if there are no resources that satisfy a Watch.
func TestBootstrapNotifyEvenOnEmptyWatch(t *testing.T) {
	ctx := context.Background()
	cli := testClient(t)

	// Create a watch with a nonexistent label filter to gaurantee no resources will satisfy the watch.
	acc := cli.Watch(ctx, Query{Name: "ConfigMaps", Kind: "ConfigMap", LabelSelector: "nonexistent-label"})

	snap := &Snap{}
	for {
		<-acc.Changed()
		if acc.Update(snap) {
			break
		}
	}

	// When we are here the first notification will have happened, and since there were no resources
	// that satisfy the selector, the ConfigMaps field should be empty.
	assert.Equal(t, 0, len(snap.ConfigMaps))
}
