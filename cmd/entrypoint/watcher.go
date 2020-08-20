package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	snapshot := &AmbassadorInputs{}
	acc := client.Watch(ctx,
		kates.Query{Namespace: ns, Name: "CRDs", Kind: "CustomResourceDefinition"},
		// kates.Query{Name: "IngressClasses", Kind: "IngressClass"}, // I guess k3s doesn't have these.?
		kates.Query{Namespace: ns, Name: "Ingresses", Kind: "Ingress",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "Services", Kind: "Service",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "AllSecrets", Kind: "Secret",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "Hosts", Kind: "Host",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "Mappings", Kind: "Mapping",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "TCPMappings", Kind: "TCPMapping",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "TLSContexts", Kind: "TLSContext",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "Modules", Kind: "Module",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "RateLimitServices", Kind: "RateLimitService",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "AuthServices", Kind: "AuthService",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "LogServices", Kind: "LogService",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "TracingServices", Kind: "TracingService",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "ConsulResolvers", Kind: "ConsulResolver",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "KubernetesEndpointResolvers", Kind: "KubernetesEndpointResolver",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "KubernetesServiceResolvers", Kind: "KubernetesServiceResolver",
			FieldSelector: fs, LabelSelector: ls},
		kates.Query{Namespace: ns, Name: "Endpoints", Kind: "Endpoints", FieldSelector: endpointFs, LabelSelector: ls},
	)

	consulSnapshot := watt.ConsulSnapshot{}
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

	for {
		select {
		case <-acc.Changed():
			var deltas []*kates.Delta
			if !acc.FilteredUpdate(snapshot, &deltas, isValid) {
				continue
			}
			unsentDeltas = append(unsentDeltas, deltas...)
		case <-consul.changed():
			consul.update(&consulSnapshot)
		case <-ctx.Done():
			return
		}

		// entrypoint based feature flag to swap out watt vs watt2

		snapshot.ReconcileSecrets(ctx, client) // straighforward plumbing
		snapshot.ReconcileConsul(ctx, consul)  // more risk, we haven't touched this code in ages, and we don't have good tests

		if !consul.isBootstrapped() {
			continue
		}

		sn := map[string]interface{}{"Kubernetes": snapshot}
		sn["Consul"] = consulSnapshot
		sn["Deltas"] = unsentDeltas
		unsentDeltas = nil

		var invalidSlice []*kates.Unstructured
		for _, inv := range invalid {
			invalidSlice = append(invalidSlice, inv)
		}
		sn["Invalid"] = invalidSlice

		//Errors map[string][]Error `json:",omitempty"`

		bytes, err := json.MarshalIndent(sn, "", "  ")
		if err != nil {
			panic(err)
		}
		encoded.Store(bytes)
		notifyReconfigWebhooks(ctx)

		// invokePython is straighforward plumbing
		// ComputeDirrrty() is optional (we still get watch hook win without it)
		// ComputeDirrrty() starting out as advisory is a low-risk way to move forward
		//invokePython(snapshot.ComputeDirrrty()) // ComputeDirrrty() should always be pronounced the way Cardi B would
		/*resources =
		envoySnapshot.update(resources)
		envoySnapshot.save()*/

		// we really only need to be incremental for a subset of things:
		//  - Mappings & Endpoints are the biggies
		//  - TLSContext are probably next

		// for Endpoints, we can probably figure out a way to wire things up where we bypass  python entirely:
		//   - maybe have the python put in a placeholder that the go code fills in
		//   - maybe use EDS to pump the endpoint data directly to the cluster

		// we could make this skeleton subsume large parts of entrypoint.sh (e.g. ambex)

		//log.Println(snapshot.Render())
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
