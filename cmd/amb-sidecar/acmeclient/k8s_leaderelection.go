package acmeclient

import (
	"os"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/cmd/amb-sidecar/events"
	"github.com/datawire/apro/cmd/amb-sidecar/types"

	// k8s types
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s clients
	k8sClientCoordinationV1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// k8s misc
	k8sLeaderElectionResourceLock "k8s.io/client-go/tools/leaderelection/resourcelock"
)

func GetLeaderElectionResourceLock(cfg types.Config, kubeinfo *k8s.KubeInfo, eventLogger *events.EventLogger) (k8sLeaderElectionResourceLock.Interface, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	client, err := k8s.NewClient(kubeinfo)
	if err != nil {
		return nil, err
	}

	restconfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return nil, err
	}

	if _, err := client.ResolveResourceType("Lease.v1.coordination.k8s.io"); err != nil {
		// Kubernetes <1.12 didn't have a "Lease" object, and was in beta in <1.14, so fall back to
		// using an Endpoints object.  Don't consider v1beta1 to be good-enough; it isn't for our copy
		// of client-go; require v1.
		coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
		if err != nil {
			return nil, err
		}
		return &k8sLeaderElectionResourceLock.EndpointsLock{
			EndpointsMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      "acmeclient",
				Namespace: cfg.AmbassadorNamespace,
			},
			Client: coreClient,
			LockConfig: k8sLeaderElectionResourceLock.ResourceLockConfig{
				Identity:      hostname,
				EventRecorder: eventLogger.Namespace(cfg.AmbassadorNamespace), // must match the namespace of the EndpointsMeta above
			},
		}, nil
	}

	coordinationClient, err := k8sClientCoordinationV1.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &k8sLeaderElectionResourceLock.LeaseLock{
		LeaseMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      "acmeclient",
			Namespace: cfg.AmbassadorNamespace,
		},
		Client: coordinationClient,
		LockConfig: k8sLeaderElectionResourceLock.ResourceLockConfig{
			Identity:      hostname,
			EventRecorder: eventLogger.Namespace(cfg.AmbassadorNamespace), // must match the namespace of the LeaseMeta above
		},
	}, nil
}
