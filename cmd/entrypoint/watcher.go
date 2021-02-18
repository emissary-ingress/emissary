package entrypoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync/atomic"
	"time"

	"github.com/datawire/ambassador/pkg/acp"
	"github.com/datawire/ambassador/pkg/debug"
	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/ambassador/pkg/watt"
	"github.com/datawire/dlib/dlog"
)

// thingToWatch is... uh... a thing we're gonna watch. Specifically, it's a
// K8s type name and an optional field selector.
type thingToWatch struct {
	typename      string
	fieldselector string
}

func watcher(ctx context.Context, ambwatch *acp.AmbassadorWatcher, encoded *atomic.Value) {
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

	// **** SETUP DONE for the Kubernetes Watcher

	notify := func(ctx context.Context) {
		notifyReconfigWebhooks(ctx, ambwatch)
	}
	watcherLoop(ctx, encoded, &k8sSource{client}, queries, &consulWatcher{}, notify)
}

type k8sSource struct {
	client *kates.Client
}

func (k *k8sSource) Watch(ctx context.Context, queries ...kates.Query) K8sWatcher {
	acc := k.client.Watch(ctx, queries...) // K8s Watcher: state manager
	return acc
}

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
func watcherLoop(ctx context.Context, encoded *atomic.Value, k8sSrc K8sSource, queries []kates.Query, consulWatcher Watcher, notify func(context.Context)) {
	k8sWatcher := k8sSrc.Watch(ctx, queries...) // K8s Watcher: state manager
	consul := newConsul(ctx, consulWatcher)     // Consul Watcher: state manager

	// **** SETUP STARTS for the Kubernetes Watcher
	//
	// It's a lot of work to set up the Kubernetes watcher. We're not actually done
	// until we instantiate our snapshot and accumulator, well below here.

	// **** STATE for the Kubernetes Watcher and Istio Cert Watcher
	//
	// To track Kubernetes things, we need a snapshot and a kates.Accumulator.
	// The snapshot is an internally-consistent view of the stuff in our K8s
	// cluster that applies to us; the accumulator is the thing that handles all
	// the logic around the "internally consistent" part of that statement.
	//
	// The snapshot here is also where we store Istio cert state, since we want
	// Istio certs to look like K8s secrets.
	snapshot := NewKubernetesSnapshot() // K8s/Istio Cert Watchers: core state
	// XXX Temporary hack: we currently store the secrets found by the Istio-cert
	// watcher in the K8s snapshot, but this gives the Istio-cert watcher an easy
	// way to note that it saw changes. This is important because if any of the
	// watchers see changes, we can't short-circuit the reconfiguration.
	istioCertChangesPresent := false

	// **** STATE for the Consul Watcher.
	//
	// To track Consul things, we again need a (different kind of) snapshot
	// and a "consul" object. The snapshot, again, is our view of the stuff
	// in the Consul world that applies to us; the consul object doesn't so
	// much have to manage consistency as it has to manage what we tell Consul
	// we're interested in.
	consulSnapshot := &watt.ConsulSnapshot{} // Consul Watcher: core state
	// XXX Temporary hack: give the Consul watcher a trivial way to note that
	// it saw changes.  This is important because if any of the watchers see
	// changes, we can't short-circuit the reconfiguration.
	consulChangesPresent := false // Consul Watcher: core state

	// **** SETUP START for the Istio Cert Watcher
	//
	// We can watch the filesystem for Istio mTLS certificates. Here, we fire
	// up the stuff we need to do that -- specifically, we need an FSWatcher
	// to watch the filesystem, an IstioCert to manage the cert, and an update
	// channel to hear about new Istio stuff. The actual
	//
	// The actual functionality here is currently keyed off the environment
	// variable AMBASSADOR_ISTIO_SECRET_DIR, but we set the update channel
	// either way to keep the select logic below simpler. If the environment
	// variable is unset, we never instantiate the FSWatcher or IstioCert,
	// so there will never be any updates on the update channel.
	istioCertUpdateChannel := make(chan IstioCertUpdate)

	// OK. Are we supposed to watch anything?
	secretDir := os.Getenv("AMBASSADOR_ISTIO_SECRET_DIR")

	if secretDir != "" {
		// Yup, get to it. First, fire up the IstioCert, and tell it to
		// post to our update channel from above.
		icert := NewIstioCert(secretDir, "istio-certs", GetAmbassadorNamespace(), istioCertUpdateChannel)

		// Next up, fire up the FSWatcher...
		fsw, err := NewFSWatcher(ctx)

		if err != nil {
			// Really, this should never, ever happen.
			panic(err)
		}

		// ...then tell the FSWatcher to watch the Istio cert directory,
		// and give it a handler function that'll update the IstioCert
		// in turn.
		//
		// XXX This handler function is really just an impedance matcher.
		// Maybe IstioCert should just have a "HandleFSWEvent"...
		fsw.WatchDir(ctx, secretDir,
			func(ctx context.Context, event FSWEvent) {
				// Is this a deletion?
				deleted := (event.Op == FSWDelete)

				// OK. Feed this event into the IstioCert.
				icert.HandleEvent(ctx, event.Path, deleted)
			},
		)
	}
	// **** SETUP DONE for the Istio Cert Watcher

	// **** STATE (again) for the Kubernetes Watcher
	//
	// We use kates.Delta objects to indicate to the rest of Ambassador
	// what has actually changed between one snapshot and the next.
	// unsentDeltas buffers deltas across iterations if a non-bootstrapped
	// watcher short-circuits, while k8sdeltas is just the current deltas
	// for the Kubernetes watcher during a given iteration.
	var unsentDeltas []*kates.Delta // K8s Watcher: core state
	var k8sdeltas []*kates.Delta    // K8s Watcher: core state

	// **** STATE (again) for the Kubernetes Watcher
	//
	// We use kates.Unstructured objects to indicate to the rest of
	// Ambassador when we find a poorly-structured object. We also have
	// a predicate function, isValid, which we use to decide that an
	// object is invalid.
	validator := newResourceValidator()

	// We have a slew of timers to keep track of things...
	dbg := debug.FromContext(ctx)

	katesUpdateTimer := dbg.Timer("katesUpdate")
	consulUpdateTimer := dbg.Timer("consulUpdate")
	notifyWebhooksTimer := dbg.Timer("notifyWebhooks")
	parseAnnotationsTimer := dbg.Timer("parseAnnotations")
	reconcileSecretsTimer := dbg.Timer("reconcileSecrets")
	reconcileConsulTimer := dbg.Timer("reconcileConsul")
	reconcileEndpointsTimer := dbg.Timer("reconcileEndpoints")

	// **** STATE for the watcher loop itself
	//
	// Is this the very first reconfigure we've done?
	firstReconfig := true // Watcher itself: core state

	// If not, what does the previous configuration look like?
	previousSnapshotJSON := []byte{} // Watcher itself: core state

	for {
		dlog.Debugf(ctx, "WATCHER: --------")

		select {
		case <-k8sWatcher.Changed():
			dlog.Debugf(ctx, "WATCHER: K8s fired")

			// Kubernetes has some changes. There's no k8sChangesPresent variable
			// because we track K8s deltas -- if we have any when all is said and
			// done, that's how we know if K8s had real changes.

			stop := katesUpdateTimer.Start()

			// Reset k8sdeltas.
			k8sdeltas = make([]*kates.Delta, 0)

			// We could probably get a win in some scenarios by using this filtered update thing to
			// pre-exclude based on ambassador-id.
			if !k8sWatcher.FilteredUpdate(snapshot, &k8sdeltas, func(un *kates.Unstructured) bool {
				return validator.isValid(ctx, un)
			}) {
				dlog.Debugf(ctx, "WATCHER: filtered-update dropped everything")
				stop()
				continue
			}

			dlog.Debugf(ctx, "WATCHER: new deltas (%d): %s", len(k8sdeltas), deltaSummary(k8sdeltas))
			stop()

		case <-consul.changed():
			dlog.Debugf(ctx, "WATCHER: Consul fired")

			// Consul has some changes. The Consul watcher doesn't currently track
			// deltas the same way that the K8s watcher does, but OTOH anything we
			// watch in Consul is something that we know we have a reason to care
			// about. So we can go ahead and declare that we've seen real Consul
			// changes here.
			consulChangesPresent = true

			consulUpdateTimer.Time(func() {
				consul.update(consulSnapshot)
			})

		case icertUpdate := <-istioCertUpdateChannel:
			dlog.Debugf(ctx, "WATCHER: ICert fired")

			// We've seen a change in the Istio cert info on the filesystem. This is
			// kind of a hack, but let's just go ahead and say that if we see an event
			// here, it's a real change -- presumably we won't be told to watch Istio
			// certs if they aren't important.
			//
			// XXX Obviously this is a crock and we should actually track whether the
			// secret is in use.
			istioCertChangesPresent = true

			// Make a SecretRef for this new secret...
			ref := snapshotTypes.SecretRef{Name: icertUpdate.Name, Namespace: icertUpdate.Namespace}

			// ...and delete or save, as appropriate.
			if icertUpdate.Op == "delete" {
				dlog.Infof(ctx, "IstioCert: certificate %s.%s deleted", icertUpdate.Name, icertUpdate.Namespace)
				delete(snapshot.FSSecrets, ref)
			} else {
				dlog.Infof(ctx, "IstioCert: certificate %s.%s updated", icertUpdate.Name, icertUpdate.Namespace)
				snapshot.FSSecrets[ref] = icertUpdate.Secret
			}
			// Once done here, snapshot.ReconcileSecrets will handle the rest.

		// BEFORE ADDING A NEW CHANNEL, READ THE COMMENT AT THE TOP OF THIS
		// FUNCTION so you don't break the short-circuiting logic.

		case <-ctx.Done():
			return
		}

		parseAnnotationsTimer.Time(func() {
			parseAnnotations(snapshot)
		})

		reconcileSecretsTimer.Time(func() {
			ReconcileSecrets(snapshot)
		})
		reconcileConsulTimer.Time(func() {
			ReconcileConsul(ctx, consul, snapshot)
		})

		reconcileEndpointsTimer.Time(func() {
			k8sdeltas = ReconcileEndpoints(ctx, snapshot, k8sdeltas)
			dlog.Debugf(ctx, "WATCHER: filtered deltas (%d): %s", len(k8sdeltas), deltaSummary(k8sdeltas))
		})

		// Do we have any real changes from any watcher? (For the K8s watcher, we can
		// check k8sdeltas. For the others, we have booleans to look at.)
		if (len(k8sdeltas) == 0) && !consulChangesPresent && !istioCertChangesPresent {
			// Nope, no changes at all -- we can short-circuit.
			dlog.Debugf(ctx, "WATCHER: all deltas filtered out")
			continue
		}

		unsentDeltas = append(unsentDeltas, k8sdeltas...)

		if !consul.isBootstrapped() {
			continue
		}

		sn := &snapshotTypes.Snapshot{
			Kubernetes: snapshot,
			Consul:     consulSnapshot,
			Invalid:    validator.getInvalid(),
			Deltas:     unsentDeltas,
		}
		unsentDeltas = nil

		snapshotJSON, err := json.MarshalIndent(sn, "", "  ")
		if err != nil {
			panic(err)
		}

		if envbool("AMBASSADOR_WATCHER_SNAPLOG") {
			snpath := time.Now().Format("/tmp/20060102T030405-snap.json")

			err = ioutil.WriteFile(snpath, snapshotJSON, 0777)

			if err != nil {
				dlog.Errorf(ctx, "WATCHER: could not save snapshot to %s: %s", snpath, err)
			} else {
				dlog.Debugf(ctx, "WATCHER: saved snapshot as %s", snpath)
			}
		}

		// If our current snapshot is the same as our previous snapshot, skip
		// it and wait for next time.
		if bytes.Equal(snapshotJSON, previousSnapshotJSON) {
			dlog.Debugf(ctx, "WATCHER: Short-circuiting: identical snapshots")
			continue
		}

		dlog.Debugf(ctx, "WATCHER: use new snapshot (%d bytes, old is %d bytes)", len(snapshotJSON), len(previousSnapshotJSON))

		// Update previousSnapshotJSON for next time...
		previousSnapshotJSON = snapshotJSON

		// ...then stash this snapshot and fire off webhooks.
		encoded.Store(snapshotJSON)
		if firstReconfig {
			dlog.Debugf(ctx, "WATCHER: Bootstrapped! Computing initial configuration...")
			firstReconfig = false
		}

		// Finally, use the reconfigure webhooks to let the rest of Ambassador
		// know about the new configuration.
		notifyWebhooksTimer.Time(func() {
			notify(ctx)
		})
	}
}

func GetQueries(ctx context.Context, interestingTypes map[string]thingToWatch) []kates.Query {
	ns := kates.NamespaceAll
	if IsAmbassadorSingleNamespace() {
		ns = GetAmbassadorNamespace()
	}

	fs := GetAmbassadorFieldSelector()
	ls := GetAmbassadorLabelSelector()

	var queries []kates.Query
	for snapshotname, queryinfo := range interestingTypes {
		query := kates.Query{
			Namespace:     ns,
			Name:          snapshotname,
			Kind:          queryinfo.typename,
			FieldSelector: queryinfo.fieldselector,
			LabelSelector: ls,
		}
		if query.FieldSelector == "" {
			query.FieldSelector = fs
		}

		queries = append(queries, query)
		dlog.Debugf(ctx, "WATCHER: watching %#v", query)
	}

	return queries
}

func GetInterestingTypes(ctx context.Context, serverTypeList []kates.APIResource) map[string]thingToWatch {
	fs := GetAmbassadorFieldSelector()
	endpointFs := "metadata.namespace!=kube-system"
	if fs != "" {
		endpointFs += fmt.Sprintf(",%s", fs)
	}

	// We set interestingTypes to the list of types that we'd like to watch (if that type exits
	// in this cluster).
	//
	// - The key in the map is the how we'll label them in the snapshot we pass to the rest of
	//   Ambassador.
	// - The typename in the map values should be the qualified "${name}.${group}", where
	//   "${name} is lowercase+plural.
	// - If the map value doesn't set a field selector, then `fs` (above) will be used.
	//
	// Most of the interestingTypes are static, but it's completely OK to add types based
	// on runtime considerations, as we do for IngressClass and the KNative stuff.
	interestingTypes := map[string]thingToWatch{
		"Services": {typename: "services."},
		// Note that we pull secrets into "K8sSecrets" and endpoints into "K8sEndpoints".
		// ReconcileSecrets and ReconcileEndpoints pull over the ones we need into "Secrets"
		// and "Endpoints" respectively.
		"K8sSecrets":   {typename: "secrets."},
		"K8sEndpoints": {typename: "endpoints.", fieldselector: endpointFs},

		//"Ingresses": {typename: "ingresses.networking.k8s.io"}, // new in Kubernetes 1.14, deprecating ingresses.extensions
		"Ingresses": {typename: "ingresses.extensions"}, // new in Kubernetes 1.2

		"AuthServices":                {typename: "authservices.getambassador.io"},
		"ConsulResolvers":             {typename: "consulresolvers.getambassador.io"},
		"DevPortals":                  {typename: "devportals.getambassador.io"},
		"Hosts":                       {typename: "hosts.getambassador.io"},
		"KubernetesEndpointResolvers": {typename: "kubernetesendpointresolvers.getambassador.io"},
		"KubernetesServiceResolvers":  {typename: "kubernetesserviceresolvers.getambassador.io"},
		"LogServices":                 {typename: "logservices.getambassador.io"},
		"Mappings":                    {typename: "mappings.getambassador.io"},
		"Modules":                     {typename: "modules.getambassador.io"},
		"RateLimitServices":           {typename: "ratelimitservices.getambassador.io"},
		"TCPMappings":                 {typename: "tcpmappings.getambassador.io"},
		"TLSContexts":                 {typename: "tlscontexts.getambassador.io"},
		"TracingServices":             {typename: "tracingservices.getambassador.io"},
	}

	if !IsAmbassadorSingleNamespace() {
		interestingTypes["IngressClasses"] = thingToWatch{typename: "ingressclasses.networking.k8s.io"} // new in Kubernetes 1.18
	}

	if IsKnativeEnabled() {
		// Note: These keynames have a "KNative" prefix, to avoid clashing with the
		// standard "networking.k8s.io" and "extensions" types.
		interestingTypes["KNativeClusterIngresses"] = thingToWatch{typename: "clusteringresses.networking.internal.knative.dev"}
		interestingTypes["KNativeIngresses"] = thingToWatch{typename: "ingresses.networking.internal.knative.dev"}
	}

	if serverTypeList != nil {
		serverTypes := make(map[string]kates.APIResource, len(serverTypeList))
		for _, typeinfo := range serverTypeList {
			serverTypes[typeinfo.Name+"."+typeinfo.Group] = typeinfo
		}

		for k, queryinfo := range interestingTypes {
			if _, haveType := serverTypes[queryinfo.typename]; !haveType {
				dlog.Infof(ctx, "Warning, unable to watch %s, unknown kind.", queryinfo.typename)
				delete(interestingTypes, k)
			}
		}
	}

	return interestingTypes
}
