package entrypoint

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/consulwatch"
	"github.com/datawire/ambassador/pkg/watt"
	consulapi "github.com/hashicorp/consul/api"
)

func (s *AmbassadorInputs) ReconcileConsul(ctx context.Context, consul *consul) {
	var mappings []*amb.Mapping
	for _, a := range s.annotations {
		m, ok := a.(*amb.Mapping)
		if ok && include(m.Spec.AmbassadorID) {
			mappings = append(mappings, m)
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
			mappings = append(mappings, m)
		}
	}

	consul.reconcile(s.ConsulResolvers, mappings)
}

type consul struct {
	watcher                   Watcher
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

func newConsul(ctx context.Context, watcher Watcher) *consul {
	result := &consul{
		watcher:        watcher,
		resolvers:      make(map[string]*resolver),
		coalescedDirty: make(chan struct{}),
		endpointsCh:    make(chan consulwatch.Endpoints),
		endpoints:      make(map[string]consulwatch.Endpoints),
	}
	go result.run(ctx)
	return result
}

func (c *consul) run(ctx context.Context) {
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
				c.cleanup()
				return
			}
		} else {
			select {
			case ep := <-c.endpointsCh:
				c.updateEndpoints(ep)
				dirty = true
			case <-ctx.Done():
				c.cleanup()
				return
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

func (c *consul) update(snap *watt.ConsulSnapshot) {
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
func (c *consul) cleanup() {
	// XXX: do we care about a clean shutdown
	/*go func() {
		<-ctx.Done()
		w.Stop()
	}()*/

	c.reconcile(nil, nil)
}

// Start and stop consul service watches as needed in order to match the supplied set of resolvers
// and mappings.
func (c *consul) reconcile(resolvers []*amb.ConsulResolver, mappings []*amb.Mapping) {
	// ==First we compute resolvers and their related mappings without actualy changing anything.==
	resolversByName := make(map[string]*amb.ConsulResolver)
	for _, cr := range resolvers {
		name := fmt.Sprintf("%s.%s", cr.GetName(), cr.GetNamespace())
		resolversByName[name] = cr
	}

	mappingsByResolver := make(map[string][]*amb.Mapping)
	for _, m := range mappings {
		if m.Spec.Resolver == "" {
			continue
		}

		// XXX: how are resolvers supposed to be resolved?
		rname := fmt.Sprintf("%s.%s", m.Spec.Resolver, m.GetNamespace())
		_, ok := resolversByName[rname]
		if !ok {
			// XXX: how do we handle typo'd resolvers?
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
		res.reconcile(c.watcher, mappings, c.endpointsCh)
	}

	// If this is the first time we are reconciling, we need to compute conditions for being
	// bootstrapped.
	if !c.firstReconcileHasHappened {
		c.firstReconcileHasHappened = true
		var keysForBootstrap []string
		for _, mappings := range mappingsByResolver {
			for _, m := range mappings {
				keysForBootstrap = append(keysForBootstrap, m.Spec.Service)
			}
		}
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.keysForBootstrap = keysForBootstrap
	}
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

func (r *resolver) reconcile(watcher Watcher, mappings []*amb.Mapping, endpoints chan consulwatch.Endpoints) {
	servicesByName := make(map[string]bool)
	for _, m := range mappings {
		// XXX: how to parse this?
		svc := m.Spec.Service
		servicesByName[svc] = true
		w, ok := r.watches[svc]
		if !ok {
			w = watcher.Watch(r.resolver, m, endpoints)
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
}

type Watcher interface {
	Watch(resolver *amb.ConsulResolver, mapping *amb.Mapping, endpoints chan consulwatch.Endpoints) Stopper
}

type Stopper interface {
	Stop()
}

type consulWatcher struct{}

func (cw *consulWatcher) Watch(resolver *amb.ConsulResolver, mapping *amb.Mapping,
	endpointsCh chan consulwatch.Endpoints) Stopper {
	// XXX: should this part be shared?
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = resolver.Spec.Address
	consul, err := consulapi.NewClient(consulConfig)
	if err != nil {
		panic(err)
	}

	// this part is per service
	svc := mapping.Spec.Service
	w, err := consulwatch.New(consul, resolver.Spec.Datacenter, svc, true)
	if err != nil {
		panic(err)
	}
	w.Watch(func(endpoints consulwatch.Endpoints, e error) {
		endpointsCh <- endpoints
	})

	go func() {
		err = w.Start(context.TODO())
		if err != nil {
			panic(err)
		}
	}()

	return w
}
