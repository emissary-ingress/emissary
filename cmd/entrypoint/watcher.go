package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/pkg/acp"
	"github.com/datawire/ambassador/v2/pkg/debug"
	ecp_v2_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	"github.com/datawire/ambassador/v2/pkg/gateway"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/ambassador/v2/pkg/watt"
	"github.com/datawire/dlib/dlog"
)

func watcher(ctx context.Context, ambwatch *acp.AmbassadorWatcher, encoded *atomic.Value,
	fastpathCh chan<- *ambex.FastpathSnapshot, clusterID string, version string) {
	client, err := kates.NewClient(kates.ClientConfig{})
	if err != nil {
		panic(err)
	}

	serverTypeList, err := client.ServerPreferredResources()
	if err != nil {
		// It's possible that an error prevented listing some apigroups, but not all; so
		// process the output even if there is an error.
		dlog.Infof(ctx, "Warning, unable to list api-resources: %v", err)
	}

	interestingTypes := GetInterestingTypes(ctx, serverTypeList)
	queries := GetQueries(ctx, interestingTypes)

	ambassadorMeta := getAmbassadorMeta(GetAmbassadorId(), clusterID, version, client)

	// **** SETUP DONE for the Kubernetes Watcher

	notify := func(ctx context.Context, disposition SnapshotDisposition, _ []byte) {
		if disposition == SnapshotReady {
			notifyReconfigWebhooks(ctx, ambwatch)
		}
	}

	fastpathUpdate := func(ctx context.Context, fastpathSnapshot *ambex.FastpathSnapshot) {
		fastpathCh <- fastpathSnapshot
	}

	k8sSrc := newK8sSource(client)
	consulSrc := &consulWatcher{}
	istioCertSrc := newIstioCertSource()

	watcherLoop(ctx, encoded, k8sSrc, queries, consulSrc, istioCertSrc, notify, fastpathUpdate, ambassadorMeta)
}

func getAmbassadorMeta(ambassadorID string, clusterID string, version string, client *kates.Client) *snapshot.AmbassadorMetaInfo {
	ambMeta := &snapshot.AmbassadorMetaInfo{
		ClusterID:         clusterID,
		AmbassadorID:      ambassadorID,
		AmbassadorVersion: version,
	}
	kubeServerVer, err := client.ServerVersion()
	if err == nil {
		ambMeta.KubeVersion = kubeServerVer.GitVersion
	}
	return ambMeta
}

type SnapshotProcessor func(context.Context, SnapshotDisposition, []byte)
type SnapshotDisposition int

const (
	// Indicates the watcher is still in the booting process and the snapshot has dangling pointers.
	SnapshotIncomplete SnapshotDisposition = iota
	// Indicates that the watcher is deferring processing of the snapshot because it is considered
	// to be a product of churn.
	SnapshotDefer
	// Indicates that the watcher is dropping the snapshot because it has determined that it is
	// logically a noop.
	SnapshotDrop
	// Indicates that the snapshot is ready to be processed.
	SnapshotReady
)

type FastpathProcessor func(context.Context, *ambex.FastpathSnapshot)

// watcher is _the_ thing that watches all the different kinds of Ambassador configuration
// events that we care about. This right here is pretty much the root of everything flowing
// into Ambassador from the outside world, so:
//
// ******** READ THE FOLLOWING COMMENT CAREFULLY! ********
//
// Since this is where _all_ the different kinds of these events (K8s, Consul, filesystem,
// whatever) are brought together and examined, and where we pass judgement on whether or
// not a given event is worth reconfiguring Ambassador or not, the interactions between
// this function and other pieces of the system can be quite a bit more complex than you
// might expect. There are two really huge things you should be bearing in mind if you
// need to work on this:
//
// 1. The set of things we're watching is not static, but it must converge.
//
//    An example: you can set up a Kubernetes watch that finds a KubernetesConsulResolver
//    resource, which will then prompt a new Consul watch to happen. At present, nothing
//    that that Consul watch could find is capable of prompting a new Kubernetes watch to
//    be created. This is important: it would be fairly easy to change things such that
//    there is a feedback loop where the set of things we watch does not converge on a
//    stable set. If such a loop exists, fixing it will probably require grokking this
//    watcher function, kates.Accumulator, and maybe the reconcilers in consul.go and
//    endpoints.go as well.
//
// 2. No one source of input events can be allowed to alter the event stream for another
//    source.
//
//    An example: at one point, a bug in the watcher function resulted in the Kubernetes
//    watcher being able to decide to short-circuit a watcher iteration -- which had the
//    effect of allowing the K8s watcher to cause _Consul_ events to be ignored. That's
//    not OK. To guard against this:
//
//    A. Refrain from adding state to the watcher loop.
//    B. Try very very hard to keep logic that applies to a single source within that
//       source's specific case in the watcher's select statement.
//    C. Don't add any more select statements, so that B. above is unambiguous.
//
// 3. If you add a new channel to watch, you MUST make sure it has a way to let the loop
//    know whether it saw real changes, so that the short-circuit logic works correctly.
//    That said, recognize that the way it works now, with the state for the individual
//    watchers in the watcher() function itself is a crock, and the path forward is to
//    refactor them into classes that can separate things more cleanly.
//
// 4. If you don't fully understand everything above, _do not touch this function without
//    guidance_.
func watcherLoop(ctx context.Context, encoded *atomic.Value, k8sSrc K8sSource, queries []kates.Query,
	consulWatcher Watcher, istioCertSrc IstioCertSource, snapshotProcessor SnapshotProcessor, fastpathProcessor FastpathProcessor, ambassadorMeta *snapshot.AmbassadorMetaInfo) {
	// Ambassador has three sources of inputs: kubernetes, consul, and the filesystem. The job of
	// the watcherLoop is to read updates from all three of these sources, assemble them into a
	// single coherent configuration, and pass them along to other parts of ambassador for
	// processing.

	// The watcherLoop must decide what information is relevant to solicit from each source. This is
	// decided a bit differently for each source.
	//
	// For kubernetes the set of subscriptions is basically hardcoded to the set of resources
	// defined in interesting_types.go, this is filtered down at boot based on RBAC limitations. The
	// filtered list is used to construct the queries that are passed into this function, and that
	// set of queries remains fixed for the lifetime of the loop, i.e. the lifetime of the
	// abmassador process (unless we are testing, in which case we may run the watcherLoop more than
	// once in a single process).
	//
	// For the consul source we derive the set of resources to watch based on the configuration in
	// kubernetes, i.e. we watch the services defined in Mappings that are configured to use a
	// consul resolver. We use the ConsulResolver that a given Mapping is configured with to find
	// the datacenter to query.
	//
	// The filesystem datasource is for istio secrets. XXX fill in more

	// Each time the wathcerLoop wakes up, it assembles updates from whatever source woke it up into
	// its view of the world. It then determines if enough information has been assembled to
	// consider ambassador "booted" and if so passes the updated view along to its output (the
	// SnapshotProcessor).

	// Setup our three sources of ambassador inputs: kubernetes, consul, and the filesystem. Each of
	// these have interfaces that enable us to run with the "real" implementation or a mock
	// implementation for our Fake test harness.
	k8sWatcher := k8sSrc.Watch(ctx, queries...)
	consul := newConsul(ctx, consulWatcher)
	istioCertWatcher := istioCertSrc.Watch(ctx)
	istio := newIstioCertWatchManager(ctx, istioCertWatcher)

	// SnapshotHolder tracks all the data structures that get updated by the various sources of
	// information. It also holds the business logic that converts the data as received to a more
	// amenable form for processing. It not only serves to group these together, but it also
	// provides a mutex to protect access to the data.
	snapshots := NewSnapshotHolder(ambassadorMeta)

	// This points to notifyCh when we have updated information to send and nil when we have no new
	// information. This is deliberately nil to begin with as we have nothing to send yet.
	var out chan *SnapshotHolder
	notifyCh := make(chan *SnapshotHolder)
	go func() {
		for {
			select {
			case sh := <-notifyCh:
				sh.Notify(ctx, encoded, consul, snapshotProcessor)
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		dlog.Debugf(ctx, "WATCHER: --------")

		// XXX Hack: the istioCertWatchManager needs to reset at the start of the
		// loop, for now. A better way, I think, will be to instead track deltas in
		// ReconcileSecrets -- that way we can ditch this crap and Istio-cert changes
		// that somehow don't generate an actual change will still not trigger a
		// reconfigure.
		istio.StartLoop(ctx)

		select {
		case <-k8sWatcher.Changed():
			// Kubernetes has some changes, so we need to handle them.
			changed := snapshots.K8sUpdate(ctx, k8sWatcher, consul, fastpathProcessor)
			if !changed {
				continue
			}
			out = notifyCh
		case <-consul.changed():
			dlog.Debugf(ctx, "WATCHER: Consul fired")
			snapshots.ConsulUpdate(ctx, consul, fastpathProcessor)
			out = notifyCh
		case icertUpdate := <-istio.Changed():
			// The Istio cert has some changes, so we need to handle them.
			snapshots.IstioUpdate(ctx, istio, icertUpdate)
			out = notifyCh
		case out <- snapshots:
			out = nil
		case <-ctx.Done():
			return
		}
	}
}

// SnapshotHolder is responsible for holding
type SnapshotHolder struct {
	// This protects the entire struct.
	mutex sync.Mutex

	// The thing that knows how to validate kubernetes resources. This is always calling into the
	// kates validator even when we are being driven by the Fake harness.
	validator *resourceValidator

	// Ambassadro meta info to pass along in the snapshot.
	ambassadorMeta *snapshot.AmbassadorMetaInfo

	// These two fields represent the view of the kubernetes world and the view of the consul
	// world. This view is constructed from the raw data given to us from each respective source,
	// plus additional fields that are computed based on the raw data. These are cumulative values,
	// they always represent the entire state of their respective worlds.
	k8sSnapshot    *snapshot.KubernetesSnapshot
	consulSnapshot *watt.ConsulSnapshot
	// XXX: you would expect there to be an analogous snapshot for istio secrets, however the istio
	// source works by directly munging the k8sSnapshot.

	// The unsentDeltas field tracks the stream of deltas that have occured in between each
	// kubernetes snapshot. This is a passthrough of the full stream of deltas reported by kates
	// which is in turn a facade fo the deltas reported by client-go.
	unsentDeltas []*kates.Delta

	endpointRoutingInfo endpointRoutingInfo
	dispatcher          *gateway.Dispatcher

	// Serial number that tracks if we need to send snapshot changes or not. This is incremented
	// when a change worth sending is made, and we copy it over to snapshotNotifiedCount when the
	// change is sent.
	snapshotChangeCount    int
	snapshotChangeNotified int

	// Has the very first reconfig happened?
	firstReconfig bool
}

func NewSnapshotHolder(ambassadorMeta *snapshot.AmbassadorMetaInfo) *SnapshotHolder {
	disp := gateway.NewDispatcher()
	err := disp.Register("Gateway", gateway.Compile_Gateway)
	if err != nil {
		panic(err)
	}
	err = disp.Register("HTTPRoute", gateway.Compile_HTTPRoute)
	if err != nil {
		panic(err)
	}
	return &SnapshotHolder{
		validator:           newResourceValidator(),
		ambassadorMeta:      ambassadorMeta,
		k8sSnapshot:         NewKubernetesSnapshot(),
		consulSnapshot:      &watt.ConsulSnapshot{},
		endpointRoutingInfo: newEndpointRoutingInfo(),
		dispatcher:          disp,
		firstReconfig:       true,
	}
}

// Get the raw update from the kubernetes watcher, then redo our computed view.
func (sh *SnapshotHolder) K8sUpdate(ctx context.Context, watcher K8sWatcher, consul *consul,
	fastpathProcessor FastpathProcessor) bool {
	dbg := debug.FromContext(ctx)

	katesUpdateTimer := dbg.Timer("katesUpdate")
	parseAnnotationsTimer := dbg.Timer("parseAnnotations")
	reconcileSecretsTimer := dbg.Timer("reconcileSecrets")
	reconcileConsulTimer := dbg.Timer("reconcileConsul")

	endpointsChanged := false
	dispatcherChanged := false
	var endpoints *ambex.Endpoints
	var dispSnapshot *ecp_v2_cache.Snapshot
	changed := func() bool {
		sh.mutex.Lock()
		defer sh.mutex.Unlock()

		// We could probably get a win in some scenarios by using this filtered update thing to
		// pre-exclude based on ambassador-id.
		var deltas []*kates.Delta
		var changed bool
		katesUpdateTimer.Time(func() {
			changed = watcher.FilteredUpdate(ctx, sh.k8sSnapshot, &deltas, func(un *kates.Unstructured) bool {
				return sh.validator.isValid(ctx, un)
			})
		})

		if !changed {
			return false
		}

		// ConsulResolvers are special in that people like to be able to interpolate enviroment
		// variables in their Spec.Address field (e.g. "address: $CONSULHOST:8500" or the like),
		// so we need to handle that, but we need to also not interpolate the same thing multiple
		// times (it's probably unlikely to cause trouble, but you just know eventually it'll
		// bite us). So we'll look through deltas for changing ConsulResolvers, and then only
		// interpolate the ones that've changed.
		//
		// Also note that legacy mode supports interpolation literally anywhere in the input, but
		// let's not do that here.
		for _, delta := range deltas {
			if (delta.Kind == "ConsulResolver") && (delta.DeltaType != kates.ObjectDelete) {
				// Oh, look, a ConsulResolver changed, and it wasn't deleted. Go find the object
				// in the snapshot so we can update it.
				//
				// XXX Yes, I know, linear searches suck. We don't expect there to be many
				// ConsulResolvers, though, and we also don't expect them to change often.
				for _, resolver := range sh.k8sSnapshot.ConsulResolvers {
					if resolver.ObjectMeta.Name == delta.Name {
						// Found it! Go do the environment variable interpolation and update
						// resolver.Spec.Address in place, so that the change makes it into
						// the snapshot.
						resolver.Spec.Address = os.ExpandEnv(resolver.Spec.Address)
					}
				}
			}
		}

		parseAnnotationsTimer.Time(func() {
			parseAnnotations(ctx, sh.k8sSnapshot)
		})

		reconcileSecretsTimer.Time(func() {
			ReconcileSecrets(ctx, sh.k8sSnapshot)
		})
		reconcileConsulTimer.Time(func() {
			ReconcileConsul(ctx, consul, sh.k8sSnapshot)
		})

		sh.endpointRoutingInfo.reconcileEndpointWatches(ctx, sh.k8sSnapshot)
		// Check if the set of endpoints we are interested in has changed. If so we need to send
		// endpoint info again even if endpoints have not changed.
		if sh.endpointRoutingInfo.watchesChanged() {
			dlog.Infof(ctx, "watches changed: %v", sh.endpointRoutingInfo.endpointWatches)
			endpointsChanged = true
		}

		endpointsOnly := true
		for _, delta := range deltas {
			sh.unsentDeltas = append(sh.unsentDeltas, delta)

			if delta.Kind == "Endpoints" {
				key := fmt.Sprintf("%s:%s", delta.Namespace, delta.Name)
				if sh.endpointRoutingInfo.endpointWatches[key] || sh.dispatcher.IsWatched(delta.Namespace, delta.Name) {
					endpointsChanged = true
				}
			} else {
				endpointsOnly = false
			}

			if sh.dispatcher.IsRegistered(delta.Kind) {
				dispatcherChanged = true
				if delta.DeltaType == kates.ObjectDelete {
					sh.dispatcher.DeleteKey(delta.Kind, delta.Namespace, delta.Name)
				}
			}
		}
		if !endpointsOnly {
			sh.snapshotChangeCount += 1
		}

		if endpointsChanged || dispatcherChanged {
			endpoints = makeEndpoints(ctx, sh.k8sSnapshot, sh.consulSnapshot.Endpoints)
			for _, gwc := range sh.k8sSnapshot.GatewayClasses {
				if err := sh.dispatcher.Upsert(gwc); err != nil {
					// TODO: Should this be more severe?
					dlog.Error(ctx, err)
				}
			}
			for _, gw := range sh.k8sSnapshot.Gateways {
				if err := sh.dispatcher.Upsert(gw); err != nil {
					// TODO: Should this be more severe?
					dlog.Error(ctx, err)
				}

			}
			for _, hr := range sh.k8sSnapshot.HTTPRoutes {
				if err := sh.dispatcher.Upsert(hr); err != nil {
					// TODO: Should this be more severe?
					dlog.Error(ctx, err)
				}
			}
			_, dispSnapshot = sh.dispatcher.GetSnapshot(ctx)
		}

		return true
	}()

	if endpointsChanged || dispatcherChanged {
		fastpath := &ambex.FastpathSnapshot{
			Endpoints: endpoints,
			Snapshot:  dispSnapshot,
		}
		fastpathProcessor(ctx, fastpath)
	}

	return changed
}

func (sh *SnapshotHolder) ConsulUpdate(ctx context.Context, consul *consul, fastpathProcessor FastpathProcessor) bool {
	var endpoints *ambex.Endpoints
	var dispSnapshot *ecp_v2_cache.Snapshot
	func() {
		sh.mutex.Lock()
		defer sh.mutex.Unlock()
		consul.update(sh.consulSnapshot)
		endpoints = makeEndpoints(ctx, sh.k8sSnapshot, sh.consulSnapshot.Endpoints)
		_, dispSnapshot = sh.dispatcher.GetSnapshot(ctx)
	}()
	fastpathProcessor(ctx, &ambex.FastpathSnapshot{
		Endpoints: endpoints,
		Snapshot:  dispSnapshot,
	})
	return true
}

func (sh *SnapshotHolder) IstioUpdate(ctx context.Context, istio *istioCertWatchManager,
	icertUpdate IstioCertUpdate) bool {
	dbg := debug.FromContext(ctx)

	istioCertUpdateTimer := dbg.Timer("istioCertUpdate")
	reconcileSecretsTimer := dbg.Timer("reconcileSecrets")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	istioCertUpdateTimer.Time(func() {
		istio.Update(ctx, icertUpdate, sh.k8sSnapshot)
	})

	reconcileSecretsTimer.Time(func() {
		ReconcileSecrets(ctx, sh.k8sSnapshot)
	})

	sh.snapshotChangeCount += 1
	return true
}

func (sh *SnapshotHolder) Notify(ctx context.Context, encoded *atomic.Value, consul *consul,
	snapshotProcessor SnapshotProcessor) {
	dbg := debug.FromContext(ctx)

	notifyWebhooksTimer := dbg.Timer("notifyWebhooks")

	// If the change is solely endpoints we don't bother making a snapshot.
	var snapshotJSON []byte
	var bootstrapped bool
	changed := true

	func() {
		sh.mutex.Lock()
		defer sh.mutex.Unlock()

		if sh.snapshotChangeNotified == sh.snapshotChangeCount {
			changed = false
			return
		}

		sn := &snapshot.Snapshot{
			Kubernetes:     sh.k8sSnapshot,
			Consul:         sh.consulSnapshot,
			Invalid:        sh.validator.getInvalid(),
			Deltas:         sh.unsentDeltas,
			AmbassadorMeta: sh.ambassadorMeta,
		}

		var err error
		snapshotJSON, err = json.MarshalIndent(sn, "", "  ")
		if err != nil {
			panic(err)
		}

		bootstrapped = consul.isBootstrapped()
		if bootstrapped {
			sh.unsentDeltas = nil
			if sh.firstReconfig {
				dlog.Debugf(ctx, "WATCHER: Bootstrapped! Computing initial configuration...")
				sh.firstReconfig = false
			}
			sh.snapshotChangeNotified = sh.snapshotChangeCount
		}
	}()

	if !changed {
		return
	}

	if bootstrapped {
		// ...then stash this snapshot and fire off webhooks.
		encoded.Store(snapshotJSON)

		// Finally, use the reconfigure webhooks to let the rest of Ambassador
		// know about the new configuration.
		notifyWebhooksTimer.Time(func() {
			snapshotProcessor(ctx, SnapshotReady, snapshotJSON)
		})
	} else {
		snapshotProcessor(ctx, SnapshotIncomplete, snapshotJSON)
		return
	}
}

// The kates aka "real" version of our injected dependencies.
type k8sSource struct {
	client *kates.Client
}

func (k *k8sSource) Watch(ctx context.Context, queries ...kates.Query) K8sWatcher {
	acc := k.client.Watch(ctx, queries...)
	return acc
}

func newK8sSource(client *kates.Client) *k8sSource {
	return &k8sSource{
		client: client,
	}
}
