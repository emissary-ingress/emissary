package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/cmd/entrypoint/internal/testqueue"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
)

// The Fake struct is a test harness for edgestack. Its goals are to help us fill out our test
// pyramid by making it super easy to create unit-like tests directly from the snapshots, bug
// reports, and other inputs provided by users who find regressions and/or encounter other problems
// in the field. Since we have no shortage of these reports, if we make it easy to create tests from
// them, we will fill out our test pyramid quickly and hopefully reduce our rate of
// regressions. This also means the tests produced this way need to scale well both in terms of
// execution time/parallelism as well as flakiness since we will quickly have a large number of
// these tests.
//
// The way this works is by isolating via dependency injection the key portions of the control plane
// where the bulk of our business logic is implemented. The Fake utilities directly feed this
// lightweight control plane its input as specified by the test code without passing the resources
// all the way through a real kubernetes API server and/or a real consul deployment. This is not
// only significantly more efficient than spinning up real kubernetes and/or consul deployments, but
// it also lets us precisely control the order of events thereby a) removing the nondeterminism that
// leads to flaky tests, and b) also allowing us to deliberately create/recreate the sort of low
// probability sequence of events that are often at the root of heisenbugs.
//
// The key to being able to build tests this way is expressing our business logic as "hermetically
// sealed" libraries, i.e. libraries with no/few hardcoded dependencies. This doesn't have to be
// done in a fancy/elegant way, it is well worth practicing "stupidly mechanical dependency
// injection" in order to quickly excise some business logic of its hardcoded dependencies and
// enable this sort of testing.
//
// See TestFakeHello, TestFakeHelloWithEnvoyConfig, and TestFakeHelloConsul for examples of how to
// get started using this struct to write tests.
type Fake struct {
	// These are all read only fields. They implement the dependencies that get injected into
	// the watcher loop.
	config FakeConfig
	T      *testing.T
	group  *dgroup.Group
	cancel context.CancelFunc

	k8sSource       *fakeK8sSource
	watcher         *fakeWatcher
	istioCertSource *fakeIstioCertSource
	// This group of fields are used to store kubernetes resources and consul endpoint data and
	// provide explicit control over when changes to that data are sent to the control plane.
	k8sStore       *K8sStore
	consulStore    *ConsulStore
	k8sNotifier    *Notifier
	consulNotifier *Notifier

	// This holds the current snapshot.
	currentSnapshot *atomic.Value

	fastpath     *testqueue.Queue // All fastpath snapshots that have been produced.
	snapshots    *testqueue.Queue // All snapshots that have been produced.
	envoyConfigs *testqueue.Queue // All envoyConfigs that have been produced.

	// This is used to make Teardown idempotent.
	teardownOnce sync.Once

	ambassadorMeta *snapshot.AmbassadorMetaInfo

	DiagdBindPort string
}

// FakeConfig provides option when constructing a new Fake.
type FakeConfig struct {
	EnvoyConfig bool          // If true then the Fake will produce envoy configs in addition to Snapshots.
	DiagdDebug  bool          // If true then diagd will have debugging enabled
	Timeout     time.Duration // How long to wait for snapshots and/or envoy configs to become available.
}

func (fc *FakeConfig) fillDefaults() {
	if fc.Timeout == 0 {
		fc.Timeout = 10 * time.Second
	}
}

// NewFake will construct a new Fake object. See RunFake for a convenient way to handle construct,
// Setup, and Teardown of a Fake with one line of code.
func NewFake(t *testing.T, config FakeConfig) *Fake {
	config.fillDefaults()
	ctx, cancel := context.WithCancel(dlog.NewTestContext(t, false))
	k8sStore := NewK8sStore()
	consulStore := NewConsulStore()

	fake := &Fake{
		config: config,
		T:      t,
		cancel: cancel,
		group:  dgroup.NewGroup(ctx, dgroup.GroupConfig{EnableWithSoftness: true}),

		k8sStore:       k8sStore,
		consulStore:    consulStore,
		k8sNotifier:    NewNotifier(),
		consulNotifier: NewNotifier(),

		currentSnapshot: &atomic.Value{},

		fastpath:     testqueue.NewQueue(t, config.Timeout),
		snapshots:    testqueue.NewQueue(t, config.Timeout),
		envoyConfigs: testqueue.NewQueue(t, config.Timeout),
	}

	fake.k8sSource = &fakeK8sSource{fake: fake, store: k8sStore}
	fake.watcher = &fakeWatcher{fake: fake, store: consulStore}
	fake.istioCertSource = &fakeIstioCertSource{}

	return fake
}

// RunFake will create a new fake, invoke its Setup method and register its Teardown method as a
// Cleanup function with the test object.
func RunFake(t *testing.T, config FakeConfig, ambMeta *snapshot.AmbassadorMetaInfo) *Fake {
	fake := NewFake(t, config)
	fake.SetAmbassadorMeta(ambMeta)
	fake.Setup()
	fake.T.Cleanup(fake.Teardown)
	return fake
}

// Setup will start up all the goroutines needed for this fake edgestack instance. Depending on the
// FakeConfig supplied wen constructing the Fake, this may also involve launching external
// processes, you should therefore ensure that you call Teardown whenever you call Setup.
func (f *Fake) Setup() {
	if f.config.EnvoyConfig {
		_, err := dexec.LookPath("diagd")
		if err != nil {
			f.T.Fatal("unable to find diagd, cannot run")
		}

		f.group.Go("snapshot_server", func(ctx context.Context) error {
			return snapshotServer(ctx, f.currentSnapshot)
		})

		f.DiagdBindPort = GetDiagdBindPort()

		f.group.Go("diagd", func(ctx context.Context) error {
			args := []string{
				"diagd",
				"/tmp",
				"/tmp/bootstrap-ads.json",
				"/tmp/envoy.json",
				"--no-envoy",
				"--host", "127.0.0.1",
				"--port", f.DiagdBindPort,
			}

			if f.config.DiagdDebug {
				args = append(args, "--debug")
			}

			cmd := dexec.CommandContext(ctx, args[0], args[1:]...)
			if envbool("DEV_SHUTUP_DIAGD") {
				devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				cmd.Stdout = devnull
				cmd.Stderr = devnull
			}
			err := cmd.Run()
			if err != nil {
				exErr, ok := err.(*dexec.ExitError)
				if ok {
					f.T.Logf("diagd exited with error: %+v", exErr)
					return nil
				}
			}
			return err
		})
	}
	f.group.Go("fake-watcher", f.runWatcher)

}

// GetFeatures grabs features from diagd. Yup, it's a little ugly.
func (f *Fake) GetFeatures(ctx context.Context, features interface{}) error {
	// If EnvoyConfig isn't set, we're not running diagd, so there's no way to get the
	// features. Just return an error immediately.
	if !f.config.EnvoyConfig {
		return fmt.Errorf("Features are not available with EnvoyConfig false")
	}

	// The way we get the features is by making a request to diagd. Why, you ask, is the
	// features dict not just always part of the IR dump? It's basically a performance thing
	// at present.
	//
	// TODO(Flynn): That's a stupid reason and we should fix it.
	featuresURL := fmt.Sprintf("http://localhost:%s/_internal/v0/features", f.DiagdBindPort)

	req, err := http.NewRequestWithContext(ctx, "GET", featuresURL, nil)

	if err != nil {
		return err
	}

	// This is test code, so we can just always force X-Ambassador-Diag-IP to 127.0.0.1,
	// so that diagd will trust the request.
	req.Header.Set("X-Ambassador-Diag-IP", "127.0.0.1")
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	// err = ioutil.WriteFile("/tmp/ambassador-features.json", body, 0644)
	//
	// if err != nil { return err }

	// Trust our caller to have passed in something that we can unmarshal into. This is
	// particularly relevant right now because there isn't a real Go type for the Features
	// dict, so our caller is probably handing in something to look at just a subset.
	//
	// TODO(Flynn): Really, we should just have an IR Features type...
	return json.Unmarshal(body, features)
}

// Teardown will clean up anything that Setup has started. It is idempotent. Note that if you use
// RunFake Setup will be called and Teardown will be automatically registered as a Cleanup function
// with the supplied testing.T
func (f *Fake) Teardown() {
	f.teardownOnce.Do(func() {
		f.cancel()
		err := f.group.Wait()
		if err != nil && err != context.Canceled {
			f.T.Fatalf("fake edgestack errored out: %+v", err)
		}
	})
}

func (f *Fake) runWatcher(ctx context.Context) error {
	interestingTypes := GetInterestingTypes(ctx, nil)
	queries := GetQueries(ctx, interestingTypes)

	return watchAllTheThingsInternal(
		ctx,
		f.currentSnapshot, // encoded
		f.k8sSource,
		queries,
		f.watcher.Watch, // watchConsulFunc
		f.istioCertSource,
		f.notifySnapshot,
		f.notifyFastpath,
		f.ambassadorMeta,
	)
}

func (f *Fake) notifyFastpath(ctx context.Context, fastpath *ambex.FastpathSnapshot) {
	f.fastpath.Add(f.T, fastpath)
}

func (f *Fake) GetEndpoints(predicate func(*ambex.Endpoints) bool) (*ambex.Endpoints, error) {
	f.T.Helper()
	untyped, err := f.fastpath.Get(f.T, func(obj interface{}) bool {
		fastpath := obj.(*ambex.FastpathSnapshot)
		return predicate(fastpath.Endpoints)
	})
	if err != nil {
		return nil, err
	}
	return untyped.(*ambex.FastpathSnapshot).Endpoints, nil
}

func (f *Fake) AssertEndpointsEmpty(timeout time.Duration) {
	f.T.Helper()
	f.fastpath.AssertEmpty(f.T, timeout, "endpoints queue not empty")
}

type SnapshotEntry struct {
	Disposition SnapshotDisposition
	Snapshot    *snapshot.Snapshot
}

func (entry SnapshotEntry) String() string {
	snapshot := "nil"
	if entry.Snapshot != nil {
		snapshot = fmt.Sprintf("&%#v", *entry.Snapshot)
	}
	return fmt.Sprintf("{Disposition: %v, Snapshot: %s}", entry.Disposition, snapshot)
}

// We pass this into the watcher loop to get notified when a snapshot is produced.
func (f *Fake) notifySnapshot(ctx context.Context, disp SnapshotDisposition, snapJSON []byte) error {
	if disp == SnapshotReady && f.config.EnvoyConfig {
		if err := notifyReconfigWebhooksFunc(ctx, &noopNotable{}, false); err != nil {
			return err
		}
		f.appendEnvoyConfig(ctx)
	}

	var snap *snapshot.Snapshot
	err := json.Unmarshal(snapJSON, &snap)
	if err != nil {
		f.T.Fatalf("error decoding snapshot: %+v", err)
	}

	f.snapshots.Add(f.T, SnapshotEntry{disp, snap})
	return nil
}

// GetSnapshotEntry will return the next SnapshotEntry that satisfies the supplied predicate.
func (f *Fake) GetSnapshotEntry(predicate func(SnapshotEntry) bool) (SnapshotEntry, error) {
	f.T.Helper()
	untyped, err := f.snapshots.Get(f.T, func(obj interface{}) bool {
		entry := obj.(SnapshotEntry)
		return predicate(entry)
	})
	if err != nil {
		return SnapshotEntry{}, err
	}
	return untyped.(SnapshotEntry), nil
}

// GetSnapshot will return the next snapshot that satisfies the supplied predicate.
func (f *Fake) GetSnapshot(predicate func(*snapshot.Snapshot) bool) (*snapshot.Snapshot, error) {
	f.T.Helper()
	entry, err := f.GetSnapshotEntry(func(entry SnapshotEntry) bool {
		return entry.Disposition == SnapshotReady && predicate(entry.Snapshot)
	})
	if err != nil {
		return nil, err
	}
	return entry.Snapshot, nil
}

func (f *Fake) appendEnvoyConfig(ctx context.Context) {
	msg, err := ambex.Decode(ctx, "/tmp/envoy.json")
	if err != nil {
		f.T.Fatalf("error decoding envoy.json after sending snapshot to python: %+v", err)
	}
	bs := msg.(*v3bootstrap.Bootstrap)
	f.envoyConfigs.Add(f.T, bs)
}

// GetEnvoyConfig will return the next envoy config that satisfies the supplied predicate.
func (f *Fake) GetEnvoyConfig(predicate func(*v3bootstrap.Bootstrap) bool) (*v3bootstrap.Bootstrap, error) {
	f.T.Helper()
	untyped, err := f.envoyConfigs.Get(f.T, func(obj interface{}) bool {
		return predicate(obj.(*v3bootstrap.Bootstrap))
	})
	if err != nil {
		return nil, err
	}
	return untyped.(*v3bootstrap.Bootstrap), nil
}

// AutoFlush will cause a flush whenever any inputs are modified.
func (f *Fake) AutoFlush(enabled bool) {
	f.k8sNotifier.AutoNotify(enabled)
	f.consulNotifier.AutoNotify(enabled)
}

// Feed will cause inputs from all datasources to be delivered to the control plane.
func (f *Fake) Flush() {
	f.k8sNotifier.Notify()
	f.consulNotifier.Notify()
}

// sets the ambassador meta info that should get sent in each snapshot
func (f *Fake) SetAmbassadorMeta(ambMeta *snapshot.AmbassadorMetaInfo) {
	f.ambassadorMeta = ambMeta
}

// UpsertFile will parse the contents of the file as yaml and feed them into the control plane
// created or updating any overlapping resources that exist.
func (f *Fake) UpsertFile(filename string) error {
	if err := f.k8sStore.UpsertFile(filename); err != nil {
		return err
	}
	f.k8sNotifier.Changed()
	return nil
}

// UpsertYAML will parse the provided YAML and feed the resources in it into the control plane,
// creating or updating any overlapping resources that exist.
func (f *Fake) UpsertYAML(yaml string) error {
	if err := f.k8sStore.UpsertYAML(yaml); err != nil {
		return err
	}
	f.k8sNotifier.Changed()
	return nil
}

// Upsert will update (or if necessary create) the supplied resource in the fake k8s datastore.
func (f *Fake) Upsert(resource kates.Object) error {
	if err := f.k8sStore.Upsert(resource); err != nil {
		return err
	}
	f.k8sNotifier.Changed()
	return nil
}

// Delete will removes the specified resource from the fake k8s datastore.
func (f *Fake) Delete(kind, namespace, name string) error {
	if err := f.k8sStore.Delete(kind, namespace, name); err != nil {
		return err
	}
	f.k8sNotifier.Changed()
	return nil
}

// ConsulEndpoint stores the supplied consul endpoint data.
func (f *Fake) ConsulEndpoint(datacenter, service, address string, port int, tags ...string) {
	f.consulStore.ConsulEndpoint(datacenter, service, address, port, tags...)
	f.consulNotifier.Changed()
}

// SendIstioCertUpdate sends the supplied Istio certificate update.
func (f *Fake) SendIstioCertUpdate(update IstioCertUpdate) {
	f.istioCertSource.updateChannel <- update
}

type fakeK8sSource struct {
	fake  *Fake
	store *K8sStore
}

func (fs *fakeK8sSource) Watch(ctx context.Context, queries ...kates.Query) (K8sWatcher, error) {
	fw := &fakeK8sWatcher{fs.store.Cursor(), make(chan struct{}), queries}
	fs.fake.k8sNotifier.Listen(func() {
		go func() {
			fw.notifyCh <- struct{}{}
		}()
	})
	return fw, nil
}

type fakeK8sWatcher struct {
	cursor   *K8sStoreCursor
	notifyCh chan struct{}
	queries  []kates.Query
}

func (f *fakeK8sWatcher) Changed() chan struct{} {
	return f.notifyCh
}

func (f *fakeK8sWatcher) FilteredUpdate(_ context.Context, target interface{}, deltas *[]*kates.Delta, predicate func(*kates.Unstructured) bool) (bool, error) {
	byname := map[string][]kates.Object{}
	resources, newDeltas, err := f.cursor.Get()
	if err != nil {
		return false, err
	}
	for _, obj := range resources {
		for _, q := range f.queries {
			var un *kates.Unstructured
			err := convert(obj, &un)
			if err != nil {
				return false, err
			}
			doesMatch, err := matches(q, obj)
			if err != nil {
				return false, err
			}
			if doesMatch && predicate(un) {
				byname[q.Name] = append(byname[q.Name], obj)
			}
		}
	}

	// XXX: this stuff is copied from kates/accumulator.go
	targetVal := reflect.ValueOf(target)
	targetType := targetVal.Type().Elem()
	for _, q := range f.queries {
		name := q.Name
		v := byname[q.Name]
		fieldEntry, ok := targetType.FieldByName(name)
		if !ok {
			return false, fmt.Errorf("no such field: %q", name)
		}
		val := reflect.New(fieldEntry.Type)
		err := convert(v, val.Interface())
		if err != nil {
			return false, err
		}
		targetVal.Elem().FieldByName(name).Set(reflect.Indirect(val))
	}

	*deltas = newDeltas

	return len(newDeltas) > 0, nil
}

func matches(query kates.Query, obj kates.Object) (bool, error) {
	queryKind, err := canon(query.Kind)
	if err != nil {
		return false, err
	}
	objKind, err := canon(obj.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return false, err
	}
	return queryKind == objKind, nil
}

type fakeWatcher struct {
	fake  *Fake
	store *ConsulStore
}

func (f *fakeWatcher) Watch(ctx context.Context, resolver *amb.ConsulResolver, svc string, endpoints chan consulwatch.Endpoints) (Stopper, error) {
	var sent consulwatch.Endpoints
	stop := f.fake.consulNotifier.Listen(func() {
		ep, ok := f.store.Get(resolver.Spec.Datacenter, svc)
		if ok && !reflect.DeepEqual(ep, sent) {
			endpoints <- ep
			sent = ep
		}
	})
	return &fakeStopper{stop}, nil
}

type fakeStopper struct {
	stop StopFunc
}

func (f *fakeStopper) Stop() {
	f.stop()
}

type fakeIstioCertSource struct {
	updateChannel chan IstioCertUpdate
}

func (src *fakeIstioCertSource) Watch(ctx context.Context) (IstioCertWatcher, error) {
	src.updateChannel = make(chan IstioCertUpdate)

	return &istioCertWatcher{
		updateChannel: src.updateChannel,
	}, nil
}
