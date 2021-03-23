package agent_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/datawire/ambassador/pkg/agent"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFromCallback(t *testing.T) {
	newCallback := func(name string) *agent.GenericCallback {
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion("argoproj.io/v1alpha1")
		obj.SetKind("Rollout")
		obj.SetName(name)
		obj.SetNamespace("default")
		return &agent.GenericCallback{
			EventType: agent.CallbackEventAdded,
			Obj:       obj,
			Sotw:      []interface{}{},
		}
	}
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
				rs.FromCallback(newCallback(name))
			}(i)
		}
		wg.Wait()

		// then
		assert.Equal(t, 10, len(rs.Deltas()))
		assert.Equal(t, "Rollout", rs.Deltas()[0].Kind)
		assert.Equal(t, "argoproj.io/v1alpha1", rs.Deltas()[0].APIVersion)
		assert.Equal(t, "default", rs.Deltas()[0].Namespace)
		assert.Equal(t, "default", rs.Deltas()[0].Namespace)
		assert.Equal(t, kates.ObjectAdd, rs.Deltas()[0].DeltaType)
	})
}
