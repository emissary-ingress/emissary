package agent

import (
	"fmt"
	"sync"

	"github.com/datawire/ambassador/pkg/kates"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RolloutStore is responsible for collecting the state transition and the
// state of the world for Argo Rollout resources in a k8s cluster.
type RolloutStore struct {
	// deltas is a collection of just metadata fields for rollouts that
	// changed its state. A state change can be: creation, update and
	// deletion.
	deltas []*kates.Delta

	// sotw refers to the state of the world which holds the current state
	// of all rollouts in a k8s cluster.
	sotw []interface{}

	mux sync.Mutex
}

// NewRolloutStore is the main RolloutStore constructor.
func NewRolloutStore() *RolloutStore {
	return &RolloutStore{}
}

// Deltas is the accessor method for the deltas attribute.
func (s *RolloutStore) Deltas() []*kates.Delta {
	return s.deltas
}

// Sotw is the accessor method for the state of the world attribure.
func (s *RolloutStore) Sotw() []interface{} {
	return s.sotw
}

// FromCallback will populate and return a Rollout store based on a GenericCallback
func (s *RolloutStore) FromCallback(callback *GenericCallback) *RolloutStore {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.sotw = callback.Sotw
	s.deltas = append(s.deltas, toDelta(callback.Obj, callback.EventType))
	return s
}

func toDelta(obj *unstructured.Unstructured, t CallbackEventType) *kates.Delta {
	deltaType := toKatesDeltaType(t)
	return &kates.Delta{
		TypeMeta: kates.TypeMeta{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
		},
		ObjectMeta: kates.ObjectMeta{
			Name:              obj.GetName(),
			Namespace:         obj.GetNamespace(),
			CreationTimestamp: obj.GetCreationTimestamp(),
		},
		DeltaType: deltaType,
	}
}

func toKatesDeltaType(t CallbackEventType) kates.DeltaType {
	var kt kates.DeltaType
	switch t {
	case CallbackEventAdded:
		kt = kates.ObjectAdd
	case CallbackEventUpdated:
		kt = kates.ObjectUpdate
	case CallbackEventDeleted:
		kt = kates.ObjectDelete
	}
	return kt
}

// StateOfWorld will convert the internal state of the world into a
// []*unstructured.Unstructured
func (s *RolloutStore) StateOfWorld() ([]*unstructured.Unstructured, error) {
	results := []*unstructured.Unstructured{}
	for _, obj := range s.sotw {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("Rollout store error: obj is %T: expected unstructured.Unstructured", obj)
		}
		results = append(results, u)
	}
	return results, nil
}
