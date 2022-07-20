package entrypoint

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/datawire/dlib/dlog"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// endpointRoutingInfo keeps track of everything we need to know to figure out if
// endpoint routing is active.
type endpointRoutingInfo struct {
	// Map from resolver name to resolver type.
	resolverTypes   map[string]ResolverType
	module          moduleResolver
	endpointWatches map[string]bool // A set to track the subset of kubernetes endpoints we care about.
	previousWatches map[string]bool
}

type ResolverType int

const (
	KubernetesServiceResolver ResolverType = iota
	KubernetesEndpointResolver
	ConsulResolver
)

func (rt ResolverType) String() string {
	switch rt {
	case KubernetesServiceResolver:
		return "KubernetesServiceResolver"
	case KubernetesEndpointResolver:
		return "KubernetesEndpointResolver"
	case ConsulResolver:
		return "ConsulResolver"
	default:
		panic(fmt.Errorf("ResolverType.String: invalid enum value: %d", rt))
	}
}

// newEndpointRoutingInfo creates a shiny new struct to hold information about
// resolvers in use and such.
func newEndpointRoutingInfo() endpointRoutingInfo {
	return endpointRoutingInfo{
		// resolverTypes keeps track of the type of every resolver in the system.
		// It starts out empty.
		//
		// Why do we need to look at all the resolvers? Because, unless the user
		// overrides them, resolvers "endpoint" and "kubernetes-endpoint" are
		// implicitly endpoint resolvers -- but they won't show up in the snapshot.
		// So we need to track whether they've been redefined. Sigh.
		resolverTypes: make(map[string]ResolverType),
		// Track which endpoints we actually want to watch.
		endpointWatches: make(map[string]bool),
	}
}

func (eri *endpointRoutingInfo) reconcileEndpointWatches(ctx context.Context, s *snapshotTypes.KubernetesSnapshot) {
	envAmbID := GetAmbassadorID()

	// Reset our state except for the previous endpoint watches. We keep them so we can detect if
	// the set of things we are interested in has changed.
	eri.resolverTypes = map[string]ResolverType{}
	eri.module = moduleResolver{}
	eri.previousWatches = eri.endpointWatches
	eri.endpointWatches = map[string]bool{}

	// Phase one processes all the configuration stuff that Mappings depend on. Right now this
	// includes Modules and Resolvers. When we are done with Phase one we have processed enough
	// resources to correctly interpret Mappings.
	for _, list := range s.Annotations {
		for _, a := range list {
			if _, isInvalid := a.(*kates.Unstructured); isInvalid {
				continue
			}
			if GetAmbID(ctx, a).Matches(envAmbID) {
				eri.checkResourcePhase1(ctx, a, "annotation")
			}
		}
	}

	// After that, walk all the other resources. We do this with separate loops
	// for each type -- since we know a priori what type they are, there's no
	// need to test every resource, and no need to walk over things we're not
	// interested in.
	for _, m := range s.Modules {
		if m.Spec.AmbassadorID.Matches(envAmbID) {
			eri.checkModule(ctx, m, "CRD")
		}
	}

	for _, r := range s.KubernetesServiceResolvers {
		if r.Spec.AmbassadorID.Matches(envAmbID) {
			eri.saveResolver(ctx, r.GetName(), KubernetesServiceResolver, "CRD")
		}
	}

	for _, r := range s.KubernetesEndpointResolvers {
		if r.Spec.AmbassadorID.Matches(envAmbID) {
			eri.saveResolver(ctx, r.GetName(), KubernetesEndpointResolver, "CRD")
		}
	}

	for _, r := range s.ConsulResolvers {
		if r.Spec.AmbassadorID.Matches(envAmbID) {
			eri.saveResolver(ctx, r.GetName(), ConsulResolver, "CRD")
		}
	}

	// Once all THAT is done, make sure to define the default "endpoint" and
	// "kubernetes-endpoint" resolvers if they don't exist.
	for _, rName := range []string{"endpoint", "kubernetes-endpoint"} {
		_, found := eri.resolverTypes[rName]

		if !found {
			dlog.Debugf(ctx, "WATCHER: endpoint resolver %s exists by default", rName)
			eri.resolverTypes[rName] = KubernetesEndpointResolver
		}
	}

	for _, list := range s.Annotations {
		for _, a := range list {
			if _, isInvalid := a.(*kates.Unstructured); isInvalid {
				continue
			}
			if GetAmbID(ctx, a).Matches(envAmbID) {
				eri.checkResourcePhase2(ctx, a, "annotation")
			}
		}
	}

	for _, m := range s.Mappings {
		if m.Spec.AmbassadorID.Matches(envAmbID) {
			eri.checkMapping(ctx, m, "CRD")
		}
	}

	for _, t := range s.TCPMappings {
		if t.Spec.AmbassadorID.Matches(envAmbID) {
			eri.checkTCPMapping(ctx, t, "CRD")
		}
	}
}

func (eri *endpointRoutingInfo) watchesChanged() bool {
	return !reflect.DeepEqual(eri.endpointWatches, eri.previousWatches)
}

// checkResourcePhase1 processes Modules and Resolvers and calls the correct type specific handler.
func (eri *endpointRoutingInfo) checkResourcePhase1(ctx context.Context, obj kates.Object, source string) {
	switch v := obj.(type) {
	case *amb.Module:
		eri.checkModule(ctx, v, source)
	case *amb.KubernetesServiceResolver:
		eri.saveResolver(ctx, v.GetName(), KubernetesServiceResolver, "CRD")
	case *amb.KubernetesEndpointResolver:
		eri.saveResolver(ctx, v.GetName(), KubernetesEndpointResolver, "CRD")
	case *amb.ConsulResolver:
		eri.saveResolver(ctx, v.GetName(), ConsulResolver, "CRD")
	}
}

// checkResourcePhase2 processes both regular and tcp Mappings and calls the correct type specific handler.
func (eri *endpointRoutingInfo) checkResourcePhase2(ctx context.Context, obj kates.Object, source string) {
	switch v := obj.(type) {
	case *amb.Mapping:
		eri.checkMapping(ctx, v, source)
	case *amb.TCPMapping:
		eri.checkTCPMapping(ctx, v, source)
	}
}

type moduleResolver struct {
	Resolver                                   string `json:"resolver"`
	UseAmbassadorNamespaceForServiceResolution bool   `json:"use_ambassador_namespace_for_service_resolution"`
}

// checkModule parses the stuff we care about out of the ambassador Module.
func (eri *endpointRoutingInfo) checkModule(ctx context.Context, mod *amb.Module, source string) {
	if mod.GetName() != "ambassador" {
		return
	}

	mr := moduleResolver{}
	err := convert(mod.Spec.Config, &mr)

	if err != nil {
		dlog.Errorf(ctx, "error parsing ambassador module: %v", err)
		return
	}

	// The default resolver is the kubernetes service resolver.
	if mr.Resolver == "" {
		mr.Resolver = "kubernetes-service"
	}

	eri.module = mr
}

// saveResolver saves an active resolver in our resolver-type map. This is used for
// all kinds of resolvers, hence the resType parameter.
func (eri *endpointRoutingInfo) saveResolver(ctx context.Context, name string, resType ResolverType, source string) {
	// No magic here, just save the silly thing.
	eri.resolverTypes[name] = resType

	dlog.Debugf(ctx, "WATCHER: %s resolver %s is active (%s)", resType.String(), name, source)
}

// checkMapping figures out what resolver is in use for a given Mapping.
func (eri *endpointRoutingInfo) checkMapping(ctx context.Context, mapping *amb.Mapping, source string) {
	// Grab the name and the (possibly-empty) resolver.
	name := mapping.GetName()
	resolver := mapping.Spec.Resolver
	service := mapping.Spec.Service

	if resolver == "" {
		// No specified resolver means "use the default resolver".
		resolver = eri.module.Resolver
		dlog.Debugf(ctx, "WATCHER: Mapping %s uses the default resolver (%s)", name, source)
	}

	if eri.resolverTypes[resolver] == KubernetesEndpointResolver {
		svc, ns, _ := eri.module.parseService(ctx, mapping, service, mapping.GetNamespace())
		eri.endpointWatches[fmt.Sprintf("%s:%s", ns, svc)] = true
	}
}

// checkTCPMapping figures out what resolver is in use for a given TCPMapping.
func (eri *endpointRoutingInfo) checkTCPMapping(ctx context.Context, tcpmapping *amb.TCPMapping, source string) {
	// Grab the name and the (possibly-empty) resolver.
	name := tcpmapping.GetName()
	resolver := tcpmapping.Spec.Resolver
	service := tcpmapping.Spec.Service

	if resolver == "" {
		// No specified resolver means "use the default resolver".
		dlog.Debugf(ctx, "WATCHER: TCPMapping %s uses the default resolver (%s)", name, source)
		resolver = eri.module.Resolver
	}

	if eri.resolverTypes[resolver] == KubernetesEndpointResolver {
		svc, ns, _ := eri.module.parseService(ctx, tcpmapping, service, tcpmapping.GetNamespace())
		eri.endpointWatches[fmt.Sprintf("%s:%s", ns, svc)] = true
	}
}

func (m *moduleResolver) parseService(ctx context.Context, resource kates.Object, svcName, svcNamespace string) (name string, namespace string, port string) {
	// First strip off the scheme if it exists.
	parts := strings.SplitN(svcName, "://", 2)
	if len(parts) > 1 {
		svcName = parts[1]
	}

	// Next split off the port if it exists.
	parts = strings.SplitN(svcName, ":", 2)
	if len(parts) > 1 {
		_, err := strconv.Atoi(parts[1])
		if err == nil {
			port = parts[1]
			svcName = parts[0]
		}
	}

	// Next check to see if it is an IP address.
	ip := net.ParseIP(svcName)
	if ip != nil {
		name = svcName
	} else if strings.Contains(svcName, ".") {
		// If it's not an ip address but does have a dot then we split it up to find the namespace.
		parts := strings.Split(svcName, ".")
		if len(parts) > 2 {
			using := strings.Join(parts[:2], ".")
			dlog.Errorf(ctx, "mapping %s in namespace %s: ignoring extra domain parts in service, using %q",
				resource.GetName(), resource.GetNamespace(), using)
		}
		name = parts[0]
		namespace = parts[1]
		return
	} else {
		name = svcName
	}

	if m.UseAmbassadorNamespaceForServiceResolution || svcNamespace == "" {
		namespace = GetAmbassadorNamespace()
	} else {
		namespace = svcNamespace
	}

	return
}
