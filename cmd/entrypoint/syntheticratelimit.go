package entrypoint

import (
	"context"
	"fmt"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func iterateOverRateLimitServices(sh *SnapshotHolder, cb func(
	rateLimitService *v3alpha1.RateLimitService, // rateLimitService
	name string, // name to unambiguously refer to the rateLimitServices by; might be more complex than "name.namespace" if it's an annotation
	parentName string, // name of the thing that the annotation is on (or empty if not an annotation)
	idx int, // index of the rateLimitService; either in sh.k8sSnapshot.RateLimitServices or in sh.k8sSnapshot.Annotations[parentName]
)) {
	envAmbID := GetAmbassadorID()

	for i, rateLimitService := range sh.k8sSnapshot.RateLimitServices {
		if rateLimitService.Spec.AmbassadorID.Matches(envAmbID) {
			name := rateLimitService.TypeMeta.Kind + "/" + rateLimitService.ObjectMeta.Name + "." + rateLimitService.ObjectMeta.Namespace
			cb(rateLimitService, name, "", i)
		}
	}

	for parentName, list := range sh.k8sSnapshot.Annotations {
		for i, obj := range list {
			if rateLimitService, ok := obj.(*v3alpha1.RateLimitService); ok && rateLimitService.Spec.AmbassadorID.Matches(envAmbID) {
				name := fmt.Sprintf("%s#%d", parentName, i)
				cb(rateLimitService, name, parentName, i)
			}
		}
	}
}

// ReconcileRateLimit is a hack to remove all RateLimitService using protocol_version: v2 only when running Edge-Stack and then inject an
// RateLimitService with protocol_version: v3 if needed. The purpose of this hack is to prevent Edge-Stack 2.3 from
// using any other RateLimitService than the default one running as part of amb-sidecar and force the protocol version to v3.
func ReconcileRateLimit(ctx context.Context, sh *SnapshotHolder, deltas *[]*kates.Delta) error {
	// We only want to remove RateLimitServices if this is an instance of Edge-Stack
	if isEdgeStack, err := IsEdgeStack(); err != nil {
		return fmt.Errorf("ReconcileRateLimitServices: %w", err)
	} else if !isEdgeStack {
		return nil
	}

	// using a name with underscores prevents it from colliding with anything real in the
	// cluster--Kubernetes resources can't have underscores in their name.
	const syntheticRateLimitServiceName = "synthetic_edge_stack_rate_limit"

	var (
		numRateLimitServices  uint64
		syntheticRateLimit    *v3alpha1.RateLimitService
		syntheticRateLimitIdx int
	)

	iterateOverRateLimitServices(sh, func(rateLimitService *v3alpha1.RateLimitService, name, parentName string, i int) {
		numRateLimitServices++
		if IsLocalhost8500(rateLimitService.Spec.Service) {
			if parentName == "" && rateLimitService.ObjectMeta.Name == syntheticRateLimitServiceName {
				syntheticRateLimit = rateLimitService
				syntheticRateLimitIdx = i
			}
			if rateLimitService.Spec.ProtocolVersion != "v3" {
				// Force the Edge Stack RateLimitService to be protocol_version=v3.  This
				// is important so that <2.3 and >=2.3 installations can coexist.
				// This is important, because for zero-downtime upgrades, they must
				// coexist briefly while the new Deployment is getting rolled out.
				dlog.Debugf(ctx, "ReconcileRateLimitServices: Forcing protocol_version=v3 on %s", name)
				rateLimitService.Spec.ProtocolVersion = "v3"
			}
		}
	})

	switch {
	case numRateLimitServices == 0: // add the synthetic rate limit service
		dlog.Debug(ctx, "ReconcileRateLimitServices: No user-provided RateLimitServices detected; injecting synthetic RateLimitService")
		syntheticRateLimit = &v3alpha1.RateLimitService{
			TypeMeta: kates.TypeMeta{
				Kind:       "RateLimitService",
				APIVersion: "getambassador.io/v3alpha1",
			},
			ObjectMeta: kates.ObjectMeta{
				Name:      syntheticRateLimitServiceName,
				Namespace: GetAmbassadorNamespace(),
			},
			Spec: v3alpha1.RateLimitServiceSpec{
				AmbassadorID:    []string{GetAmbassadorID()},
				Service:         "127.0.0.1:8500",
				ProtocolVersion: "v3",
			},
		}
		sh.k8sSnapshot.RateLimitServices = append(sh.k8sSnapshot.RateLimitServices, syntheticRateLimit)
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticRateLimit.TypeMeta,
			ObjectMeta: syntheticRateLimit.ObjectMeta,
			DeltaType:  kates.ObjectAdd,
		})
	case numRateLimitServices > 1 && syntheticRateLimit != nil: // remove the synthetic rate limit service
		dlog.Debugf(ctx, "ReconcileRateLimitServices: %d user-provided RateLimitServices detected; removing synthetic RateLimitServices", numRateLimitServices-1)
		sh.k8sSnapshot.RateLimitServices = append(
			sh.k8sSnapshot.RateLimitServices[:syntheticRateLimitIdx],
			sh.k8sSnapshot.RateLimitServices[syntheticRateLimitIdx+1:]...)
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticRateLimit.TypeMeta,
			ObjectMeta: syntheticRateLimit.ObjectMeta,
			DeltaType:  kates.ObjectDelete,
		})
	}

	return nil
}
