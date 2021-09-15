package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/datawire/ambassador/v2/cmd/watt/thingconsul"
	"github.com/datawire/ambassador/v2/cmd/watt/thingkube"
	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/limiter"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/ambassador/v2/pkg/watt"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

type WatchHook func(p *supervisor.Process, snapshot string) watchapi.WatchSet

type Aggregator struct {
	// Public //////////////////////////////////////////////////////////////

	// Public input channels that other things can use to send us information.
	KubernetesEvents chan<- thingkube.K8sEvent      // Kubernetes state
	ConsulEvents     chan<- thingconsul.ConsulEvent // Consul endpoints

	// Internal ////////////////////////////////////////////////////////////

	// These are the read-ends of those public inputs
	kubernetesEvents <-chan thingkube.K8sEvent
	consulEvents     <-chan thingconsul.ConsulEvent

	// Output channel used to send info to other things
	k8sWatches    chan<- []watchapi.KubernetesWatchSpec // the k8s watch manager
	consulWatches chan<- []watchapi.ConsulWatchSpec     // consul watch manager
	snapshots     chan<- string                         // the invoker

	// Static information that doesn't change after initialization
	requiredKinds []string // not considered "bootstrapped" until we hear about all these kinds
	watchHook     WatchHook
	limiter       limiter.Limiter
	validator     *kates.Validator

	// Runtime information that changes

	resourcesMu         sync.RWMutex
	ids                 map[string]bool
	kubernetesResources map[string]map[string][]k8s.Resource
	consulEndpoints     map[string]consulwatch.Endpoints

	errorsMu sync.RWMutex
	errors   map[string][]watt.Error

	notifyMux    sync.Mutex
	bootstrapped bool
}

func NewAggregator(snapshots chan<- string, k8sWatches chan<- []watchapi.KubernetesWatchSpec, consulWatches chan<- []watchapi.ConsulWatchSpec,
	requiredKinds []string, watchHook WatchHook, limiter limiter.Limiter, validator *kates.Validator) *Aggregator {
	kubernetesEvents := make(chan thingkube.K8sEvent)
	consulEvents := make(chan thingconsul.ConsulEvent)
	return &Aggregator{
		// public
		KubernetesEvents: kubernetesEvents,
		ConsulEvents:     consulEvents,
		// internal
		kubernetesEvents:    kubernetesEvents,
		consulEvents:        consulEvents,
		k8sWatches:          k8sWatches,
		consulWatches:       consulWatches,
		snapshots:           snapshots,
		requiredKinds:       requiredKinds,
		watchHook:           watchHook,
		limiter:             limiter,
		ids:                 make(map[string]bool),
		kubernetesResources: make(map[string]map[string][]k8s.Resource),
		consulEndpoints:     make(map[string]consulwatch.Endpoints),
		errors:              make(map[string][]watt.Error),
		validator:           validator,
	}
}

func (a *Aggregator) Work(p *supervisor.Process) error {
	// In order to invoke `maybeNotify`, which is a very time consuming
	// operation, we coalesce events:
	//
	// 1. Be continuously reading all available events from
	//    a.kubernetesEvents and a.consulEvents and store thingkube.K8sEvents
	//    in the potentialKubernetesEventSignal variable. This means
	//    at any given point (modulo caveats below), the
	//    potentialKubernetesEventSignal variable will have the
	//    latest Kubernetes event available.
	//
	// 2. At the same time, whenever there is capacity to write
	//    down the kubernetesEventProcessor channel, we send
	//    potentialKubernetesEventSignal to be processed.
	//
	//    The anonymous goroutine below will be constantly reading
	//    from the kubernetesEventProcessor channel and performing
	//    a blocking a.maybeNotify(). This means that we can only
	//    *write* to the kubernetesEventProcessor channel when we are
	//    not currently processing an event, but when that happens, we
	//    will still read from a.kubernetesEvents and a.consulEvents
	//    and update potentialKubernetesEventSignal.
	//
	// There are three caveats to the above:
	//
	// 1. At startup, we don't yet have a event to write, but
	//    we're not processing anything, so we will try to write
	//    something down the kubernetesEventProcessor channel.
	//    To cope with this, the invoking goroutine will ignore events
	//    signals that have a event.skip flag.
	//
	// 2. If we process an event quickly, or if there aren't new
	//    events available, then we end up busy looping and
	//    sending the same potentialKubernetesEventSignal value down
	//    the kubernetesEventProcessor channel multiple times. To cope
	//    with this, whenever we have successfully written to the
	//    kubernetesEventProcessor channel, we do a *blocking* read of
	//    the next event from a.kubernetesEvents and a.consulEvents.
	//
	// 3. Always be calling a.setKubernetesResources as soon as we
	//    receive an event. This is a fast non-blocking call that
	//    update watches, we can't coalesce this call.

	p.Ready()

	type eventSignal struct {
		kubernetesEvent thingkube.K8sEvent
		skip            bool
	}

	kubernetesEventProcessor := make(chan eventSignal)
	go func() {
		for event := range kubernetesEventProcessor {
			if event.skip {
				// ignore the initial eventSignal to deal with the
				// corner case where we haven't yet received an event yet.
				continue
			}
			a.maybeNotify(p)
		}
	}()

	potentialKubernetesEventSignal := eventSignal{kubernetesEvent: thingkube.K8sEvent{}, skip: true}
	for {
		select {
		case potentialKubernetesEvent := <-a.kubernetesEvents:
			// if a new KubernetesEvents is available to be read,
			// and we can't write to the kubernetesEventProcessor channel,
			// then we will overwrite potentialKubernetesEvent
			// with a newer event while still processing a.setKubernetesResources
			a.setKubernetesResources(potentialKubernetesEvent)
			potentialKubernetesEventSignal = eventSignal{kubernetesEvent: potentialKubernetesEvent, skip: false}
		case kubernetesEventProcessor <- potentialKubernetesEventSignal:
			// if we aren't currently blocked in
			// a.maybeNotify() then the above goroutine will be
			// reading from the kubernetesEventProcessor channel and we
			// will send the current potentialKubernetesEventSignal
			// value over the kubernetesEventProcessor channel to be
			// processed
			select {
			case potentialKubernetesEvent := <-a.kubernetesEvents:
				// here we do blocking read of the next event for caveat #2.
				a.setKubernetesResources(potentialKubernetesEvent)
				potentialKubernetesEventSignal = eventSignal{kubernetesEvent: potentialKubernetesEvent, skip: false}
			case event := <-a.consulEvents:
				a.updateConsulResources(event)
				a.maybeNotify(p)
			case <-p.Shutdown():
				return nil
			}
		case event := <-a.consulEvents:
			// we are always reading and processing ConsulEvents directly,
			// not coalescing them.
			a.updateConsulResources(event)
			a.maybeNotify(p)
		case <-p.Shutdown():
			return nil
		}
	}
}

func (a *Aggregator) updateConsulResources(event thingconsul.ConsulEvent) {
	a.resourcesMu.Lock()
	defer a.resourcesMu.Unlock()
	a.ids[event.WatchId] = true
	a.consulEndpoints[event.Endpoints.Service] = event.Endpoints
}

func (a *Aggregator) setKubernetesResources(event thingkube.K8sEvent) {
	if len(event.Errors) > 0 {
		a.errorsMu.Lock()
		defer a.errorsMu.Unlock()
		for _, kError := range event.Errors {
			a.errors[kError.Source] = append(a.errors[kError.Source], kError)
		}
	} else {
		a.resourcesMu.Lock()
		defer a.resourcesMu.Unlock()
		a.ids[event.WatchID] = true
		submap, ok := a.kubernetesResources[event.WatchID]
		if !ok {
			submap = make(map[string][]k8s.Resource)
			a.kubernetesResources[event.WatchID] = submap
		}
		submap[event.Kind] = event.Resources
	}
}

func (a *Aggregator) generateSnapshot(p *supervisor.Process) (string, error) {
	a.errorsMu.RLock()
	defer a.errorsMu.RUnlock()
	a.resourcesMu.RLock()
	defer a.resourcesMu.RUnlock()

	k8sResources := make(map[string][]k8s.Resource)
	for _, submap := range a.kubernetesResources {
		for k, v := range submap {
			a.validate(p, v)
			k8sResources[k] = append(k8sResources[k], v...)
		}
	}
	s := watt.Snapshot{
		Consul:     watt.ConsulSnapshot{Endpoints: a.consulEndpoints},
		Kubernetes: k8sResources,
		Errors:     a.errors,
	}

	jsonBytes, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return "{}", err
	}

	return string(jsonBytes), nil
}

// watt only runs in legacy mode now, and legacy mode is defined
// to not do fast validation.
// var fastValidation = len(os.Getenv("AMBASSADOR_FAST_VALIDATION")) > 0
var fastValidation = false

func (a *Aggregator) validate(p *supervisor.Process, resources []k8s.Resource) {
	if !fastValidation {
		return
	}

	for _, r := range resources {
		err := a.validator.Validate(p.Context(), map[string]interface{}(r))
		if err == nil {
			delete(r, "errors")
		} else {
			r["errors"] = err.Error()
		}
	}
}

func (a *Aggregator) isKubernetesBootstrapped(p *supervisor.Process) bool {
	a.resourcesMu.RLock()
	defer a.resourcesMu.RUnlock()

	submap, sok := a.kubernetesResources[""]
	if !sok {
		return false
	}
	for _, k := range a.requiredKinds {
		_, ok := submap[k]
		if !ok {
			return false
		}
	}
	return true
}

// Returns true if the current state of the world is complete. The
// kubernetes state of the world is always complete by definition
// because the kubernetes client provides that guarantee. The
// aggregate state of the world is complete when any consul services
// referenced by kubernetes have populated endpoint information (even
// if the value of the populated info is an empty set of endpoints).
func (a *Aggregator) isComplete(p *supervisor.Process, watchset watchapi.WatchSet) bool {
	a.resourcesMu.RLock()
	defer a.resourcesMu.RUnlock()

	complete := true

	for _, w := range watchset.KubernetesWatches {
		if _, ok := a.ids[w.WatchId()]; ok {
			p.Debugf("initialized k8s watch: %s", w.WatchId())
		} else {
			complete = false
			p.Debugf("waiting for k8s watch: %s", w.WatchId())
		}
	}

	for _, w := range watchset.ConsulWatches {
		if _, ok := a.ids[w.WatchId()]; ok {
			p.Debugf("initialized consul watch: %s", w.WatchId())
		} else {
			complete = false
			p.Debugf("waiting for consul watch: %s", w.WatchId())
		}
	}

	return complete
}

func (a *Aggregator) maybeNotify(p *supervisor.Process) {
	now := time.Now()
	delay := a.limiter.Limit(now)
	if delay == 0 {
		a.notify(p)
	} else if delay > 0 {
		time.AfterFunc(delay, func() {
			a.notify(p)
		})
	}
}

func (a *Aggregator) notify(p *supervisor.Process) {
	a.notifyMux.Lock()
	defer a.notifyMux.Unlock()

	if !a.isKubernetesBootstrapped(p) {
		return
	}

	watchset := a.getWatches(p)

	p.Debugf("found %d kubernetes watches", len(watchset.KubernetesWatches))
	p.Debugf("found %d consul watches", len(watchset.ConsulWatches))
	a.k8sWatches <- watchset.KubernetesWatches
	a.consulWatches <- watchset.ConsulWatches

	if !a.bootstrapped && a.isComplete(p, watchset) {
		p.Logf("bootstrapped!")
		a.bootstrapped = true
	}

	if a.bootstrapped {
		snapshot, err := a.generateSnapshot(p)
		if err != nil {
			p.Logf("generate snapshot failed %v", err)
			return
		}

		a.snapshots <- snapshot
	}
}

func (a *Aggregator) getWatches(p *supervisor.Process) watchapi.WatchSet {
	snapshot, err := a.generateSnapshot(p)
	if err != nil {
		p.Logf("generate snapshot failed %v", err)
		return watchapi.WatchSet{}
	}
	result := a.watchHook(p, snapshot)
	return result.Interpolate()
}

func ExecWatchHook(watchHooks []string) WatchHook {
	return func(p *supervisor.Process, snapshot string) watchapi.WatchSet {
		result := watchapi.WatchSet{}

		for _, hook := range watchHooks {
			ws := invokeHook(p.Context(), hook, snapshot)
			result.KubernetesWatches = append(result.KubernetesWatches, ws.KubernetesWatches...)
			result.ConsulWatches = append(result.ConsulWatches, ws.ConsulWatches...)
		}

		return result
	}
}

func invokeHook(ctx context.Context, hook, snapshot string) watchapi.WatchSet {
	cmd := dexec.CommandContext(ctx, "sh", "-c", hook)
	cmd.DisableLogging = true
	cmd.Stdin = strings.NewReader(snapshot)
	watches, err := cmd.Output()
	if err != nil {
		dlog.Infof(ctx, "watch hook failed: %v", err)
		return watchapi.WatchSet{}
	}

	decoder := json.NewDecoder(bytes.NewReader(watches))
	decoder.DisallowUnknownFields()

	var result watchapi.WatchSet
	if err := decoder.Decode(&result); err != nil {
		dlog.Infof(ctx, "watchset decode failed: %v", err)
		return watchapi.WatchSet{}
	}

	return result
}
