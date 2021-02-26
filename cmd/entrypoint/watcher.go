package entrypoint

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/datawire/ambassador/pkg/acp"
	"github.com/datawire/ambassador/pkg/debug"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/ambassador/pkg/watt"
	"github.com/datawire/dlib/dlog"
)

func watcher(ctx context.Context, ambwatch *acp.AmbassadorWatcher, encoded *atomic.Value, clusterID string, version string) {
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

	notify := func(ctx context.Context, disposition SnapshotDisposition, snap *snapshot.Snapshot) {
		if disposition == SnapshotReady {
			notifyReconfigWebhooks(ctx, ambwatch)
		}
	}

	k8sSrc := newK8sSource(client)
	consulSrc := &consulWatcher{}
	istioCertSrc := newIstioCertSource()

	watcherLoop(ctx, encoded, k8sSrc, queries, consulSrc, istioCertSrc, notify, ambassadorMeta)
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

type SnapshotProcessor func(context.Context, SnapshotDisposition, *snapshot.Snapshot)
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
	consulWatcher Watcher, istioCertSrc IstioCertSource, notify SnapshotProcessor, ambassadorMeta *snapshot.AmbassadorMetaInfo) {
	// These timers keep track of various parts of the processing of the watcher loop. They don't
	// directly impact the logic at all.
	dbg := debug.FromContext(ctx)

	katesUpdateTimer := dbg.Timer("katesUpdate")
	consulUpdateTimer := dbg.Timer("consulUpdate")
	istioCertUpdateTimer := dbg.Timer("istioCertUpdate")
	notifyWebhooksTimer := dbg.Timer("notifyWebhooks")
	parseAnnotationsTimer := dbg.Timer("parseAnnotations")
	reconcileSecretsTimer := dbg.Timer("reconcileSecrets")
	reconcileConsulTimer := dbg.Timer("reconcileConsul")

	// Synthesize the thing that knows how to watch kubernetes resources. This is always either a
	// *kates.Accumulator or the mocks supplied by the Fake harness we use for testing.
	k8sWatcher := k8sSrc.Watch(ctx, queries...)
	// Synthesize the thing that knows how to validate kubernetes resources. This is always calling
	// into the kates validator even when we are being driven by the Fake harness.
	validator := newResourceValidator()

	// The consul object manages our consul watches. The consulWatcher is either the real thing or a
	// mock from the Fake harness.
	consul := newConsul(ctx, consulWatcher) // Consul Watcher: state manager

	// Likewise for the Istio cert watcher and manager.
	istioCertWatcher := istioCertSrc.Watch(ctx)
	istio := newIstioCertWatchManager(ctx, istioCertWatcher)

	// All the variables above this line are read-only with respect to the logic in the loop. The
	// four variables defined below are updated as the loop iterates.

	// These two variables represent the view of the kubernetes world and the view of the consul
	// world. This view is constructed from the raw data given to us from each respective source,
	// plus additional fields that are computed based on the raw data. These are cumulative values,
	// they always represent the entire state of their respective worlds and are fully replaced each
	// time there is an update from their respective sources.
	k8sSnapshot := NewKubernetesSnapshot()
	consulSnapshot := &watt.ConsulSnapshot{}
	// XXX: you would expect there to be an analogous snapshot for istio secrets, however the istio
	// source works by directly munging the k8sSnapshot.

	// The unsentDeltas field tracks the stream of deltas that have occured in between each
	// kubernetes snapshot. This is a passthrough of the full stream of deltas reported by kates
	// which is in turn a facade fo the deltas reported by client-go.
	var unsentDeltas []*kates.Delta

	// Is this the very first reconfigure we've done?
	firstReconfig := true

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
			stop := katesUpdateTimer.Start()

			// We could probably get a win in some scenarios by using this filtered update thing to
			// pre-exclude based on ambassador-id.
			var deltas []*kates.Delta
			changed := k8sWatcher.FilteredUpdate(k8sSnapshot, &deltas, func(un *kates.Unstructured) bool {
				return validator.isValid(ctx, un)
			})

			stop()

			if !changed {
				continue
			}

			unsentDeltas = append(unsentDeltas, deltas...)

			parseAnnotationsTimer.Time(func() {
				parseAnnotations(k8sSnapshot)
			})

			reconcileSecretsTimer.Time(func() {
				ReconcileSecrets(ctx, k8sSnapshot)
			})
			reconcileConsulTimer.Time(func() {
				ReconcileConsul(ctx, consul, k8sSnapshot)
			})
		case <-consul.changed():
			dlog.Debugf(ctx, "WATCHER: Consul fired")

			consulUpdateTimer.Time(func() {
				consul.update(consulSnapshot)
			})
		case icertUpdate := <-istio.Changed():
			// The Istio cert has some changes, so we need to handle them.
			istioCertUpdateTimer.Time(func() {
				istio.Update(ctx, icertUpdate, k8sSnapshot)
			})

			reconcileSecretsTimer.Time(func() {
				ReconcileSecrets(ctx, k8sSnapshot)
			})

		// BEFORE ADDING A NEW CHANNEL, READ THE COMMENT AT THE TOP OF THIS
		// FUNCTION so you don't break the short-circuiting logic.

		case <-ctx.Done():
			return
		}

		sn := &snapshot.Snapshot{
			Kubernetes:     k8sSnapshot,
			Consul:         consulSnapshot,
			Invalid:        validator.getInvalid(),
			Deltas:         unsentDeltas,
			AmbassadorMeta: ambassadorMeta,
		}

		if !consul.isBootstrapped() {
			notify(ctx, SnapshotIncomplete, sn)
			continue
		}

		unsentDeltas = nil

		snapshotJSON, err := json.MarshalIndent(sn, "", "  ")
		if err != nil {
			panic(err)
		}

		// ...then stash this snapshot and fire off webhooks.
		encoded.Store(snapshotJSON)
		if firstReconfig {
			dlog.Debugf(ctx, "WATCHER: Bootstrapped! Computing initial configuration...")
			firstReconfig = false
		}

		// Finally, use the reconfigure webhooks to let the rest of Ambassador
		// know about the new configuration.
		notifyWebhooksTimer.Time(func() {
			notify(ctx, SnapshotReady, sn)
		})
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
