package entrypoint

import (
	"context"
	//"github.com/datawire/dlib/derror"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
)

// This is a gross hack to remove all AuthServices using protocol_version: v2 only when running Edge-Stack and then inject an
// AuthService with protocol_version: v3 if needed. The purpose of this hack is to prevent Edge-Stack 2.3 from
// using any other AuthService than the default one running as part of amb-sidecar and force the protocol version to v3.
func ReconcileAuthServices(ctx context.Context, sh *SnapshotHolder, deltas *[]*kates.Delta) error {
	// We only want to remove AuthServices if this is an instance of Edge-Stack
	isEdgeStack, err := IsEdgeStack()
	if err != nil {
		return err
	} else if !isEdgeStack {
		dlog.Infof(ctx, "[debug-authservice] Not reconciling AuthServices, not an Edge Stack Install")
		return nil
	}
	dlog.Infof(ctx, "[debug-authservice] Is Edge Stack, begin AuthService proto check")

	syntheticAuth := &v3alpha1.AuthService{
		TypeMeta: kates.TypeMeta{
			Kind:       "AuthService",
			APIVersion: "getambassador.io/v3alpha1",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "synthetic-edge-stack-auth",
			Namespace: "default",
		},
		Spec: v3alpha1.AuthServiceSpec{
			AuthService:     "127.0.0.1:8500",
			Proto:           "grpc",
			ProtocolVersion: "v3",
		},
	}

	var authServices []*v3alpha1.AuthService
	syntheticAuthExists := false
	for _, authService := range sh.k8sSnapshot.AuthServices {
		// Keep any AuthServices already using protocol_version: v3
		if authService.ObjectMeta.Name == "synthetic-edge-stack-auth" {
			dlog.Infof(ctx, "[debug-authservice] The Synthetic AuthService exists in the snapshot already")
			syntheticAuthExists = true
		} else if authService.Spec.ProtocolVersion == "v3" {
			dlog.Infof(ctx, "[debug-authservice] Found an AuthService in the snapshot with proto v3 that is NOT the synthetic AuthService: %v", authService)
			authServices = append(authServices, authService)
		}
	}
	if len(authServices) == 0 && !syntheticAuthExists {
		dlog.Infof(ctx, "[debug-authservice] Did not find any AuthServices with proto v3. Injecting synthetic AuthService...")
		// There are no valid AuthServices with protocol_version: v3. A synthetic one needs to be injected.
		authServices = append(authServices, syntheticAuth)

		// loop through the deltas and remove any AuthService deltas adding other AuthServices before the Synthetic delta is inserted
		var newDeltas []*kates.Delta
		for _, delta := range *deltas {
			// Keep all the deltas that are not for AuthServices. The AuthService deltas can be kept as long as they are not an add delta.
			if (delta.Kind != "AuthService") || (delta.Kind == "AuthService" && delta.DeltaType != kates.ObjectAdd) {
				if delta.Kind == "AuthService" {
					dlog.Infof(ctx, "[debug-authservice] Keeping Delta for AuthService: %v", delta)
				}
				newDeltas = append(newDeltas, delta)
			} else if delta.Kind == "AuthService" {
				dlog.Infof(ctx, "[debug-authservice] Discarding Delta for AuthService: %v", delta)
			}
		}
		newDeltas = append(newDeltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectAdd,
		})

		*deltas = newDeltas
		dlog.Infof(ctx, "[debug-authservice] AuthServices after Synthetic Injection: %v", authServices)
		sh.k8sSnapshot.AuthServices = authServices
	} else if len(authServices) >= 1 && syntheticAuthExists {
		dlog.Infof(ctx, "[debug-authservice] Valid AuthServices are present and the Synthetic Auth needs to be removed: %v", authServices)

		// One or more Valid AuthServices are present. The synthetic AuthService exists and needs to be removed now.
		sh.k8sSnapshot.AuthServices = authServices
		var newDeltas []*kates.Delta
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectDelete,
		})
		dlog.Infof(ctx, "[debug-authservice] Deltas with Synthetic removal: %v", newDeltas)
		*deltas = newDeltas
	} else {
		dlog.Infof(ctx, "[debug-authservice] Valid AuthServices are present and the Synthetic Auth DOES NOT needs to be removed")
	}

	return nil
}
