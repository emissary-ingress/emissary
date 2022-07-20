package entrypoint

import (
	"context"
	"fmt"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/emissaryutil"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

// Checks if the provided string is a loopback IP address with port 8500
func IsLocalhost8500(svcStr string) bool {
	_, hostname, port, err := emissaryutil.ParseServiceName(svcStr)
	return err == nil && port == 8500 && emissaryutil.IsLocalhost(hostname)
}

func iterateOverAuthServices(sh *SnapshotHolder, cb func(
	authService *v3alpha1.AuthService, // duh
	name string, // name to unambiguously refer to the authService by; might be more complex than "name.namespace" if it's an annotation
	parentName string, // name of the thing that the annotation is on (or empty if not an annotation)
	idx int, // index of the authService; either in sh.k8sSnapshot.AuthServices or in sh.k8sSnapshot.Annotations[parentName]
)) {
	envAmbID := GetAmbassadorID()

	for i, authService := range sh.k8sSnapshot.AuthServices {
		if authService.Spec.AmbassadorID.Matches(envAmbID) {
			name := authService.TypeMeta.Kind + "/" + authService.ObjectMeta.Name + "." + authService.ObjectMeta.Namespace
			cb(authService, name, "", i)
		}
	}
	for parentName, list := range sh.k8sSnapshot.Annotations {
		for i, obj := range list {
			if authService, ok := obj.(*v3alpha1.AuthService); ok && authService.Spec.AmbassadorID.Matches(envAmbID) {
				name := fmt.Sprintf("%s#%d", parentName, i)
				cb(authService, name, parentName, i)
			}
		}
	}
}

// This is a gross hack to remove all AuthServices using protocol_version: v2 only when running Edge-Stack and then inject an
// AuthService with protocol_version: v3 if needed. The purpose of this hack is to prevent Edge-Stack 2.3 from
// using any other AuthService than the default one running as part of amb-sidecar and force the protocol version to v3.
func ReconcileAuthServices(ctx context.Context, sh *SnapshotHolder, deltas *[]*kates.Delta) error {
	// We only want to remove AuthServices if this is an instance of Edge-Stack
	if isEdgeStack, err := IsEdgeStack(); err != nil {
		return fmt.Errorf("ReconcileAuthServices: %w", err)
	} else if !isEdgeStack {
		return nil
	}

	// using a name with underscores prevents it from colliding with anything real in the
	// cluster--Kubernetes resources can't have underscores in their name.
	const syntheticAuthServiceName = "synthetic_edge_stack_auth"

	var (
		numAuthServices  uint64
		syntheticAuth    *v3alpha1.AuthService
		syntheticAuthIdx int
	)
	iterateOverAuthServices(sh, func(authService *v3alpha1.AuthService, name, parentName string, i int) {
		numAuthServices++
		if IsLocalhost8500(authService.Spec.AuthService) {
			if parentName == "" && authService.ObjectMeta.Name == syntheticAuthServiceName {
				syntheticAuth = authService
				syntheticAuthIdx = i
			}
			if authService.Spec.ProtocolVersion != "v3" {
				// Force the Edge Stack AuthService to be protocol_version=v3.  This
				// is important so that <2.3 and >=2.3 installations can coexist.
				// This is important, because for zero-downtime upgrades, they must
				// coexist briefly while the new Deployment is getting rolled out.
				dlog.Debugf(ctx, "ReconcileAuthServices: Forcing protocol_version=v3 on %s", name)
				authService.Spec.ProtocolVersion = "v3"
			}
		}
	})

	switch {
	case numAuthServices == 0: // add the synthetic auth service
		dlog.Debug(ctx, "ReconcileAuthServices: No user-provided AuthServices detected; injecting synthetic AuthService")
		syntheticAuth = &v3alpha1.AuthService{
			TypeMeta: kates.TypeMeta{
				Kind:       "AuthService",
				APIVersion: "getambassador.io/v3alpha1",
			},
			ObjectMeta: kates.ObjectMeta{
				Name:      syntheticAuthServiceName,
				Namespace: GetAmbassadorNamespace(),
			},
			Spec: v3alpha1.AuthServiceSpec{
				AmbassadorID:    []string{GetAmbassadorID()},
				AuthService:     "127.0.0.1:8500",
				Proto:           "grpc",
				ProtocolVersion: "v3",
			},
		}
		sh.k8sSnapshot.AuthServices = append(sh.k8sSnapshot.AuthServices, syntheticAuth)
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectAdd,
		})
	case numAuthServices > 1 && syntheticAuth != nil: // remove the synthetic auth service
		dlog.Debugf(ctx, "ReconcileAuthServices: %d user-provided AuthServices detected; removing synthetic AuthService", numAuthServices-1)
		sh.k8sSnapshot.AuthServices = append(
			sh.k8sSnapshot.AuthServices[:syntheticAuthIdx],
			sh.k8sSnapshot.AuthServices[syntheticAuthIdx+1:]...)
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectDelete,
		})
	}

	return nil
}
