package entrypoint

import (
	"context"
	"fmt"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
)

// thingToWatch is... uh... a thing we're gonna watch. Specifically, it's a
// K8s type name and an optional field selector.
type thingToWatch struct {
	typename      string
	fieldselector string
}

// GetQueries takes a set of interesting types, and returns a set of kates.Query to watch
// for them.
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

// GetInterestingTypes takes a list of available server types, and returns the types we think
// are interesting to watch.
func GetInterestingTypes(ctx context.Context, serverTypeList []kates.APIResource) map[string]thingToWatch {
	fs := GetAmbassadorFieldSelector()
	endpointFs := "metadata.namespace!=kube-system"
	if fs != "" {
		endpointFs += fmt.Sprintf(",%s", fs)
	}

	serverTypes := make(map[string]kates.APIResource, len(serverTypeList))
	if serverTypeList != nil {
		for _, typeinfo := range serverTypeList {
			serverTypes[typeinfo.Name+"."+typeinfo.Group] = typeinfo
		}
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
		"Services":  {typename: "services."},
		"Endpoints": {typename: "endpoints.", fieldselector: endpointFs},
		// Note that we pull secrets into "K8sSecrets".
		// ReconcileSecrets will pull over the ones we need into "Secrets"
		// and "Endpoints" respectively.
		"K8sSecrets": {typename: "secrets."},

		"AuthServices":                {typename: "authservices.getambassador.io"},
		"ConsulResolvers":             {typename: "consulresolvers.getambassador.io"},
		"DevPortals":                  {typename: "devportals.getambassador.io"},
		"Hosts":                       {typename: "hosts.getambassador.io"},
		"KubernetesEndpointResolvers": {typename: "kubernetesendpointresolvers.getambassador.io"},
		"KubernetesServiceResolvers":  {typename: "kubernetesserviceresolvers.getambassador.io"},
		"Listeners":                   {typename: "listeners.getambassador.io"},
		"LogServices":                 {typename: "logservices.getambassador.io"},
		"Mappings":                    {typename: "mappings.getambassador.io"},
		"Modules":                     {typename: "modules.getambassador.io"},
		"RateLimitServices":           {typename: "ratelimitservices.getambassador.io"},
		"TCPMappings":                 {typename: "tcpmappings.getambassador.io"},
		"TLSContexts":                 {typename: "tlscontexts.getambassador.io"},
		"TracingServices":             {typename: "tracingservices.getambassador.io"},

		// Gateway API resources
		"GatewayClasses": {typename: "gatewayclasses.networking.x-k8s.io"},
		"Gateways":       {typename: "gateways.networking.x-k8s.io"},
		"HTTPRoutes":     {typename: "httproutes.networking.x-k8s.io"},
	}

	_, haveOldIngress := serverTypes["ingresses.extensions"]        // First appeared in Kubernetes 1.2, gone in Kubernetes 1.22.
	_, haveNewIngress := serverTypes["ingresses.networking.k8s.io"] // New in Kubernetes 1.14, deprecating ingresses.extensions.
	if haveOldIngress && !haveNewIngress {
		interestingTypes["Ingresses"] = thingToWatch{typename: "ingresses.extensions"}
	} else {
		// Add this even if !haveNewIngress, so that the warning below triggers for it.
		interestingTypes["Ingresses"] = thingToWatch{typename: "ingresses.networking.k8s.io"}
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

	for k, queryinfo := range interestingTypes {
		if _, haveType := serverTypes[queryinfo.typename]; !haveType {
			dlog.Warnf(ctx, "Warning, unable to watch %s, unknown kind.", queryinfo.typename)
			delete(interestingTypes, k)
		}
	}

	return interestingTypes
}
