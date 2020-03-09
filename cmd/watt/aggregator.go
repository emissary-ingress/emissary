package watt

import (
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/datawire/ambassador/pkg/consulwatch"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/limiter"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/ambassador/pkg/watt"
)

type WatchHook func(p *supervisor.Process, snapshot string) WatchSet

type aggregator struct {
	// Input channel used to tell us about kubernetes state.
	KubernetesEvents chan k8sEvent
	// Input channel used to tell us about consul endpoints.
	ConsulEvents chan consulEvent
	// Output channel used to communicate with the k8s watch manager.
	k8sWatches chan<- []KubernetesWatchSpec
	// Output channel used to communicate with the consul watch manager.
	consulWatches chan<- []ConsulWatchSpec
	// Output channel used to communicate with the invoker.
	snapshots chan<- string
	// We won't consider ourselves "bootstrapped" until we hear
	// about all these kinds.
	requiredKinds       []string
	watchHook           WatchHook
	limiter             limiter.Limiter
	ids                 map[string]bool
	kubernetesResources map[string]map[string][]k8s.Resource
	consulEndpoints     map[string]consulwatch.Endpoints
	bootstrapped        bool
	notifyMux           sync.Mutex
	errors              map[string][]watt.Error
}

func NewAggregator(snapshots chan<- string, k8sWatches chan<- []KubernetesWatchSpec, consulWatches chan<- []ConsulWatchSpec,
	requiredKinds []string, watchHook WatchHook, limiter limiter.Limiter) *aggregator {
	return &aggregator{
		KubernetesEvents:    make(chan k8sEvent),
		ConsulEvents:        make(chan consulEvent),
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
	}
}

func (a *aggregator) Work(p *supervisor.Process) error {
	// In order to invoke `maybeNotify`, which is a very time consuming
	// operation, we coalesce events:
	//
	// 1. Be continuously reading all available events from
	//    a.KubernetesEvents and a.ConsulEvents and store k8sEvents
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
	//    will still read from a.KubernetesEvents and a.ConsulEvents
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
	//    the next event from a.KubernetesEvents and a.ConsulEvents.
	//
	// 3. Always be calling a.setKubernetesResources as soon as we
	//    receive an event. This is a fast non-blocking call that
	//    update watches, we can't coalesce this call.

	p.Ready()

	type eventSignal struct {
		kubernetesEvent k8sEvent
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

	potentialKubernetesEventSignal := eventSignal{kubernetesEvent: k8sEvent{}, skip: true}
	for {
		select {
		case potentialKubernetesEvent := <-a.KubernetesEvents:
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
			case potentialKubernetesEvent := <-a.KubernetesEvents:
				// here we do blocking read of the next event for caveat #2.
				a.setKubernetesResources(potentialKubernetesEvent)
				potentialKubernetesEventSignal = eventSignal{kubernetesEvent: potentialKubernetesEvent, skip: false}
			case event := <-a.ConsulEvents:
				a.updateConsulResources(event)
				a.maybeNotify(p)
			case <-p.Shutdown():
				return nil
			}
		case event := <-a.ConsulEvents:
			// we are always reading and processing ConsulEvents directly,
			// not coalescing them.
			a.updateConsulResources(event)
			a.maybeNotify(p)
		case <-p.Shutdown():
			return nil
		}
	}
}

func (a *aggregator) updateConsulResources(event consulEvent) {
	a.ids[event.WatchId] = true
	a.consulEndpoints[event.Endpoints.Service] = event.Endpoints
}

func (a *aggregator) setKubernetesResources(event k8sEvent) {
	if len(event.errors) > 0 {
		for _, kError := range event.errors {
			a.errors[kError.Source] = append(a.errors[kError.Source], kError)
		}
		return
	}
	a.ids[event.watchId] = true
	submap, ok := a.kubernetesResources[event.watchId]
	if !ok {
		submap = make(map[string][]k8s.Resource)
		a.kubernetesResources[event.watchId] = submap
	}
	submap[event.kind] = event.resources
}

func (a *aggregator) generateSnapshot() (string, error) {
	errors := make(map[string][]watt.Error, len(a.errors))
	for source, errs := range a.errors {
		errors[source] = errs
	}

	k8sResources := make(map[string][]k8s.Resource)
	for _, submap := range a.kubernetesResources { // keyed by watchID
		for k, v := range submap {
			k8sResources[k] = append(k8sResources[k], v...)
			for _, resource := range v {
				// FIXME(lukeshu): Which resources to look for annotations on is hard-coded; it probably
				// should be specifiable from the outside like normal resource watches.  This is putting
				// a business-logic decision in the systems-logic code.
				if resource.QKind() == "Service.v1." ||
					resource.QKind() == "Ingress.v1beta1.extensions" ||
					resource.QKind() == "Ingress.v1beta1.networking.k8s.io" || // 1.14+
					resource.QKind() == "Ingress.v1.networking.k8s.io" { // slated for 1.18, I think?
					annotationResources, annotationErrs := parseAnnotationResources(resource)
					for _, annotationResource := range annotationResources {
						k8sResources[annotationResource.Kind()] = append(k8sResources[annotationResource.Kind()], annotationResource)
					}
					for _, annotationErr := range annotationErrs {
						errors[annotationErr.Source] = append(errors[annotationErr.Source], annotationErr)
					}
				}
			}
		}
	}
	s := watt.Snapshot{
		Consul:     watt.ConsulSnapshot{Endpoints: a.consulEndpoints},
		Kubernetes: k8sResources,
		Errors:     errors,
	}

	jsonBytes, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return "{}", err
	}

	return string(jsonBytes), nil
}

func (a *aggregator) isKubernetesBootstrapped(p *supervisor.Process) bool {
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
func (a *aggregator) isComplete(p *supervisor.Process, watchset WatchSet) bool {
	complete := true

	for _, w := range watchset.KubernetesWatches {
		if _, ok := a.ids[w.WatchId()]; ok {
			p.Logf("initialized k8s watch: %s", w.WatchId())
		} else {
			complete = false
			p.Logf("waiting for k8s watch: %s", w.WatchId())
		}
	}

	for _, w := range watchset.ConsulWatches {
		if _, ok := a.ids[w.WatchId()]; ok {
			p.Logf("initialized consul watch: %s", w.WatchId())
		} else {
			complete = false
			p.Logf("waiting for consul watch: %s", w.WatchId())
		}
	}

	return complete
}

func (a *aggregator) maybeNotify(p *supervisor.Process) {
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

func (a *aggregator) notify(p *supervisor.Process) {
	a.notifyMux.Lock()
	defer a.notifyMux.Unlock()

	if !a.isKubernetesBootstrapped(p) {
		return
	}

	watchset := a.getWatches(p)

	p.Logf("found %d kubernetes watches", len(watchset.KubernetesWatches))
	p.Logf("found %d consul watches", len(watchset.ConsulWatches))
	a.k8sWatches <- watchset.KubernetesWatches
	a.consulWatches <- watchset.ConsulWatches

	if !a.bootstrapped && a.isComplete(p, watchset) {
		p.Logf("bootstrapped!")
		a.bootstrapped = true
	}

	if a.bootstrapped {
		snapshot, err := a.generateSnapshot()
		if err != nil {
			p.Logf("generate snapshot failed %v", err)
			return
		}

		a.snapshots <- snapshot
	}
}

func (a *aggregator) getWatches(p *supervisor.Process) WatchSet {
	snapshot, err := a.generateSnapshot()
	if err != nil {
		p.Logf("generate snapshot failed %v", err)
		return WatchSet{}
	}
	result := a.watchHook(p, snapshot)
	return result.interpolate()
}

func ExecWatchHook(watchHooks []string) WatchHook {
	return func(p *supervisor.Process, snapshot string) WatchSet {
		result := WatchSet{}

		for _, hook := range watchHooks {
			ws := invokeHook(p, hook, snapshot)
			result.KubernetesWatches = append(result.KubernetesWatches, ws.KubernetesWatches...)
			result.ConsulWatches = append(result.ConsulWatches, ws.ConsulWatches...)
		}

		return result
	}
}

func lines(st string) []string {
	return strings.Split(st, "\n")
}

func invokeHook(p *supervisor.Process, hook, snapshot string) WatchSet {
	cmd := exec.Command("sh", "-c", hook)
	cmd.Stdin = strings.NewReader(snapshot)
	var watches, errors strings.Builder
	cmd.Stdout = &watches
	cmd.Stderr = &errors
	err := cmd.Run()
	stderr := errors.String()
	if stderr != "" {
		for _, line := range lines(stderr) {
			p.Logf("watch hook stderr: %s", line)
		}
	}
	if err != nil {
		p.Logf("watch hook failed: %v", err)
		return WatchSet{}
	}

	encoded := watches.String()

	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	result := WatchSet{}
	err = decoder.Decode(&result)
	if err != nil {
		for _, line := range lines(encoded) {
			p.Logf("watch hook: %s", line)
		}
		p.Logf("watchset decode failed: %v", err)
		return WatchSet{}
	}

	return result
}
