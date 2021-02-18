package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/consulwatch"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
)

type Fake struct {
	k8sSource    *fakeK8sSource
	watcher      *fakeWatcher
	snapshot     *atomic.Value
	store        *K8sStore
	bootstrapped chan struct{}
}

func NewFake() *Fake {
	store := NewK8sStore()
	return &Fake{
		k8sSource:    &fakeK8sSource{store: store},
		snapshot:     &atomic.Value{},
		store:        store,
		bootstrapped: make(chan struct{}),
	}
}

func RunFake(ctx context.Context) *Fake {
	fake := NewFake()
	go fake.Run(ctx)
	return fake
}

func (f *Fake) Run(ctx context.Context) {
	interestingTypes := GetInterestingTypes(ctx, nil)
	queries := GetQueries(ctx, interestingTypes)
	watcherLoop(ctx, f.snapshot, f.k8sSource, queries, f.watcher, func(ctx context.Context) {
		close(f.bootstrapped)
	})
}

func (f *Fake) GetSnapshotBytes() []byte {
	<-f.bootstrapped
	return f.snapshot.Load().([]byte)
}

func (f *Fake) GetSnapshotString() string {
	return string(f.GetSnapshotBytes())
}

func (f *Fake) GetSnapshot() snapshot.Snapshot {
	var result snapshot.Snapshot
	err := json.Unmarshal(f.GetSnapshotBytes(), &result)
	if err != nil {
		panic(err)
	}
	return result
}

func (f *Fake) ApplyFile(filename string) {
	f.store.UpsertFile(filename)
	f.k8sSource.notify()
}

type fakeK8sSource struct {
	store    *K8sStore
	watchers []*fakeK8sWatcher
}

func (fs *fakeK8sSource) notify() {
	for _, fw := range fs.watchers {
		fw.notify()
	}
}

func (fs *fakeK8sSource) Watch(ctx context.Context, queries ...kates.Query) K8sWatcher {
	fw := &fakeK8sWatcher{fs.store.Cursor(), make(chan struct{}), queries}
	fs.watchers = append(fs.watchers, fw)
	return fw
}

type fakeK8sWatcher struct {
	cursor   *K8sStoreCursor
	notifyCh chan struct{}
	queries  []kates.Query
}

func (f *fakeK8sWatcher) notify() {
	go func() {
		f.notifyCh <- struct{}{}
	}()
}

func (f *fakeK8sWatcher) Changed() chan struct{} {
	return f.notifyCh
}

func (f *fakeK8sWatcher) FilteredUpdate(target interface{}, deltas *[]*kates.Delta, predicate func(*kates.Unstructured) bool) bool {
	byname := map[string][]kates.Object{}
	resources, newDeltas := f.cursor.Get()
	for _, obj := range resources {
		for _, q := range f.queries {
			var un *kates.Unstructured
			err := convert(obj, &un)
			if err != nil {
				panic(err)
			}
			if matches(q, obj) && predicate(un) {
				byname[q.Name] = append(byname[q.Name], obj)
			}
		}
	}

	// XXX: this stuff is copied from kates/accumulator.go
	targetVal := reflect.ValueOf(target)
	targetType := targetVal.Type().Elem()
	for name, v := range byname {
		fieldEntry, ok := targetType.FieldByName(name)
		if !ok {
			panic(fmt.Sprintf("no such field: %q", name))
		}
		val := reflect.New(fieldEntry.Type)
		err := convert(v, val.Interface())
		if err != nil {
			panic(err)
		}
		targetVal.Elem().FieldByName(name).Set(reflect.Indirect(val))
	}

	*deltas = newDeltas

	return len(newDeltas) > 0
}

func matches(query kates.Query, obj kates.Object) bool {
	kind := canon(query.Kind)
	gvk := obj.GetObjectKind().GroupVersionKind()
	return kind == canon(gvk.Kind)
}

type fakeWatcher struct {
}

func (f *fakeWatcher) Watch(resolver *amb.ConsulResolver, mapping *amb.Mapping, endpoints chan consulwatch.Endpoints) Stopper {
	return &fakeStopper{}
}

type fakeStopper struct {
}

func (f *fakeStopper) Stop() {}
