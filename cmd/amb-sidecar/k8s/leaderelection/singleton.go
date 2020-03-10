package leaderelection

import (
	"context"
	"sync"
	"time"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/cmd/amb-sidecar/types"

	// k8s misc
	k8sLeaderElection "k8s.io/client-go/tools/leaderelection"
)

// RunAsSingleton performs leader-election to make sure that only one copy of
// the given callback function is running in the cluster at a time (per
// ambassador_id per namespace).  RunAsSingleton may return immediately with an
// error if it fails to initialize; otherwise it returns nil, and only returns
// when the context is canceled.  Until then, it participates in leader-election
// in a loop.
//
// The context given to the callback function is canceled if when we are no
// longer the leader (or the parent context has been canceled).
//
// The cluster-wide lock will be in the AMBASSADOR_NAMESPACE, named either
// "{leasename}" or "{leasename}-{ambassador_id}" (depending on whether
// ambassador_id is "default").
//
// The leaseDuration controls how quickly elections happen--a shorter duration
// means that the death of a leader is noticed more quickly, but puts more load
// on the api-server.  Kubernetes itself uses '15 seconds', so that is a good
// value for "real time" uses.  The ACME client isn't terribly time-sensitive,
// so it uses a much more relaxed '60 seconds'.  You can think of this as the
// maximum amount of time that it may take for a new leader to be elected after
// the previous leader is killed.
func RunAsSingleton(
	ctx context.Context,
	cfg types.Config,
	kubeinfo *k8s.KubeInfo,
	leasename string,
	leaseDuration time.Duration,
	fn func(context.Context),
) error {

	leaderLock, err := newLeaderElectionResourceLock(ctx, cfg, kubeinfo, leasename)
	if err != nil {
		return err
	}

	var mu sync.Mutex
	leaderElector, err := k8sLeaderElection.NewLeaderElector(k8sLeaderElection.LeaderElectionConfig{
		Lock: leaderLock,
		// The ratios between LeaseDuration:RenewDeadline:RetryPeriod
		// mimic the ratios used in kubernetes.git.
		LeaseDuration: leaseDuration,
		RenewDeadline: 2 * leaseDuration / 3,
		RetryPeriod:   2 * leaseDuration / 15,
		//WatchDog: TODO, // XXX: this could be a robustness win
		Callbacks: k8sLeaderElection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// ctx will be canceled when we are no longer the leader (or the parent
				// context has been canceled; i.e. we are shutting down).

				// leaderElector.Run() doesn't wait for the OnStartedLeading callback to
				// return, so because this callback function may not return immediately when
				// ctx is canceled, we have to serialize with ourself.
				mu.Lock()
				defer mu.Unlock()

				fn(ctx)
			},
			// client-go requires that we provide an OnStoppedLeading callback,
			// even if there's nothing to do.  *sigh*
			OnStoppedLeading: func() {},
		},
	})
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			leaderElector.Run(ctx)
		}
	}
}
