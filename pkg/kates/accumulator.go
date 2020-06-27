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

func NewAccumulator(ctx context.Context, client *Client, queries ...Query) *Accumulator {
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
		a.client.patch(&items, ms.mapping, ms.selector)
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
