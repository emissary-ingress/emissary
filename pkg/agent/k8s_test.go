package agent_test

import (
	"context"
	"testing"

	"github.com/datawire/ambassador/pkg/agent"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

type informerMock struct {
	eventHandler cache.ResourceEventHandler
	run          func(handler cache.ResourceEventHandler)
}

func (i *informerMock) AddEventHandler(handler cache.ResourceEventHandler) {
	i.eventHandler = handler

}
func (i *informerMock) Run(stopCh <-chan struct{}) {
	i.run(i.eventHandler)
}

func (i *informerMock) ListCache() []interface{} {
	return nil
}

func newInformerMock(run func(handler cache.ResourceEventHandler)) *informerMock {
	return &informerMock{
		run: run,
	}
}

func (i *informerMock) fakeInformer(di dynamic.Interface, ns string, gvr *schema.GroupVersionResource) agent.Informer {
	return i
}

func TestWatchGeneric(t *testing.T) {
	t.Run("will watch generic resource successfully", func(t *testing.T) {
		// given
		t.Parallel()
		runFunc := func(handler cache.ResourceEventHandler) {
			obj := &unstructured.Unstructured{}
			obj.SetName("obj1-added")
			handler.OnAdd(obj)

			objNew := &unstructured.Unstructured{}
			objNew.SetName("obj1-new")
			handler.OnUpdate(obj, objNew)

			objDel := &unstructured.Unstructured{}
			objDel.SetName("obj1-del")
			handler.OnDelete(objDel)
		}
		mock := newInformerMock(runFunc)
		dc := agent.NewDynamicClient(nil, mock.fakeInformer)
		rolloutGvr, _ := schema.ParseResourceArg("rollouts.v1alpha1.argoproj.io")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// when
		rolloutCallback := dc.WatchGeneric(ctx, "default", rolloutGvr)

		// then
		assert.NotNil(t, rolloutCallback)
		for i := 0; i < 3; i++ {
			select {
			case callback := <-rolloutCallback:
				switch callback.EventType {
				case agent.CallbackEventAdded:
					assert.Equal(t, "obj1-added", callback.Obj.GetName())
				case agent.CallbackEventUpdated:
					assert.Equal(t, "obj1-new", callback.Obj.GetName())
				case agent.CallbackEventDeleted:
					assert.Equal(t, "obj1-del", callback.Obj.GetName())
				}
			}
		}
	})
}
