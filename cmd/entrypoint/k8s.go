package entrypoint

import (
	"context"

	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

type k8sSource struct {
	client *kates.Client
}

func (k *k8sSource) Watch(ctx context.Context, queries ...kates.Query) K8sWatcher {
	acc := k.client.Watch(ctx, queries...)
	return acc
}

func newK8sSource(client *kates.Client) *k8sSource {
	return &k8sSource{
		client: client,
	}
}

// k8sWatchManager is the interface between all the Kubernetes-watching stuff
// and the watcher (in watcher.go).
type k8sWatchManager struct {
	// To track Kubernetes things, we need a snapshot and a K8sWatcher.
	// The snapshot is an internally-consistent view of the stuff in our K8s
	// cluster that applies to us; the K8sWatcher is responsible for paying
	// attention to the outside world and managing the logic around the
	// "internally consistent" part of that statement.
	//
	// The snapshot here is also where we store Istio cert state, since we want
	// Istio certs to look like K8s secrets. This is kind of a crock.
	snapshot *snapshotTypes.KubernetesSnapshot
	watcher  K8sWatcher

	// We use kates.Delta objects to indicate to the rest of Ambassador
	// what has actually changed between one snapshot and the next.
	deltas []*kates.Delta
}

// Changed returns a channel to listen on for change notifications dealing with
// Kubernetes stuff.
func (k8s *k8sWatchManager) Changed() chan struct{} {
	return k8s.watcher.Changed()
}

// Update actually does the work of updating our internal state with changes,
// and returning whether or not anything of interest actually happened.
func (k8s *k8sWatchManager) Update(ctx context.Context, isValid func(un *kates.Unstructured) bool) bool {
	dlog.Debugf(ctx, "WATCHER: K8s fired")

	// Kubernetes has some changes. We use our watcher's FilteredUpdate method
	// to sort out whether any of the changes are worth paying attention to. If
	// there are, stuff 'em in k8s.deltas.
	k8s.deltas = make([]*kates.Delta, 0)

	if !k8s.watcher.FilteredUpdate(k8s.snapshot, &k8s.deltas, isValid) {
		dlog.Debugf(ctx, "WATCHER: filtered-update dropped everything")
		return false
	}

	dlog.Debugf(ctx, "WATCHER: new deltas (%d): %s", len(k8s.deltas), deltaSummary(k8s.deltas))
	return true
}

// newK8sWatchManager returns a new K8sWatchManager. I know, profound.
func newK8sWatchManager(ctx context.Context, watcher K8sWatcher) *k8sWatchManager {
	k8s := k8sWatchManager{
		snapshot: NewKubernetesSnapshot(),
		watcher:  watcher,
		// No need to initialize deltas, we do that in Update()
	}

	return &k8s
}
