package entrypoint

import (
	"context"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// ReconcileEndpoints checks to see if we're actually interested in endpoints, and skips
// them if not.
func ReconcileEndpoints(ctx context.Context, s *snapshotTypes.KubernetesSnapshot, deltas []*kates.Delta) []*kates.Delta {
	// We're interested in Endpoints if we have Mappings using a KubernetesEndpointResolver,
	// or if AMBASSADOR_FORCE_ENDPOINTS is set.
	//
	// Do the easy check first.
	takeEndpoints := ForceEndpoints()

	if !takeEndpoints {
		// The easy check didn't indicate that we have to do endpoints, so we'll have
		// to do the harder check of actually checking resolvers.
		eri := newEndpointRoutingInfo()
		takeEndpoints = eri.isRoutingActive(ctx, s)
	}

	if takeEndpoints {
		// Endpoint routing is active, so we need to take all the endpoints.
		deltas = acceptEndpoints(ctx, s, deltas)
	} else {
		// Endpoint routing is not active; so we need to _discard_ all the endpoints
		// from this snapshot.
		deltas = discardEndpoints(ctx, s, deltas)
	}

	return deltas
}

// acceptEndpoints makes sure that s.Endpoints is up to date with s.K8sEndpoints, and
// returns an appropriate set of deltas.
func acceptEndpoints(ctx context.Context, s *snapshotTypes.KubernetesSnapshot, deltas []*kates.Delta) []*kates.Delta {
	dlog.Debug(ctx, "WATCHER: accept Endpoints")

	// If s.Endpoints is currently empty, then we've either just turned on endpoint
	// routing after it was off, or we've just gotten some new endpoints after we had
	// absolutely none at all. In either case, make sure that all the endpoints we're
	// accepting have kates.Delta objects to go with them, so that everything downstream
	// sees a consistent view of the world.
	needAdditions := (len(s.Endpoints) == 0)

	// Shallow copy from s.K8sEndpoints (which our K8s watch is keeping up to date with
	// the cluster) into s.Endpoints (which is what will actuall get sent to the rest of
	// Ambassador). This can be a shallow copy, but it needs to be a copy: we do not want
	// to mess with s.K8sEndpoints, lest we confuse the watcher or something.
	s.Endpoints = make([]*kates.Endpoints, len(s.K8sEndpoints))
	copy(s.Endpoints, s.K8sEndpoints)

	if needAdditions {
		dlog.Debug(ctx, "WATCHER: reinitialize Endpoints deltas")

		// This is the first round of Endpoints after we had none, so make sure we have
		// an ObjectAdd delta for every Endpoint we're handing out.
		newDeltas := synthesizeEndpointDeltas(ctx, s.K8sEndpoints, kates.ObjectAdd)

		// We can't just stuff all those deltas into our existing set, or we might have
		// duplicates -- instead, do the easy thing and just strip out any existing
		// Endpoints deltas (it makes life easier) before appending our new deltas.
		deltas = dropEndpointDeltas(ctx, deltas)
		deltas = append(deltas, newDeltas...)
	}

	return deltas
}

// discardEndpoints makes sure that s.Endpoints is _empty_, and returns an appropriate
// set of deltas.
func discardEndpoints(ctx context.Context, s *snapshotTypes.KubernetesSnapshot, deltas []*kates.Delta) []*kates.Delta {
	dlog.Debug(ctx, "WATCHER: skip Endpoints")

	// If s.Endpoints is currently not empty, then we've just had endpoint routing turned
	// off after it was on, and we need to make sure that all the endpoints we're tossing
	// have kates.Delta objects to go with them, so that everything downstream sees a
	// consistent view of the world.
	needDeletions := (len(s.Endpoints) > 0)

	// Easy bit first: empty out s.Endpoints. Note well that we don't touch s.K8sEndpoints!
	// It's what we use to stay in sync with the cluster's actual Endpoints.
	s.Endpoints = make([]*kates.Endpoints, 0)

	// Once that's done, toss any existing Endpoints Deltas...
	deltas = dropEndpointDeltas(ctx, deltas)

	// ...then synthesize deletions if we need to. (Yes, this might be synthesizing
	// replacements for deltas we just throwed away. That's OK, it keeps the logic
	// simpler.)
	if needDeletions {
		dlog.Debug(ctx, "WATCHER: reinitialize Endpoints deltas")

		newDeltas := synthesizeEndpointDeltas(ctx, s.K8sEndpoints, kates.ObjectDelete)
		deltas = append(deltas, newDeltas...)
	}

	return deltas
}

// dropEndpointsDeltas strips out Endpoints deltas from a list of kates.Delta,
// returning the (possibly reduced) list.
func dropEndpointDeltas(ctx context.Context, deltas []*kates.Delta) []*kates.Delta {
	// We delete by shuffling over the ones we want to keep, then returning the
	// slice of whatever is left over. This has the weird seeming-side-effect that
	// if there are no Endpoints deltas, we end up doing deltas[i] = deltas[i] for
	// every delta... but that's really cheap, and really it's not even a side
	// effect unless we've landed in a situation where we're running this over the
	// same list in more than one thread at a time. Which we won't.

	i := 0

	for _, delta := range deltas {
		if delta.GetObjectKind().GroupVersionKind().Kind != "Endpoints" {
			deltas[i] = delta
			i++
		}
	}

	// Return the (possibly-shortened) slice that we actually want.
	return deltas[:i]
}

// synthesizeEndpointDeltas creates a list of kates.Delta for every Endpoint
// passed in.
func synthesizeEndpointDeltas(ctx context.Context, endpoints []*kates.Endpoints, deltaType kates.DeltaType) []*kates.Delta {
	deltas := make([]*kates.Delta, 0, len(endpoints))

	for _, endpoint := range endpoints {
		// A kates.Delta isn't the whole object, just a subset.
		//
		// XXX This should really be a call to kates.NewDelta, but that wants an
		// Unstructured instead of an Endpoints.
		// deltas = append(deltas, kates.NewDelta(deltaType, &endpoint))

		gvk := endpoint.GetObjectKind().GroupVersionKind()

		delta := kates.Delta{
			TypeMeta: kates.TypeMeta{
				APIVersion: gvk.Version,
				Kind:       gvk.Kind,
			},
			ObjectMeta: kates.ObjectMeta{
				Name:      endpoint.GetName(),
				Namespace: endpoint.GetNamespace(),
				// Not sure we need this, but it marshals as null if we don't provide it.
				CreationTimestamp: endpoint.GetCreationTimestamp(),
			},
			DeltaType: deltaType,
		}

		deltas = append(deltas, &delta)
	}

	return deltas
}

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
