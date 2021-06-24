package agent_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/datawire/ambassador/v2/pkg/agent"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func newGenericCallback(apiVersion, kind, name string, eventType agent.CallbackEventType) *agent.GenericCallback {
	obj := &unstructured.Unstructured{}
	id := uuid.New().String()
	obj.SetUID(types.UID(id))

	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace("default")
	return &agent.GenericCallback{
		EventType: eventType,
		Obj:       obj,
		Sotw:      []interface{}{},
	}
}

func TestRolloutStore(t *testing.T) {
	t.Run("will populate the rolloutstore successfully", func(t *testing.T) {
		// given
		t.Parallel()
		rs := agent.NewRolloutStore()
		wg := sync.WaitGroup{}
		wg.Add(10)

		// when
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer wg.Done()
				name := fmt.Sprintf("Rollout%d", i)
				callback := newGenericCallback("argoproj.io/v1alpha1", "Rollout", name, agent.CallbackEventAdded)
				rs.FromCallback(callback)
			}(i)
		}
		wg.Wait()

		// then
		assert.Equal(t, 10, len(rs.Deltas()))
		assert.Equal(t, "Rollout", rs.Deltas()[0].Kind)
		assert.Equal(t, "argoproj.io/v1alpha1", rs.Deltas()[0].APIVersion)
		assert.Equal(t, "default", rs.Deltas()[0].Namespace)
		assert.Equal(t, kates.ObjectAdd, rs.Deltas()[0].DeltaType)
		sotw := rs.StateOfWorld()
		assert.Equal(t, 10, len(sotw))
	})
}
func TestApplicationStore(t *testing.T) {
	t.Run("will populate the rolloutstore successfully", func(t *testing.T) {
		// given
		t.Parallel()
		as := agent.NewApplicationStore()
		wg := sync.WaitGroup{}
		wg.Add(10)

		// when
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer wg.Done()
				name := fmt.Sprintf("Application%d", i)
				callback := newGenericCallback("argoproj.io/v1alpha1", "Application", name, agent.CallbackEventUpdated)
				as.FromCallback(callback)
			}(i)
		}
		wg.Wait()

		// then
		assert.Equal(t, 10, len(as.Deltas()))
		assert.Equal(t, "Application", as.Deltas()[0].Kind)
		assert.Equal(t, "argoproj.io/v1alpha1", as.Deltas()[0].APIVersion)
		assert.Equal(t, "default", as.Deltas()[0].Namespace)
		assert.Equal(t, kates.ObjectUpdate, as.Deltas()[0].DeltaType)
		sotw := as.StateOfWorld()
		assert.Equal(t, 10, len(sotw))
	})

}
