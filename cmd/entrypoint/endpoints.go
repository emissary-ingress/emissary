package entrypoint

import (
	"context"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// endpointRoutingInfo keeps track of everything we need to know to figure out if
// endpoint routing is active.
type endpointRoutingInfo struct {
	resolverTypes        map[string]string // What kinds of resolvers are defined?
	resolversInUse       map[string]bool   // What are the names of the resolvers actually in use?
	defaultResolverName  string            // What is the default resolver in use?
	defaultResolverInUse bool              // Is the default resolver actually in use?
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
		resolverTypes: make(map[string]string),

		// resolversInUse keeps track of all resolvers actually referenced by any
		// Mapping or TCPMapping. It also starts out empty, and we cheat and use a
		// map[string]bool as a set type here.
		resolversInUse: make(map[string]bool),

		// defaultResolverName is the default resolver defined in the Ambassador
		// Module. Unless overridden, it's "kubernetes-service" to use the built-in
		// KubernetesServiceResolver (see the `resolve_resolver` method in `ir.py`).
		defaultResolverName: "kubernetes-service",

		// defaultResolverInUse keeps track of whether any mapping actually uses the
		// default resolver.
		//
		// XXX Why do we need this? It's because we may very well see Mappings before
		// we see the Ambassador Module, so we might not know whether or not the default
		// resolver has been overridden when we see a mapping.
		defaultResolverInUse: false,
	}
}

func (eri *endpointRoutingInfo) isRoutingActive(ctx context.Context, s *snapshotTypes.KubernetesSnapshot) bool {
	// Here's what we have to do:
	//
	// 1. Are there any KubernetesEndpointResolvers in the system?
	// 2. Do any Mappings or TCPMappings reference them?
	//
	// This should be relatively easy, but annotations make it annoying. Also,
	// we need to find the Ambassador Module, because it can specify a default
	// resolver.
	//
	// So. Start by walking all the annotations, and whatever they are, check
	// them out.
	for _, a := range s.Annotations {
		if include(GetAmbId(a)) {
			eri.checkResource(ctx, a, "annotation")
		}
	}

	// After that, walk all the other resources. We do this with separate loops
	// for each type -- since we know a priori what type they are, there's no
	// need to test every resource, and no need to walk over things we're not
	// interested in.
	for _, m := range s.Modules {
		if include(m.Spec.AmbassadorID) {
			eri.checkModule(ctx, m, "CRD")
		}
	}

	for _, r := range s.KubernetesServiceResolvers {
		if include(r.Spec.AmbassadorID) {
			eri.saveResolver(ctx, r.GetName(), "service", "CRD")
		}
	}

	for _, r := range s.KubernetesEndpointResolvers {
		if include(r.Spec.AmbassadorID) {
			eri.saveResolver(ctx, r.GetName(), "endpoint", "CRD")
		}
	}

	for _, r := range s.ConsulResolvers {
		if include(r.Spec.AmbassadorID) {
			eri.saveResolver(ctx, r.GetName(), "consul", "CRD")
		}
	}

	for _, m := range s.Mappings {
		if include(m.Spec.AmbassadorID) {
			eri.checkMapping(ctx, m, "CRD")
		}
	}

	for _, t := range s.TCPMappings {
		if include(t.Spec.AmbassadorID) {
			eri.checkTCPMapping(ctx, t, "CRD")
		}
	}

	// Once all THAT is done, make sure to define the default "endpoint" and
	// "kubernetes-endpoint" resolvers if they don't exist.
	for _, rName := range []string{"endpoint", "kubernetes-endpoint"} {
		_, found := eri.resolverTypes[rName]

		if !found {
			dlog.Debugf(ctx, "WATCHER: endpoint resolver %s exists by default", rName)
			eri.resolverTypes[rName] = "endpoint"
		}
	}

	// Once all THAT is done, see if any resolvers in use are endpoint resolvers.
	// Check the default first.
	if eri.defaultResolverInUse {
		rType, found := eri.resolverTypes[eri.defaultResolverName]

		if found {
			dlog.Debugf(ctx, "WATCHER: default resolver %s is an active %s resolver", eri.defaultResolverName, rType)

			if rType == "endpoint" {
				// Yup, it's an endpoint resolver. That's enough to know that endpoint
				// routing is active, so short-circuit here.
				return true
			}
		}
	}

	// Either the default resolver isn't in use, or it isn't an endpoint resolver.
	// In either case, we need to check the other resolvers in use.
	for rName := range eri.resolversInUse {
		rType, found := eri.resolverTypes[rName]

		if found {
			dlog.Debugf(ctx, "WATCHER: referenced resolver %s is an active %s resolver", rName, rType)

			if rType == "endpoint" {
				// Again, just getting one is sufficient, so we can short-circuit here.
				return true
			}
		}
	}

	// If we get this far, no endpoint resolvers are in use, so endpoint routing
	// isn't active.
	dlog.Debugf(ctx, "WATCHER: no endpoint resolvers in use")
	return false
}

// checkResource figures out if a resource (from an annotation) is something we're
// interested in, and calls the correct handler if so.
func (eri *endpointRoutingInfo) checkResource(ctx context.Context, obj kates.Object, source string) {
	mod, ok := obj.(*amb.Module)
	if ok {
		eri.checkModule(ctx, mod, source)
		return
	}

	sr, ok := obj.(*amb.KubernetesServiceResolver)
	if ok {
		eri.saveResolver(ctx, sr.GetName(), "service", "CRD")
		return
	}

	epr, ok := obj.(*amb.KubernetesEndpointResolver)
	if ok {
		eri.saveResolver(ctx, epr.GetName(), "endpoint", "CRD")
		return
	}

	cr, ok := obj.(*amb.ConsulResolver)
	if ok {
		eri.saveResolver(ctx, cr.GetName(), "consul", "CRD")
		return
	}

	mapping, ok := obj.(*amb.Mapping)
	if ok {
		eri.checkMapping(ctx, mapping, source)
		return
	}

	tcpmapping, ok := obj.(*amb.TCPMapping)
	if ok {
		eri.checkTCPMapping(ctx, tcpmapping, source)
		return
	}
}

type moduleResolver struct {
	Resolver string `json:"resolver"`
}

// checkModule looks at a Module to see if it has resolver info. This can only happen
// for the ambassador Module.
func (eri *endpointRoutingInfo) checkModule(ctx context.Context, mod *amb.Module, source string) {
	if mod.GetName() != "ambassador" {
		return
	}

	// Yup, OK. Grab its resolver.
	mr := moduleResolver{}
	err := convert(mod.Spec.Config, &mr)

	if err != nil {
		dlog.Errorf(ctx, "error extracting resolver from module: %v", err)
		return
	}

	if mr.Resolver != "" {
		dlog.Debugf(ctx, "WATCHER: amod (%s) resolver %s", source, mr.Resolver)
		eri.defaultResolverName = mr.Resolver
	}
}

// saveResolver saves an active resolver in our resolver-type map. This is used for
// all kinds of resolvers, hence the resType parameter.
func (eri *endpointRoutingInfo) saveResolver(ctx context.Context, name string, resType string, source string) {
	// No magic here, just save the silly thing.
	eri.resolverTypes[name] = resType

	dlog.Debugf(ctx, "WATCHER: %s resolver %s is active (%s)", resType, name, source)
}

// checkMapping figures out what resolver is in use for a given Mapping.
func (eri *endpointRoutingInfo) checkMapping(ctx context.Context, mapping *amb.Mapping, source string) {
	// Grab the name and the (possibly-empty) resolver.
	name := mapping.GetName()
	resolver := mapping.Spec.Resolver

	if resolver == "" {
		// No specified resolver means "use the default resolver". We don't necessarily know
		// what the default resolver will be yet, so just note that "the default" is in use.
		dlog.Debugf(ctx, "WATCHER: Mapping %s uses the default resolver (%s)", name, source)
		eri.defaultResolverInUse = true
		return
	}

	// Given an actual resolver name, just mark that specific resolver as in use.
	dlog.Debugf(ctx, "WATCHER: Mapping %s uses resolver %s (%s)", name, resolver, source)
	eri.resolversInUse[resolver] = true
}

// checkTCPMapping figures out what resolver is in use for a given TCPMapping.
func (eri *endpointRoutingInfo) checkTCPMapping(ctx context.Context, tcpmapping *amb.TCPMapping, source string) {
	// Grab the name and the (possibly-empty) resolver.
	name := tcpmapping.GetName()
	resolver := tcpmapping.Spec.Resolver

	if resolver == "" {
		// No specified resolver means "use the default resolver". We don't necessarily know
		// what the default resolver will be yet, so just note that "the default" is in use.
		dlog.Debugf(ctx, "WATCHER: TCPMapping %s uses the default resolver (%s)", name, source)
		eri.defaultResolverInUse = true
		return
	}

	// Given an actual resolver name, just mark that specific resolver as in use.
	dlog.Debugf(ctx, "WATCHER: TCPMapping %s uses resolver %s (%s)", name, resolver, source)
	eri.resolversInUse[resolver] = true
}
