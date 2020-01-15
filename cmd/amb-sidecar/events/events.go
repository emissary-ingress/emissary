package events

import (
	"os"
	"sync"

	// k8s misc
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
	k8sRecord "k8s.io/client-go/tools/record"

	// k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/ambassador/pkg/dlog"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

type EventLogger struct {
	cfg        types.Config
	hostname   string
	coreClient *k8sClientCoreV1.CoreV1Client
	logger     dlog.Logger

	lock      sync.Mutex
	recorders map[string]k8sRecord.EventRecorder
}

func NewEventLogger(cfg types.Config, coreClient *k8sClientCoreV1.CoreV1Client, logger dlog.Logger) (*EventLogger, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return &EventLogger{
		cfg:        cfg,
		hostname:   hostname,
		coreClient: coreClient,
		logger:     logger,

		recorders: make(map[string]k8sRecord.EventRecorder),
	}, nil
}

func (el *EventLogger) NoNamespace() k8sRecord.EventRecorder {
	return el.Namespace(el.cfg.PodNamespace)
}

func (el *EventLogger) Namespace(namespace string) k8sRecord.EventRecorder {
	el.lock.Lock()
	defer el.lock.Unlock()

	if existing, exists := el.recorders[namespace]; exists {
		return existing
	}

	eventBroadcaster := k8sRecord.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8sClientCoreV1.EventSinkImpl{
		Interface: el.coreClient.Events(namespace),
	})
	eventBroadcaster.StartLogging(el.logger.
		WithField("event-namespace", namespace).
		Infof)
	eventRecorder := eventBroadcaster.NewRecorder(
		// a k8sRuntime.Scheme, only used if we try to record an event
		// for an object without apiVersion/kind.  *We* don't do that,
		// but client-go might internally, if we pass it an
		// EventRecorder.  So, load in client-go's internal Scheme.
		k8sScheme.Scheme,

		k8sTypesCoreV1.EventSource{
			Component: "Ambassador Edge Stack",
			Host:      el.hostname,
		})

	el.recorders[namespace] = eventRecorder

	return eventRecorder
}
