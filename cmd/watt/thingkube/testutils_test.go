package thingkube_test

import (
	"fmt"
	"time"

	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
)

func createDoNothingWorker(name string) *supervisor.Worker {
	return &supervisor.Worker{
		Name: name,
		Work: func(p *supervisor.Process) error {
			<-p.Shutdown()
			time.Sleep(500 * time.Millisecond)
			return nil
		},
		Retry: false,
	}
}

type MockWatchMaker struct {
	errorBeforeCreate bool
}

func (m *MockWatchMaker) MakeKubernetesWatch(spec watchapi.KubernetesWatchSpec) (*supervisor.Worker, error) {
	if m.errorBeforeCreate {
		return nil, fmt.Errorf("failed to create watch (errorBeforeCreate: %t)", m.errorBeforeCreate)
	}

	return createDoNothingWorker(
		fmt.Sprintf("%s|%s|%s|%s", spec.Namespace, spec.Kind, spec.FieldSelector, spec.LabelSelector)), nil
}

func (m *MockWatchMaker) MakeConsulWatch(spec watchapi.ConsulWatchSpec) (*supervisor.Worker, error) {
	if m.errorBeforeCreate {
		return nil, fmt.Errorf("failed to create watch (errorBeforeCreate: %t)", m.errorBeforeCreate)
	}

	return createDoNothingWorker(fmt.Sprintf("%s|%s|%s", spec.ConsulAddress, spec.Datacenter, spec.ServiceName)), nil
}
