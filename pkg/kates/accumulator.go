package kates

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
)

// The Accumulator struct is used to efficiently maintain an in-memory copy of kubernetes resources
// present in a cluster in a form that is easy for business logic to process. It functions as a
// bridge between delta-based kubernetes watches on individual Kinds and the complete/consistent set
// of objects on which business logic needs to operate. In that sense it accumulates both multiple
// kinds of kubernetes resources into a single snapshot, as well as accumulating deltas on
// individual objects into relevant sets of objects.
//
// The Goals/Requirements below are based heavily on the needs of Ambassador as they have evolved
// over the years. A lot of this comes down to the fact that unlike the exemplary
// deployment/replicaset controller examples which typically operate on a single resource and render
// it into another (deployment -> N replicasets, replicaset -> N pods), Ambassador's controller
// logic has some additional requirements:
//
//  1. Complete knowledge of resources in a cluster. Because many thousands of Mappings are
//     ultimately assembled into a single envoy configuration responsible for ingress into the
//     cluster, the consequences of producing an envoy configuration when you e.g. know about only
//     half of those Mappings is catastrophic (you are black-holing half your traffic).
//
//  2. Complete knowledge of multiple resources. Instead of having one self contained input like a
//     deployment or a replicaset, Ambassador's business logic has many inputs, and the consequence
//     of producing an envoy without knowledge of *all* of those inputs is equally catastrophic,
//     e.g. it's no use knowing about all the Mappings if you don't know about any of the Hosts yet.
//
// Goals/Requirements:
//
//  1. Bootstrap of a single Kind: the Accumulator will ensure that all pre-existing resources of
//     that Kind have been loaded into memory prior to triggering any notifications. This guarantees
//     we will never trigger business logic on an egregiously incomplete view of the cluster
//     (e.g. when 500 out of 1000 Mappings have been loaded) and makes it safe for the business
//     logic to assume complete knowledge of the cluster.
//
//  2. When multiple Kinds are needed by a controller, the Accumulator will not notify the
//     controller until all the Kinds have been fully bootstrapped.
//
//  3. Graceful load shedding: When the rate of change of resources is very fast, the API and
//     implementation are structured so that individual object deltas get coalesced into a single
//     snapshot update. This prevents excessively triggering business logic to process an entire
//     snapshot for each individual object change that occurs.
type Accumulator struct {
	client *Client
	fields map[string]*field
	// keyed by unKey(*Unstructured), tracks excluded resources for filtered updates
	excluded map[string]bool
	synced   int
	changed  chan struct{}
	mutex    sync.Mutex
}

type field struct {
	query    Query
	selector Selector
	mapping  *meta.RESTMapping

	// The values and deltas map are keyed by unKey(*Unstructured)
	values map[string]*Unstructured
	// The values map has a true for a new or update object, false for a deleted object.
	deltas map[string]*Delta

	synced      bool
	firstUpdate bool
}

type DeltaType int

const (
	ObjectAdd DeltaType = iota
	ObjectUpdate
	ObjectDelete
)

func (dt DeltaType) MarshalJSON() ([]byte, error) {
	switch dt {
	case ObjectAdd:
		return []byte(`"add"`), nil
	case ObjectUpdate:
		return []byte(`"update"`), nil
	case ObjectDelete:
		return []byte(`"delete"`), nil
	default:
		panic("missing case")
	}
}

func (dt *DeltaType) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	switch str {
	case "add":
		*dt = ObjectAdd
	case "update":
		*dt = ObjectUpdate
	case "delete":
		*dt = ObjectDelete
	default:
		return fmt.Errorf("unrecognized delta type: %s", str)
	}

	return nil
}

type Delta struct {
	TypeMeta   `json:""`
	ObjectMeta `json:"metadata,omitempty"`
	DeltaType  DeltaType `json:"deltaType"`
}

func newDelta(deltaType DeltaType, obj *Unstructured) *Delta {
	// We don't want all of the object, just a subset.
	return &Delta{
		TypeMeta: TypeMeta{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
		},
		ObjectMeta: ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			// Not sure we need this, but it marshals as null if we don't provide it.
			CreationTimestamp: obj.GetCreationTimestamp(),
		},
		DeltaType: deltaType,
	}
}

func newAccumulator(ctx context.Context, client *Client, queries ...Query) *Accumulator {
	changed := make(chan struct{})

	fields := make(map[string]*field)
	rawUpdateCh := make(chan rawUpdate)

	for _, q := range queries {
		mapping, err := client.mappingFor(q.Kind)
		if err != nil {
			panic(err)
		}
		sel, err := ParseSelector(q.LabelSelector)
		if err != nil {
			panic(err)
		}
		fields[q.Name] = &field{
			query:    q,
			mapping:  mapping,
			selector: sel,
			values:   make(map[string]*Unstructured),
			deltas:   make(map[string]*Delta),
		}
		client.watchRaw(ctx, q, rawUpdateCh, client.cliFor(mapping, q.Namespace))
	}

	acc := &Accumulator{client, fields, map[string]bool{}, 0, changed, sync.Mutex{}}

	// This coalesces reads from rawUpdateCh to notifications that changes are available to be
	// processed. This loop along with the logic in storeField guarantees the 3
	// Goals/Requirements listed in the documentation for the Accumulator struct, i.e. Ensuring
	// all Kinds are bootstrapped before any notification occurs, as well as ensuring that we
	// continue to coalesce updates in the background while business logic is executing in order
	// to ensure graceful load shedding.
	go func() {
		canSend := false

		for {
			var rawUp rawUpdate
			if canSend {
				select {
				case changed <- struct{}{}:
					canSend = false
					continue
				case rawUp = <-rawUpdateCh:
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case rawUp = <-rawUpdateCh:
				case <-ctx.Done():
					return
				}
			}

			// Don't overwrite canSend if storeField returns false. We may not yet have
			// had a chance to send a notification down the changed channel.
			if acc.storeUpdate(rawUp) {
				canSend = true
			}
		}
	}()

	return acc
}

func (a *Accumulator) Changed() chan struct{} {
	return a.changed
}

func (a *Accumulator) Update(target interface{}) bool {
	return a.UpdateWithDeltas(target, nil)
}

func (a *Accumulator) UpdateWithDeltas(target interface{}, deltas *[]*Delta) bool {
	return a.FilteredUpdate(target, deltas, nil)
}

// The FilteredUpdate method updates the target snapshot with only those resources for which
// "predicate" returns true. The predicate is only called when objects are added/updated, it is not
// repeatedly called on objects that have not changed. The predicate must not modify its argument.
func (a *Accumulator) FilteredUpdate(target interface{}, deltas *[]*Delta, predicate func(*Unstructured) bool) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.update(reflect.ValueOf(target), deltas, predicate)
}

func (a *Accumulator) storeUpdate(update rawUpdate) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	field := a.fields[update.name]
	if update.new != nil {
		key := unKey(update.new)
		oldValue, oldExists := field.values[key]
		field.values[key] = update.new

		if oldExists && oldValue.GetResourceVersion() == update.new.GetResourceVersion() {
			// no delta in this case, we have already delivered the new value and the delta via a
			// patch
		} else {
			if update.old == nil {
				field.deltas[key] = newDelta(ObjectAdd, update.new)
			} else {
				field.deltas[key] = newDelta(ObjectUpdate, update.new)
			}
		}
	} else if update.old != nil {
		key := unKey(update.old)
		_, oldExists := field.values[key]
		delete(field.values, key)

		if !oldExists {
			// no delta in this case, we have already delivered the deletion and the delta via a
			// patch
		} else {
			field.deltas[key] = newDelta(ObjectDelete, update.old)
		}
	}
	if update.informer.HasSynced() && !field.synced {
		field.synced = true
		a.synced += 1
	}
	return a.synced >= len(a.fields)
}

func (a *Accumulator) updateField(target reflect.Value, name string, field *field, deltas *[]*Delta,
	predicate func(*Unstructured) bool) bool {
	a.client.patchWatch(field)

	if field.firstUpdate && len(field.deltas) == 0 {
		return false
	}

	field.firstUpdate = true
	for key, delta := range field.deltas {
		delete(field.deltas, key)
		if deltas != nil {
			*deltas = append(*deltas, delta)
		}

		if predicate != nil {
			if delta.DeltaType == ObjectDelete {
				delete(a.excluded, key)
			} else {
				un := field.values[key]
				if predicate(un) {
					delete(a.excluded, key)
				} else {
					a.excluded[key] = true
				}
			}
		}
	}

	var items []*Unstructured
	for key, un := range field.values {
		if a.excluded[key] {
			continue
		}
		items = append(items, un)
	}

	jsonBytes, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}

	fieldEntry, ok := target.Type().Elem().FieldByName(name)
	if !ok {
		panic(fmt.Sprintf("no such field: %q", name))
	}

	var val reflect.Value
	if fieldEntry.Type.Kind() == reflect.Slice {
		val = reflect.New(fieldEntry.Type)
		err := json.Unmarshal(jsonBytes, val.Interface())
		if err != nil {
			panic(err)
		}
	} else if fieldEntry.Type.Kind() == reflect.Map {
		val = reflect.MakeMap(fieldEntry.Type)
		for _, item := range items {
			innerVal := reflect.New(fieldEntry.Type.Elem())
			err := convert(item, innerVal.Interface())
			if err != nil {
				panic(err)
			}
			val.SetMapIndex(reflect.ValueOf(item.GetName()), reflect.Indirect(innerVal))
		}
	} else {
		panic(fmt.Sprintf("don't know how to unmarshal to: %v", fieldEntry.Type))
	}

	target.Elem().FieldByName(name).Set(reflect.Indirect(val))

	return true
}

func (a *Accumulator) update(target reflect.Value, deltas *[]*Delta, predicate func(*Unstructured) bool) bool {
	if deltas != nil {
		*deltas = nil
	}

	updated := false
	for name, field := range a.fields {
		if a.updateField(target, name, field, deltas, predicate) {
			updated = true
		}
	}

	return updated
}

func unKeySort(items []*Unstructured) {
	sort.Slice(items, func(i, j int) bool {
		ik := unKey(items[i])
		jk := unKey(items[j])
		return strings.Compare(ik, jk) < 0
	})
}
