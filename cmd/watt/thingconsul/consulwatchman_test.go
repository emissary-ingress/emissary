package thingconsul_test

import (
	"context"
	"testing"
	"time"

	"github.com/ecodia/golang-awaitility/awaitility"
	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/dlib/dlog"

	. "github.com/datawire/ambassador/v2/cmd/watt/thingconsul"
)

type consulwatchmanIsolator struct {
	aggregatorToWatchmanCh        chan []watchapi.ConsulWatchSpec
	consulEndpointsToAggregatorCh chan consulwatch.Endpoints
	watchman                      ConsulWatchMan
	sup                           *supervisor.Supervisor
	done                          chan struct{}
	t                             *testing.T
	cancel                        context.CancelFunc
}

func TestAddAndRemoveConsulWatchers(t *testing.T) {
	iso := startConsulwatchmanIsolator(t)
	defer iso.Stop()

	specs := []watchapi.ConsulWatchSpec{
		{ConsulAddress: "127.0.0.1", ServiceName: "foo-in-consul", Datacenter: "dc1"},
		{ConsulAddress: "127.0.0.1", ServiceName: "bar-in-consul", Datacenter: "dc1"},
		{ConsulAddress: "127.0.0.1", ServiceName: "baz-in-consul", Datacenter: "dc1"},
	}

	iso.aggregatorToWatchmanCh <- specs

	err := awaitility.Await(100*time.Millisecond, 1000*time.Millisecond, func() bool {
		return iso.watchman.NumWatched() == len(specs)
	})

	if err != nil {
		t.Fatal(err)
	}

	iso.watchman.WithWatched(func(watched map[string]*supervisor.Worker) {
		assert.Len(t, watched, len(specs))
		for k, worker := range watched {
			assert.Equal(t, k, worker.Name)
		}
	})

	specs = []watchapi.ConsulWatchSpec{
		{ConsulAddress: "127.0.0.1", ServiceName: "bar-in-consul", Datacenter: "dc1"},
		{ConsulAddress: "127.0.0.1", ServiceName: "baz-in-consul", Datacenter: "dc1"},
	}

	iso.aggregatorToWatchmanCh <- specs
	err = awaitility.Await(100*time.Millisecond, 1000*time.Millisecond, func() bool {
		return iso.watchman.NumWatched() == len(specs)
	})

	if err != nil {
		t.Fatal(err)
	}

	iso.watchman.WithWatched(func(watched map[string]*supervisor.Worker) {
		assert.Len(t, watched, len(specs))
		for k, worker := range watched {
			assert.Equal(t, k, worker.Name)
		}
	})

	specs = []watchapi.ConsulWatchSpec{
		{ConsulAddress: "127.0.0.1", ServiceName: "bar-in-consul", Datacenter: "dc1"},
		{ConsulAddress: "127.0.0.1", ServiceName: "baz-in-consul", Datacenter: "dc1"},
	}

	iso.aggregatorToWatchmanCh <- specs
	err = awaitility.Await(100*time.Millisecond, 1000*time.Millisecond, func() bool {
		return iso.watchman.NumWatched() == len(specs)
	})

	if err != nil {
		t.Fatal(err)
	}

	iso.watchman.WithWatched(func(watched map[string]*supervisor.Worker) {
		assert.Len(t, watched, len(specs))
		for k, worker := range watched {
			assert.Equal(t, k, worker.Name)
		}
	})
}

func startConsulwatchmanIsolator(t *testing.T) *consulwatchmanIsolator {
	iso := newConsulwatchmanIsolator(t)
	iso.Start()
	return iso
}

func (iso *consulwatchmanIsolator) Start() {
	go func() {
		errs := iso.sup.Run()
		if len(errs) > 0 {
			iso.t.Errorf("unexpected errors: %v", errs)
		}
		close(iso.done)
	}()
}

func (iso *consulwatchmanIsolator) Stop() {
	iso.sup.Shutdown()
	iso.cancel()
	<-iso.done
}

func newConsulwatchmanIsolator(t *testing.T) *consulwatchmanIsolator {
	iso := &consulwatchmanIsolator{
		// by using zero length channels for inputs here, we can
		// control the total ordering of all inputs and therefore
		// intentionally trigger any order of events we want to test
		aggregatorToWatchmanCh: make(chan []watchapi.ConsulWatchSpec),

		// we need to create buffered channels for outputs because
		// nothing is asynchronously reading them in the test
		consulEndpointsToAggregatorCh: make(chan consulwatch.Endpoints, 100),

		// for signaling when the isolator is done
		done: make(chan struct{}),
	}

	iso.watchman = NewConsulWatchMan(nil, iso.aggregatorToWatchmanCh)

	ctx, cancel := context.WithTimeout(dlog.NewTestContext(t, false), 10*time.Second)
	iso.cancel = cancel
	iso.sup = supervisor.WithContext(ctx)
	iso.sup.Supervise(&supervisor.Worker{
		Name: "consulwatchman",
		Work: iso.watchman.Work,
	})
	return iso
}
