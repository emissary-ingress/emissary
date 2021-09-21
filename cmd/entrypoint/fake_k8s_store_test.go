package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	store := entrypoint.NewK8sStore()

	// This cursor, the early cursor starts before we load the file, so we should naturally see
	// deltas for all the objects in the file.
	cEarly := store.Cursor()
	// The file has two mappings, one named foo, and one named bar.
	store.UpsertFile("testdata/TestStore.yaml")

	// This cursor, the late cursor starts after we load the file so we should see the synthetically
	// generated deltas to catch it up, and then the actual deltas.
	cLate := store.Cursor()

	// Make sure we see the natural deltas.
	resourcesEarly, deltasEarly := cEarly.Get()
	assert.Equal(t, 2, len(resourcesEarly))
	require.Equal(t, 2, len(deltasEarly))
	assert.Equal(t, kates.ObjectAdd, deltasEarly[0].DeltaType)
	assert.Equal(t, "bar", deltasEarly[0].Name)
	assert.Equal(t, kates.ObjectAdd, deltasEarly[1].DeltaType)
	assert.Equal(t, "foo", deltasEarly[1].Name)

	// Make sure we see the synthetic deltas.
	resourcesLate, deltasLate := cLate.Get()
	assert.Equal(t, 2, len(resourcesLate))
	require.Equal(t, 2, len(deltasLate))
	assert.Equal(t, kates.ObjectAdd, deltasLate[0].DeltaType)
	assert.Equal(t, "bar", deltasLate[0].Name)
	assert.Equal(t, kates.ObjectAdd, deltasLate[1].DeltaType)
	assert.Equal(t, "foo", deltasLate[1].Name)

	// Now let's update an object.
	fooKey := entrypoint.K8sKey{"Mapping", "default", "foo"}
	foo := resourcesEarly[fooKey]
	require.NotNil(t, foo)
	store.Upsert(foo)

	// Make sure we see the update for the original cursor.
	resourcesEarly, deltasEarly = cEarly.Get()
	assert.Equal(t, 2, len(resourcesEarly))
	require.Equal(t, 1, len(deltasEarly))
	assert.Equal(t, kates.ObjectUpdate, deltasEarly[0].DeltaType)
	assert.Equal(t, "foo", deltasEarly[0].Name)

	// Make sure we see the update for the late cursor.
	resourcesLate, deltasLate = cLate.Get()
	assert.Equal(t, 2, len(resourcesLate))
	require.Equal(t, 1, len(deltasLate))
	assert.Equal(t, kates.ObjectUpdate, deltasLate[0].DeltaType)
	assert.Equal(t, "foo", deltasLate[0].Name)

	// Now let's delete an object.
	store.Delete("Mapping", "default", "foo")

	// Observe the delete from the early cursor.
	resourcesEarly, deltasEarly = cEarly.Get()
	assert.Equal(t, 1, len(resourcesEarly))
	require.Equal(t, 1, len(deltasEarly))
	assert.Equal(t, kates.ObjectDelete, deltasEarly[0].DeltaType)
	assert.Equal(t, "foo", deltasEarly[0].Name)

	// Observe the delete from the late cursor.
	resourcesLate, deltasLate = cLate.Get()
	assert.Equal(t, 1, len(resourcesLate))
	require.Equal(t, 1, len(deltasLate))
	assert.Equal(t, kates.ObjectDelete, deltasLate[0].DeltaType)
	assert.Equal(t, "foo", deltasLate[0].Name)

	// Now that we have had a whole bunch of deltas, lets create another cursor and make sure we
	// don't get the full history, just the synthetic Add deltas.
	c := store.Cursor()
	resources, deltas := c.Get()
	assert.Equal(t, 1, len(resources))
	require.Equal(t, 1, len(deltas))
	assert.Equal(t, kates.ObjectAdd, deltas[0].DeltaType)
	assert.Equal(t, "bar", deltas[0].Name)

	// Now lets add back foo and check all three cursors.
	store.Upsert(foo)
	resources, deltas = c.Get()
	assert.Equal(t, 2, len(resources))
	require.Equal(t, 1, len(deltas))
	assert.Equal(t, kates.ObjectAdd, deltas[0].DeltaType)
	assert.Equal(t, "foo", deltas[0].Name)

	// Observe the add back from the early cursor.
	resourcesEarly, deltasEarly = cEarly.Get()
	assert.Equal(t, 2, len(resourcesEarly))
	require.Equal(t, 1, len(deltasEarly))
	assert.Equal(t, kates.ObjectAdd, deltasEarly[0].DeltaType)
	assert.Equal(t, "foo", deltasEarly[0].Name)

	// Observe the add back from the late cursor.
	resourcesLate, deltasLate = cLate.Get()
	assert.Equal(t, 2, len(resourcesLate))
	require.Equal(t, 1, len(deltasLate))
	assert.Equal(t, kates.ObjectAdd, deltasLate[0].DeltaType)
	assert.Equal(t, "foo", deltasLate[0].Name)
}

func TestNamespaceDefault(t *testing.T) {
	store := entrypoint.NewK8sStore()
	store.UpsertFile("testdata/NamespaceDefault.yaml")
	c := store.Cursor()
	resources, _ := c.Get()
	assert.NotEmpty(t, resources)
	for _, r := range resources {
		assert.Equal(t, "default", r.GetNamespace())
	}
}
