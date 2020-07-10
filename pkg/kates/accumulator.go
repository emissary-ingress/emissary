package kates

import (
	"bytes"
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
//     we will never trigger business logic on an outdated view of the cluster (e.g. when 500 out of
//     100 Mappings have been loaded) and makes it safe for the business logic to assume complete
//     knowledge of the cluster.
//
//  2. When multiple Kinds are needed by a controller, the Accumulator will not notify the
//     controller until all the Kinds have been fully bootstrapped.
//
//  3. Graceful load shedding: When the rate of change of resources is very fast, the API and
//     implementation are structured so that individual object deltas get coalesced into a single
//     snapshot update. This prevents excessively triggering business logic to process an entire
//     snapshot for each individual object change that occurs.
type Accumulator struct {
	client  *Client
	fields  map[string]*field
	mapsels map[string]mapsel
	changed chan struct{}
	mutex   sync.Mutex
}

type field struct {
	items []*Unstructured
	prev  []byte
}

type mapsel struct {
	mapping  *meta.RESTMapping
	selector Selector
	query    Query
}

func newAccumulator(ctx context.Context, client *Client, queries ...Query) *Accumulator {
	changed := make(chan struct{})

	mapsels := make(map[string]mapsel)
	channel := make(chan rawUpdate)

	for _, q := range queries {
		mapping, err := client.mappingFor(q.Kind)
		if err != nil {
			panic(err)
		}
		sel, err := ParseSelector(q.LabelSelector)
		if err != nil {
			panic(err)
		}
		mapsels[q.Name] = mapsel{mapping, sel, q}
		client.watchRaw(ctx, q.Kind, channel, client.cliFor(mapping, q.Namespace), q.LabelSelector, q.Name)
	}

	acc := &Accumulator{client, make(map[string]*field), mapsels, changed, sync.Mutex{}}

	go func() {
		canSend := false

		for {
			var rawUp rawUpdate
			if canSend {
				select {
				case changed <- struct{}{}:
					canSend = false
					continue
				case rawUp = <-channel:
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case rawUp = <-channel:
				case <-ctx.Done():
					return
				}
			}
			if acc.storeField(rawUp.correlation.(string), rawUp.items) {
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
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.update(reflect.ValueOf(target))
}

func (a *Accumulator) field(name string) *field {
	f, ok := a.fields[name]
	if !ok {
		f = &field{}
		a.fields[name] = f
	}
	return f
}

func (a *Accumulator) storeField(name string, items []*Unstructured) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.field(name).items = items
	return len(a.fields) >= len(a.mapsels)
}

func (a *Accumulator) updateField(target reflect.Value, name string, items []*Unstructured) bool {
	field := a.field(name)

	unKeySort(items)

	jsonBytes, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}

	fieldEntry, ok := target.Type().Elem().FieldByName(name)
	if !ok {
		panic(fmt.Sprintf("no such field: %q", name))
	}

	if !bytes.Equal(field.prev, jsonBytes) {
		field.prev = jsonBytes
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
	return false

}

func (a *Accumulator) update(target reflect.Value) bool {
	updated := false
	for name, field := range a.fields {
		items := field.items[:]
		ms := a.mapsels[name]
		a.client.patchWatch(&items, ms.mapping, ms.selector)
		if a.updateField(target, name, items) {
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
