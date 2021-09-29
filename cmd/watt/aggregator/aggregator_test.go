package aggregator_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/cmd/watt/thingconsul"
	"github.com/datawire/ambassador/v2/cmd/watt/thingkube"
	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/limiter"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/ambassador/v2/pkg/watt"
	"github.com/datawire/dlib/dlog"

	. "github.com/datawire/ambassador/v2/cmd/watt/aggregator"
)

type aggIsolator struct {
	snapshots     chan string
	k8sWatches    chan []watchapi.KubernetesWatchSpec
	consulWatches chan []watchapi.ConsulWatchSpec
	aggregator    *Aggregator
	sup           *supervisor.Supervisor
	done          chan struct{}
	t             *testing.T
	cancel        context.CancelFunc
}

func newAggIsolator(t *testing.T, requiredKinds []string, watchHook WatchHook) *aggIsolator {
	// aggregator uses zero length channels for its inputs so we can
	// control the total ordering of all inputs and therefore
	// intentionally trigger any order of events we want to test
	iso := &aggIsolator{
		// we need to create buffered channels for outputs
		// because nothing is asynchronously reading them in
		// the test
		k8sWatches:    make(chan []watchapi.KubernetesWatchSpec, 100),
		consulWatches: make(chan []watchapi.ConsulWatchSpec, 100),
		snapshots:     make(chan string, 100),
		// for signaling when the isolator is done
		done: make(chan struct{}),
	}
	client, err := kates.NewClient(kates.ClientConfig{})
	require.NoError(t, err)
	validator, err := kates.NewValidator(client, nil)
	require.NoError(t, err)
	iso.aggregator = NewAggregator(iso.snapshots, iso.k8sWatches, iso.consulWatches, requiredKinds, watchHook,
		limiter.NewUnlimited(),
		validator)
	ctx, cancel := context.WithTimeout(dlog.NewTestContext(t, false), 10*time.Second)
	iso.cancel = cancel
	iso.sup = supervisor.WithContext(ctx)
	iso.sup.Supervise(&supervisor.Worker{
		Name: "aggregator",
		Work: iso.aggregator.Work,
	})
	iso.t = t
	return iso
}

func startAggIsolator(t *testing.T, requiredKinds []string, watchHook WatchHook) *aggIsolator {
	iso := newAggIsolator(t, requiredKinds, watchHook)
	iso.Start()
	return iso
}

func (iso *aggIsolator) Start() {
	go func() {
		errs := iso.sup.Run()
		if len(errs) > 0 {
			iso.t.Errorf("unexpected errors: %v", errs)
		}
		close(iso.done)
	}()
}

func (iso *aggIsolator) Stop() {
	iso.sup.Shutdown()
	iso.cancel()
	<-iso.done
}

func resources(input string) []k8s.Resource {
	result, err := k8s.ParseResources("aggregator-test", input)
	if err != nil {
		panic(err)
	}
	return result
}

var (
	SERVICES = resources(`
---
kind: Service
apiVersion: v1
metadata:
  name: foo
spec:
  selector:
    pod: foo
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
`)
	RESOLVER = resources(`
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: bar
  annotations:
    "getambassador.io/consul-resolver": "true"
data:
  consulAddress: "127.0.0.1:8500"
  datacenter: "dc1"
  service: "bar"
`)
)

// make sure we shutdown even before achieving a bootstrapped state
func TestAggregatorShutdown(t *testing.T) {
	iso := startAggIsolator(t, nil, nil)
	defer iso.Stop()
}

var WATCH = watchapi.ConsulWatchSpec{
	ConsulAddress: "127.0.0.1:8500",
	Datacenter:    "dc1",
	ServiceName:   "bar",
}

// Check that we bootstrap properly... this means *not* emitting a
// snapshot until we have:
//
//   a) achieved synchronization with the kubernetes API server
//
//   b) received (possibly empty) endpoint info about all referenced
//      consul services...
func TestAggregatorBootstrap(t *testing.T) {
	watchHook := func(p *supervisor.Process, snapshot string) watchapi.WatchSet {
		if strings.Contains(snapshot, "configmap") {
			return watchapi.WatchSet{
				ConsulWatches: []watchapi.ConsulWatchSpec{WATCH},
			}
		} else {
			return watchapi.WatchSet{}
		}
	}
	iso := startAggIsolator(t, []string{"service", "configmap"}, watchHook)
	defer iso.Stop()

	// initial kubernetes state is just services
	iso.aggregator.KubernetesEvents <- thingkube.K8sEvent{
		WatchID:   "",
		Kind:      "service",
		Resources: SERVICES,
		Errors:    nil,
	}

	// we should not generate a snapshot or consulWatches yet
	// because we specified configmaps are required
	expect(t, iso.consulWatches, Timeout(100*time.Millisecond))
	expect(t, iso.snapshots, Timeout(100*time.Millisecond))

	// the configmap references a consul service, so we shouldn't
	// get a snapshot yet, but we should get watches
	iso.aggregator.KubernetesEvents <- thingkube.K8sEvent{
		WatchID:   "",
		Kind:      "configmap",
		Resources: RESOLVER,
		Errors:    nil,
	}
	expect(t, iso.snapshots, Timeout(100*time.Millisecond))
	expect(t, iso.consulWatches, func(watches []watchapi.ConsulWatchSpec) bool {
		if len(watches) != 1 {
			t.Logf("expected 1 watch, got %d watches", len(watches))
			return false
		}

		if watches[0].ServiceName != "bar" {
			return false
		}

		return true
	})

	// now lets send in the first endpoints, and we should get a
	// snapshot
	iso.aggregator.ConsulEvents <- thingconsul.ConsulEvent{
		WatchId: WATCH.WatchId(),
		Endpoints: consulwatch.Endpoints{
			Service: "bar",
			Endpoints: []consulwatch.Endpoint{
				{
					Service: "bar",
					Address: "1.2.3.4",
					Port:    80,
				},
			},
		},
	}

	expect(t, iso.snapshots, func(snapshot string) bool {
		s := &watt.Snapshot{}
		err := json.Unmarshal([]byte(snapshot), s)
		if err != nil {
			return false
		}
		_, ok := s.Consul.Endpoints["bar"]
		return ok
	})
}
