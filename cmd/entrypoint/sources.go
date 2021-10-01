package entrypoint

import (
	"context"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kates/k8sresourcetypes"
)

type K8sSource interface {
	Watch(ctx context.Context, queries ...kates.Query) (K8sWatcher, error)
}

type K8sWatcher interface {
	Changed() chan struct{}
	FilteredUpdate(ctx context.Context, target interface{}, deltas *[]*kates.Delta, predicate func(*k8sresourcetypes.Unstructured) bool) (bool, error)
}

type IstioCertSource interface {
	Watch(ctx context.Context) (IstioCertWatcher, error)
}

type IstioCertWatcher interface {
	Changed() chan IstioCertUpdate
}
