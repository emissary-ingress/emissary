package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync/atomic"

	"github.com/datawire/ambassador/pkg/acp"
	"github.com/datawire/ambassador/pkg/debug"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/watt"
	"github.com/datawire/dlib/dlog"
)

func watcher(ctx context.Context, ambwatch *acp.AmbassadorWatcher, encoded *atomic.Value) {
	crdYAML, err := ioutil.ReadFile(findCRDFilename())
	if err != nil {
		panic(err)
	}

	crdObjs, err := kates.ParseManifests(string(crdYAML))
	if err != nil {
		panic(err)
	}
	validator, err := kates.NewValidator(nil, crdObjs)
	if err != nil {
		panic(err)
	}

	client, err := kates.NewClient(kates.ClientConfig{})
	if err != nil {
		panic(err)
	}

	ns := kates.NamespaceAll
	if IsAmbassadorSingleNamespace() {
		ns = GetAmbassadorNamespace()
	}

	fs := GetAmbassadorFieldSelector()
	ls := GetAmbassadorLabelSelector()

	endpointFs := "metadata.namespace!=kube-system"
	if fs != "" {
		endpointFs += fmt.Sprintf(",%s", fs)
	}

	serverTypeList, err := client.ServerPreferredResources()
	if err != nil {
		// It's possible that an error prevented listing some apigroups, but not all; so
		// process the output even if there is an error.
		log.Printf("Warning, unable to list api-resources: %v", err)
	}
	serverTypes := make(map[string]kates.APIResource, len(serverTypeList))
	for _, typeinfo := range serverTypeList {
		serverTypes[typeinfo.Name+"."+typeinfo.Group] = typeinfo
	}

	// We set interestingTypes to the list of types that we'd like to watch (if that type exits
	// in this cluster).
	//
	// - The key in the map is the how we'll label them in the snapshot we pass to the rest of
	//   Ambassador.
	// - The typename in the map values should be the qualified "${name}.${group}", where
	//   "${name} is lowercase+plural.
	// - If the map value doesn't set a field selector, then `fs` (above) will be used.
	interestingTypes := map[string]struct {
		typename      string
		fieldselector string
	}{
		"Services":   {typename: "services."},
		"K8sSecrets": {typename: "secrets."}, // Note: "K8sSecrets" is not the "obvious" keyname
		"Endpoints":  {typename: "endpoints.", fieldselector: endpointFs},

		"IngressClasses": {typename: "ingressclasses.networking.k8s.io"}, // new in Kubernetes 1.18
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
	if IsKnativeEnabled() {
		interestingKNativeTypes := map[string]struct {
			typename      string
			fieldselector string
		}{
			// Note: These keynames have a "KNative" prefix, to avoid clashing with the
			// standard "networking.k8s.io" and "extensions" types.
			"KNativeClusterIngresses": {typename: "clusteringresses.networking.internal.knative.dev"},
			"KNativeIngresses":        {typename: "ingresses.networking.internal.knative.dev"},
		}
		for k, v := range interestingKNativeTypes {
			interestingTypes[k] = v
		}
	}

	var queries []kates.Query
	for snapshotname, queryinfo := range interestingTypes {
		if _, haveType := serverTypes[queryinfo.typename]; !haveType {
			log.Printf("Warning, unable to watch %s, unknown kind.", queryinfo.typename)
			continue
		}

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
	}

	snapshot := NewAmbassadorInputs()
	acc := client.Watch(ctx, queries...)

	consulSnapshot := &watt.ConsulSnapshot{}
	consul := newConsul(ctx, &consulWatcher{})

	// Time to fire up the stuff we need to watch the filesystem for Istio
	// certs -- specifically, we need an FSWatcher to watch the filesystem,
	// an IstioCert to manage the cert, and an update channel to hear about
	// new Istio stuff.
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

	var unsentDeltas []*kates.Delta

	invalid := map[string]*kates.Unstructured{}
	isValid := func(un *kates.Unstructured) bool {
		key := string(un.GetUID())
		err := validator.Validate(ctx, un)
		if err != nil {
			copy := un.DeepCopy()
			copy.Object["errors"] = err.Error()
			invalid[key] = copy
			return false
		} else {
			delete(invalid, key)
			return true
		}
	}

	dbg := debug.FromContext(ctx)

	katesUpdateTimer := dbg.Timer("katesUpdate")
	consulUpdateTimer := dbg.Timer("consulUpdate")
	notifyWebhooksTimer := dbg.Timer("notifyWebhooks")
	parseAnnotationsTimer := dbg.Timer("parseAnnotations")
	reconcileSecretsTimer := dbg.Timer("reconcileSecrets")
	reconcileConsulTimer := dbg.Timer("reconcileConsul")

	firstReconfig := true

	for {
		select {
		case <-acc.Changed():
			stop := katesUpdateTimer.Start()
			var deltas []*kates.Delta
			// We could probably get a win in some scenarios by using this filtered update thing to
			// pre-exclude based on ambassador-id.
			if !acc.FilteredUpdate(snapshot, &deltas, isValid) {
				stop()
				continue
			}
			unsentDeltas = append(unsentDeltas, deltas...)
			stop()
		case <-consul.changed():
			consulUpdateTimer.Time(func() {
				consul.update(consulSnapshot)
			})
		case icertUpdate := <-istioCertUpdateChannel:
			// Make a SecretRef for this new secret...
			ref := SecretRef{Name: icertUpdate.Name, Namespace: icertUpdate.Namespace}

			// ...and delete or save, as appropriate.
			if icertUpdate.Op == "delete" {
				dlog.Infof(ctx, "IstioCert: certificate %s.%s deleted", icertUpdate.Name, icertUpdate.Namespace)
				delete(snapshot.FSSecrets, ref)
			} else {
				dlog.Infof(ctx, "IstioCert: certificate %s.%s updated", icertUpdate.Name, icertUpdate.Namespace)
				snapshot.FSSecrets[ref] = icertUpdate.Secret
			}
			// Once done here, snapshot.ReconcileSecrets will handle the rest.
		case <-ctx.Done():
			return
		}

		parseAnnotationsTimer.Time(snapshot.parseAnnotations)

		reconcileSecretsTimer.Time(snapshot.ReconcileSecrets)
		reconcileConsulTimer.Time(func() {
			snapshot.ReconcileConsul(ctx, consul)
		})

		if !consul.isBootstrapped() {
			continue
		}

		var invalidSlice []*kates.Unstructured
		for _, inv := range invalid {
			invalidSlice = append(invalidSlice, inv)
		}

		sn := &Snapshot{
			Kubernetes: snapshot,
			Consul:     consulSnapshot,
			Invalid:    invalidSlice,
			Deltas:     unsentDeltas,
		}
		unsentDeltas = nil

		bytes, err := json.MarshalIndent(sn, "", "  ")
		if err != nil {
			panic(err)
		}
		encoded.Store(bytes)
		if firstReconfig {
			log.Println("Bootstrapped! Computing initial configuration...")
			firstReconfig = false
		}
		notifyWebhooksTimer.Time(func() {
			notifyReconfigWebhooks(ctx, ambwatch)
		})

		// we really only need to be incremental for a subset of things:
		//  - Mappings & Endpoints are the biggies
		//  - TLSContext are probably next

		// for Endpoints, we can probably figure out a way to wire things up where we bypass  python entirely:
		//   - maybe have the python put in a placeholder that the go code fills in
		//   - maybe use EDS to pump the endpoint data directly to the cluster
	}
}

func findCRDFilename() string {
	searchPath := []string{
		"/opt/ambassador/etc/crds.yaml",
		"docs/yaml/ambassador/ambassador-crds.yaml",
	}

	for _, candidate := range searchPath {
		if fileExists(candidate) {
			return candidate
		}
	}

	panic(fmt.Sprintf("couldn't find CRDs at any of the following locations: %s", strings.Join(searchPath, ", ")))
}
