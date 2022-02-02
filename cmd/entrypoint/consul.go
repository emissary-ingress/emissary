package entrypoint

import (
	"context"
	"reflect"
	"sync"

	consulapi "github.com/hashicorp/consul/api"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

// consulMapping contains the necessary subset of Ambassador Mapping and TCPMapping
// definitions needed for consul reconcilation and watching to happen.
type consulMapping struct {
	Service  string
	Resolver string
}

func ReconcileConsul(ctx context.Context, consul *consul, s *snapshotTypes.KubernetesSnapshot) error {
	var mappings []consulMapping
	for _, list := range s.Annotations {
		for _, a := range list {
			switch m := a.(type) {
			case *amb.Mapping:
				if include(m.Spec.AmbassadorID) {
					mappings = append(mappings, consulMapping{Service: m.Spec.Service, Resolver: m.Spec.Resolver})
				}
			case *amb.TCPMapping:
				if include(m.Spec.AmbassadorID) {
					mappings = append(mappings, consulMapping{Service: m.Spec.Service, Resolver: m.Spec.Resolver})
				}
			}
		}
	}

	var resolvers []*amb.ConsulResolver
	for _, cr := range s.ConsulResolvers {
		if include(cr.Spec.AmbassadorID) {
			resolvers = append(resolvers, cr)
		}
	}

	for _, m := range s.Mappings {
		if include(m.Spec.AmbassadorID) {
			mappings = append(mappings, consulMapping{Service: m.Spec.Service, Resolver: m.Spec.Resolver})
		}
	}

	for _, tm := range s.TCPMappings {
		if include(tm.Spec.AmbassadorID) {
			mappings = append(mappings, consulMapping{Service: tm.Spec.Service, Resolver: tm.Spec.Resolver})
		}
	}

	return consul.reconcile(ctx, s.ConsulResolvers, mappings)
}

type consul struct {
	watchFunc                 watchConsulFunc
	resolvers                 map[string]*resolver
	firstReconcileHasHappened bool

	// The changed method returns this channel. We write down this channel to signal that a new
	// snapshot is available since the last time the update method was invoke.
	coalescedDirty chan struct{}
	// Individual watches write to this when new endpoint data is available. It is always being read
	// by the implementation, so writing will never block.
	endpointsCh chan consulwatch.Endpoints

	// The mutex protects access to endpoints, keysForBootstrap, and bootstrapped.
	mutex            sync.Mutex
	endpoints        map[string]consulwatch.Endpoints
	keysForBootstrap []string
	bootstrapped     bool
}

func newConsul(ctx context.Context, watchFunc watchConsulFunc) *consul {
	result := &consul{
		watchFunc:      watchFunc,
		resolvers:      make(map[string]*resolver),
		coalescedDirty: make(chan struct{}),
		endpointsCh:    make(chan consulwatch.Endpoints),
		endpoints:      make(map[string]consulwatch.Endpoints),
	}
	go func() {
		if err := result.run(ctx); err != nil {
			panic(err) // TODO: Find a better way of reporting errors from goroutines.
		}
	}()
	return result
}

func (c *consul) run(ctx context.Context) error {
	dirty := false
	for {
		if dirty {
			select {
			case c.coalescedDirty <- struct{}{}:
				dirty = false
			case ep := <-c.endpointsCh:
				c.updateEndpoints(ep)
				dirty = true
			case <-ctx.Done():
				return c.cleanup(ctx)
			}
		} else {
			select {
			case ep := <-c.endpointsCh:
				c.updateEndpoints(ep)
				dirty = true
			case <-ctx.Done():
				return c.cleanup(ctx)
			}
		}
	}
}

func (c *consul) updateEndpoints(endpoints consulwatch.Endpoints) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.endpoints[endpoints.Service] = endpoints
}

func (c *consul) changed() chan struct{} {
	return c.coalescedDirty
}

func (c *consul) update(snap *snapshotTypes.ConsulSnapshot) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	snap.Endpoints = make(map[string]consulwatch.Endpoints, len(c.endpoints))
	for k, v := range c.endpoints {
		snap.Endpoints[k] = v
	}
}

func (c *consul) isBootstrapped() bool {
	if !c.firstReconcileHasHappened {
		return false
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// we want bootstrappedness to be idempotent
	if c.bootstrapped {
		return true
	}

	for _, key := range c.keysForBootstrap {
		if _, ok := c.endpoints[key]; !ok {
			return false
		}
	}

	c.bootstrapped = true

	return true
}

// Stop all service watches.
func (c *consul) cleanup(ctx context.Context) error {
	// XXX: do we care about a clean shutdown
	/*go func() {
		<-ctx.Done()
		w.Stop()
	}()*/

	return c.reconcile(ctx, nil, nil)
}

// Start and stop consul service watches as needed in order to match the supplied set of resolvers
// and mappings.
func (c *consul) reconcile(ctx context.Context, resolvers []*amb.ConsulResolver, mappings []consulMapping) error {
	// ==First we compute resolvers and their related mappings without actualy changing anything.==
	resolversByName := make(map[string]*amb.ConsulResolver)
	for _, cr := range resolvers {
		// Ambassador can find resolvers in any namespace, but they're not partitioned
		// by namespace once located, so just save using the name.
		resolversByName[cr.GetName()] = cr
	}

	mappingsByResolver := make(map[string][]consulMapping)
	for _, m := range mappings {
		// Everything here is keyed off m.Spec.Resolver -- again, it's fine to use a resolver
		// from any namespace, as long as it was loaded.
		//
		// (This implies that if you typo a resolver name, things won't work.)

		rname := m.Resolver

		if rname == "" {
			continue
		}

		_, ok := resolversByName[rname]
		if !ok {
			continue
		}
		mappingsByResolver[rname] = append(mappingsByResolver[rname], m)
	}

	// Prune any resolvers that don't actually have mappings
	for name := range resolversByName {
		_, ok := mappingsByResolver[name]
		if !ok {
			delete(resolversByName, name)
		}
	}

	// ==Now we implement the changes implied by resolversByName and mappingsByResolver.==

	// First we (re)create any new or modified resolvers.
	for name, cr := range resolversByName {
		oldr, ok := c.resolvers[name]
		// The resolver hasn't change so continue. Make sure we only compare the spec, since we
		// don't want to delete/recreate resolvers on things like label changes.
		if ok && reflect.DeepEqual(oldr.resolver.Spec, cr.Spec) {
			continue
		}
		// It exists, but is different, so we delete/recreate i.
		if ok {
			oldr.deleted()
		}
		c.resolvers[name] = newResolver(cr)
	}

	// Now we delete unneeded resolvers.
	for name, resolver := range c.resolvers {
		_, ok := resolversByName[name]
		if !ok {
			resolver.deleted()
			delete(c.resolvers, name)
		}
	}

	// Finally we reconcile each mapping.
	for rname, mappings := range mappingsByResolver {
		res := c.resolvers[rname]
		if err := res.reconcile(ctx, c.watchFunc, mappings, c.endpointsCh); err != nil {
			return err
		}
	}

	// If this is the first time we are reconciling, we need to compute conditions for being
	// bootstrapped.
	if !c.firstReconcileHasHappened {
		c.firstReconcileHasHappened = true
		var keysForBootstrap []string
		for _, mappings := range mappingsByResolver {
			for _, m := range mappings {
				keysForBootstrap = append(keysForBootstrap, m.Service)
			}
		}
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.keysForBootstrap = keysForBootstrap
	}
	return nil
}

type resolver struct {
	resolver *amb.ConsulResolver
	watches  map[string]Stopper
}

func newResolver(spec *amb.ConsulResolver) *resolver {
	return &resolver{resolver: spec, watches: make(map[string]Stopper)}
}

func (r *resolver) deleted() {
	for _, w := range r.watches {
		w.Stop()
	}
}

func (r *resolver) reconcile(ctx context.Context, watchFunc watchConsulFunc, mappings []consulMapping, endpoints chan consulwatch.Endpoints) error {
	servicesByName := make(map[string]bool)
	for _, m := range mappings {
		// XXX: how to parse this?
		svc := m.Service
		servicesByName[svc] = true
		w, ok := r.watches[svc]
		if !ok {
			var err error
			w, err = watchFunc(ctx, r.resolver, svc, endpoints)
			if err != nil {
				return err
			}
			r.watches[svc] = w
		}
	}

	for name, w := range r.watches {
		_, ok := servicesByName[name]
		if !ok {
			w.Stop()
			delete(r.watches, name)
		}
	}
	return nil
}

type watchConsulFunc func(ctx context.Context, resolver *amb.ConsulResolver, svc string, endpoints chan consulwatch.Endpoints) (Stopper, error)

type Stopper interface {
	Stop()
}

func watchConsul(
	ctx context.Context,
	resolver *amb.ConsulResolver,
	svc string,
	endpointsCh chan consulwatch.Endpoints,
) (Stopper, error) {
	// XXX: should this part be shared?
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = resolver.Spec.Address
	consul, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	// this part is per service
	w, err := consulwatch.New(consul, resolver.Spec.Datacenter, svc, true)
	if err != nil {
		return nil, err
	}

	w.Watch(func(endpoints consulwatch.Endpoints, e error) {
		if endpoints.Id == "" {
			// For Ambassador, overwrite the Id with the resolver's datacenter -- the
			// Consul watcher doesn't actually hand back the DC, and we need it.
			endpoints.Id = resolver.Spec.Datacenter
		}

		endpointsCh <- endpoints
	})

	go func() {
		if err := w.Start(ctx); err != nil {
			panic(err) // TODO: Find a better way of reporting errors from goroutines.
		}
	}()

	return w, nil
}
