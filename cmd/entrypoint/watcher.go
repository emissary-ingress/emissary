package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync/atomic"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/watt"
)

func watcher(ctx context.Context, encoded *atomic.Value) {
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

	client, err := kates.NewClient(kates.ClientOptions{})
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

	var CRDs []*kates.CustomResourceDefinition
	err = client.List(ctx, kates.Query{Kind: "CustomResourceDefinition"}, &CRDs)
	if err != nil {
		panic(err)
	}

	crdNames := map[string]bool{}
	for _, crd := range CRDs {
		crdNames[crd.Spec.Names.Kind] = true
		crdNames[crd.GetName()] = true
	}

	for _, name := range []string{"Ingress", "Service", "Secret", "Endpoints"} {
		crdNames[name] = true
	}

	allQueries := []kates.Query{
		//kates.Query{Name: "IngressClasses", Kind: "IngressClass"}, // XXX: what is an ingress class?
		{Namespace: ns, Name: "Ingresses", Kind: "Ingress",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "Services", Kind: "Service",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "AllSecrets", Kind: "Secret",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "Hosts", Kind: "Host",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "Mappings", Kind: "Mapping",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "TCPMappings", Kind: "TCPMapping",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "TLSContexts", Kind: "TLSContext",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "Modules", Kind: "Module",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "RateLimitServices", Kind: "RateLimitService",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "AuthServices", Kind: "AuthService",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "LogServices", Kind: "LogService",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "TracingServices", Kind: "TracingService",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "ConsulResolvers", Kind: "ConsulResolver",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "KubernetesEndpointResolvers", Kind: "KubernetesEndpointResolver",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "KubernetesServiceResolvers", Kind: "KubernetesServiceResolver",
			FieldSelector: fs, LabelSelector: ls},
		{Namespace: ns, Name: "Endpoints", Kind: "Endpoints", FieldSelector: endpointFs, LabelSelector: ls},
	}

	if IsKnativeEnabled() {
		allQueries = append(allQueries,
			kates.Query{Namespace: ns, Name: "KNativeClusterIngresses",
				Kind: "clusteringresses.networking.internal.knative.dev", FieldSelector: fs, LabelSelector: ls},
			kates.Query{Namespace: ns, Name: "KNativeIngresses", Kind: "ingresses.networking.internal.knative.dev",
				FieldSelector: fs, LabelSelector: ls})
	}

	var queries []kates.Query

	for _, q := range allQueries {
		if crdNames[q.Kind] {
			queries = append(queries, q)
		} else {
			log.Printf("Warning, unable to watch %s, unknown kind.", q.Kind)
		}
	}

	snapshot := &AmbassadorInputs{}
	acc := client.Watch(ctx, queries...)

	consulSnapshot := &watt.ConsulSnapshot{}
	consul := newConsul(ctx, &consulWatcher{})

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

	firstReconfig := true

	for {
		select {
		case <-acc.Changed():
			var deltas []*kates.Delta
			// We could probably get a win in some scenarios by using this filtered update thing to
			// pre-exclude based on ambassador-id.
			if !acc.FilteredUpdate(snapshot, &deltas, isValid) {
				continue
			}
			unsentDeltas = append(unsentDeltas, deltas...)
		case <-consul.changed():
			consul.update(consulSnapshot)
		case <-ctx.Done():
			return
		}

		snapshot.parseAnnotations()

		snapshot.ReconcileSecrets()
		snapshot.ReconcileConsul(ctx, consul)

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
		notifyReconfigWebhooks(ctx)

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
