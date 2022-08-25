package agent

import (
	"fmt"
	"sync"

	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
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
	sotw map[types.UID]*unstructured.Unstructured

	mux sync.Mutex
}

// ApplicationStore is responsible for collecting the state transition and the
// state of the world for Argo Application resources in a k8s cluster.
type ApplicationStore struct {
	// deltas is a collection having just metadata fields for Argo Applications
	// that had their state changed. A state change can be: creation, update and
	// deletion.
	deltas []*kates.Delta

	// sotw refers to the state of the world which holds the current state
	// of all rollouts in a k8s cluster.
	sotw map[types.UID]*unstructured.Unstructured

	mux sync.Mutex
}

// NewApplicationStore is the main ApplicationStore constructor.
func NewApplicationStore() *ApplicationStore {
	return &ApplicationStore{}
}

// Deltas is the accessor method for the deltas attribute.
func (a *ApplicationStore) Deltas() []*kates.Delta {
	return a.deltas
}

// StateOfWorld will convert the internal state of the world into a
// []*unstructured.Unstructured
func (a *ApplicationStore) StateOfWorld() []*unstructured.Unstructured {
	results := []*unstructured.Unstructured{}
	for _, v := range a.sotw {
		results = append(results, v)
	}
	return results
}

// FromCallback will populate and return an Application store based on a GenericCallback
func (a *ApplicationStore) FromCallback(callback *GenericCallback) (*ApplicationStore, error) {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.sotw == nil {
		uMap, err := toUntructuredMap(callback.Sotw)
		if err != nil {
			return nil, err
		}
		a.sotw = uMap
	}
	a.deltas = append(a.deltas, toDelta(callback.Obj, callback.EventType))
	switch callback.EventType {
	case CallbackEventAdded, CallbackEventUpdated:
		a.sotw[callback.Obj.GetUID()] = callback.Obj
	case CallbackEventDeleted:
		delete(a.sotw, callback.Obj.GetUID())
	}
	return a, nil
}

// NewRolloutStore is the main RolloutStore constructor.
func NewRolloutStore() *RolloutStore {
	return &RolloutStore{}
}

// Deltas is the accessor method for the deltas attribute.
func (s *RolloutStore) Deltas() []*kates.Delta {
	return s.deltas
}

// StateOfWorld will convert the internal state of the world into a
// []*unstructured.Unstructured
func (a *RolloutStore) StateOfWorld() []*unstructured.Unstructured {
	results := []*unstructured.Unstructured{}
	for _, v := range a.sotw {
		results = append(results, v)
	}
	return results
}

// FromCallback will populate and return a Rollout store based on a GenericCallback
func (r *RolloutStore) FromCallback(callback *GenericCallback) (*RolloutStore, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.sotw == nil {
		uMap, err := toUntructuredMap(callback.Sotw)
		if err != nil {
			return nil, err
		}
		r.sotw = uMap
	}
	r.deltas = append(r.deltas, toDelta(callback.Obj, callback.EventType))
	switch callback.EventType {
	case CallbackEventAdded, CallbackEventUpdated:
		r.sotw[callback.Obj.GetUID()] = callback.Obj
	case CallbackEventDeleted:
		delete(r.sotw, callback.Obj.GetUID())
	}
	return r, nil
}

func toUntructuredMap(objs []interface{}) (map[types.UID]*unstructured.Unstructured, error) {
	results := make(map[types.UID]*unstructured.Unstructured)
	for _, obj := range objs {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("toUntructuredSlice error: obj is %T: expected unstructured.Unstructured", obj)
		}
		results[u.GetUID()] = u
	}
	return results, nil
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
