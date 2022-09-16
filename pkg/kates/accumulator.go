package kates

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

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

type changeStatus int

const (
	awaitingDispatch changeStatus = iota
	dispatched
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
		return nil, fmt.Errorf("invalid DeltaType enum: %d", dt)
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

func NewDelta(deltaType DeltaType, obj *Unstructured) *Delta {
	return newDelta(deltaType, obj)
}

func NewDeltaFromObject(deltaType DeltaType, obj Object) (*Delta, error) {
	var un *Unstructured
	err := convert(obj, &un)
	if err != nil {
		return nil, err
	}
	return NewDelta(deltaType, un), nil
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

func newAccumulator(ctx context.Context, client *Client, queries ...Query) (*Accumulator, error) {
	changed := make(chan struct{})

	fields := make(map[string]*field)
	rawUpdateCh := make(chan rawUpdate)

	for _, q := range queries {
		field, err := client.newField(q)
		if err != nil {
			return nil, err
		}
		fields[q.Name] = field
		client.watchRaw(ctx, q, rawUpdateCh, client.cliFor(field.mapping, q.Namespace))
	}

	acc := &Accumulator{
		client:   client,
		fields:   fields,
		excluded: map[string]bool{},
		synced:   0,
		changed:  changed,
		mutex:    sync.Mutex{},
	}

	go acc.Listen(ctx, rawUpdateCh, client.maxAccumulatorInterval)

	return acc, nil
}

// Listen for updates from rawUpdateCh and sends notifications, coalescing reads as neccessary.
// This loop along with the logic in storeField isused to satisfy the 3 Goals/Requirements listed in
// the documentation for the Accumulator struct, i.e. Ensuring all Kinds are bootstrapped before any
// notification occurs, as well as ensuring that we continue to coalesce updates in the background while
// business logic is executing in order to ensure graceful load shedding.
func (a *Accumulator) Listen(ctx context.Context, rawUpdateCh <-chan rawUpdate, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var changeStatus changeStatus
	var lastChangeSent time.Time
	var synced bool

	sendUpdate := func() {
		a.changed <- struct{}{}
		changeStatus = dispatched
		lastChangeSent = time.Now()
	}

	for {
		select {
		// We have two paths here:
		// 1. If we get new data and it has been past our set interval since we last updated anything,
		//    we go ahead and immediately send that.
		//
		// 2. If we get new data but we just recently sent a change within our interval, we'll
		//    wait until we get our next Tick before sending a change.
		case rawUp := <-rawUpdateCh:
			synced = a.storeUpdate(rawUp)
			since := rawUp.ts.Sub(lastChangeSent)
			if synced && since >= interval {
				sendUpdate()
			} else {
				changeStatus = awaitingDispatch
			}
		case <-ticker.C:
			if synced && changeStatus == awaitingDispatch {
				sendUpdate()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Accumulator) Changed() <-chan struct{} {
	return a.changed
}

func (a *Accumulator) Update(ctx context.Context, target interface{}) (bool, error) {
	return a.UpdateWithDeltas(ctx, target, nil)
}

func (a *Accumulator) UpdateWithDeltas(ctx context.Context, target interface{}, deltas *[]*Delta) (bool, error) {
	return a.FilteredUpdate(ctx, target, deltas, nil)
}

// The FilteredUpdate method updates the target snapshot with only those resources for which
// "predicate" returns true. The predicate is only called when objects are added/updated, it is not
// repeatedly called on objects that have not changed. The predicate must not modify its argument.
func (a *Accumulator) FilteredUpdate(ctx context.Context, target interface{}, deltas *[]*Delta, predicate func(*Unstructured) bool) (bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.update(ctx, reflect.ValueOf(target), deltas, predicate)
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
	if update.synced && !field.synced {
		field.synced = true
		a.synced += 1
	}
	return a.synced >= len(a.fields)
}

func (a *Accumulator) updateField(
	ctx context.Context,
	target reflect.Value,
	name string,
	field *field,
	deltas *[]*Delta,
	predicate func(*Unstructured) bool,
) (bool, error) {
	if err := a.client.patchWatch(ctx, field); err != nil {
		return false, err
	}

	if field.firstUpdate && len(field.deltas) == 0 {
		return false, nil
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
		return false, err
	}

	fieldEntry, ok := target.Type().Elem().FieldByName(name)
	if !ok {
		return false, fmt.Errorf("no such field: %q", name)
	}

	var val reflect.Value
	if fieldEntry.Type.Kind() == reflect.Slice {
		val = reflect.New(fieldEntry.Type)
		err := json.Unmarshal(jsonBytes, val.Interface())
		if err != nil {
			return false, err
		}
	} else if fieldEntry.Type.Kind() == reflect.Map {
		val = reflect.MakeMap(fieldEntry.Type)
		for _, item := range items {
			innerVal := reflect.New(fieldEntry.Type.Elem())
			err := convert(item, innerVal.Interface())
			if err != nil {
				return false, err
			}
			val.SetMapIndex(reflect.ValueOf(item.GetName()), reflect.Indirect(innerVal))
		}
	} else {
		return false, fmt.Errorf("don't know how to unmarshal to: %v", fieldEntry.Type)
	}

	target.Elem().FieldByName(name).Set(reflect.Indirect(val))

	return true, nil
}

func (a *Accumulator) update(ctx context.Context, target reflect.Value, deltas *[]*Delta, predicate func(*Unstructured) bool) (bool, error) {
	if deltas != nil {
		*deltas = nil
	}

	updated := false
	for name, field := range a.fields {
		_updated, err := a.updateField(ctx, target, name, field, deltas, predicate)
		if _updated {
			updated = true
		}
		if err != nil {
			return updated, err
		}
	}

	return updated, nil
}
