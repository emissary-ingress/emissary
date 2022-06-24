package entrypoint

import (
	"context"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/emissaryutil"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// Iterates over the annotations in a snapshot to check if any AuthServices are present.
func annotationsContainAuthService(annotations map[string]snapshot.AnnotationList) bool {
	for _, list := range annotations {
		for _, obj := range list {
			switch obj.(type) {
			case *v3alpha1.AuthService:
				return true
			default:
				continue
			}
		}
	}
	return false
}

// Checks if the provided string is a loopback IP address with port 8500
func IsLocalhost8500(svcStr string) bool {
	_, hostname, port, err := emissaryutil.ParseServiceName(svcStr)
	return err == nil && port == 8500 && emissaryutil.IsLocalhost(hostname)
}

// This is a gross hack to remove all AuthServices using protocol_version: v2 only when running Edge-Stack and then inject an
// AuthService with protocol_version: v3 if needed. The purpose of this hack is to prevent Edge-Stack 2.3 from
// using any other AuthService than the default one running as part of amb-sidecar and force the protocol version to v3.
func ReconcileAuthServices(ctx context.Context, sh *SnapshotHolder, deltas *[]*kates.Delta) error {
	// We only want to remove AuthServices if this is an instance of Edge-Stack
	isEdgeStack, err := IsEdgeStack()
	if err != nil {
		return err
	} else if !isEdgeStack {
		dlog.Debugf(ctx, "[WATCHER]: Is not EdgeStack so not going to reconcile Services")
		return nil
	}
	// We also dont want to do anything with AuthServices if the Docker demo mode is running
	if envbool("AMBASSADOR_DEMO_MODE") {
		return nil
	}

	// Construct a synthetic AuthService to be injected if we dont find any valid AuthServices
	injectSyntheticAuth := true
	syntheticAuth := &v3alpha1.AuthService{
		TypeMeta: kates.TypeMeta{
			Kind:       "AuthService",
			APIVersion: "getambassador.io/v3alpha1",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "synthetic-edge-stack-auth",
			Namespace: GetAmbassadorNamespace(),
		},
		Spec: v3alpha1.AuthServiceSpec{
			AuthService:     "127.0.0.1:8500",
			Proto:           "grpc",
			ProtocolVersion: "v3",
			AmbassadorID:    []string{"_automatic_"},
		},
	}

	var authServices []*v3alpha1.AuthService
	syntheticAuthExists := false

	dlog.Debugf(ctx, "[WATCHER]: Number of AuthServices in snapshot - %d", len(sh.k8sSnapshot.AuthServices))
	for _, authService := range sh.k8sSnapshot.AuthServices {
		// check if the AuthService points at 127.0.0.1:8500 (edge-stack)
		if IsLocalhost8500(authService.Spec.AuthService) {
			// If it does point at localhost, make sure it is v3, otherwise we need to inject the synthetic AuthService
			if authService.Spec.ProtocolVersion == "v3" {
				injectSyntheticAuth = false
				if authService.ObjectMeta.Name == "synthetic-edge-stack-auth" {
					syntheticAuthExists = true
				} else {
					authServices = append(authServices, authService)
				}
			} else {
				// In the event that there is an AuthService that does not have protocol_version: v3
				// Then we use the spec of that AuthService as the Synthetic v3 AuthService we will inject later
				syntheticAuth.Spec = authService.Spec
				syntheticAuth.Spec.ProtocolVersion = "v3"
			}
		} else {
			dlog.Debugf(ctx, "[WATCHER]: Keeping custom Auth service name: %s namespace: %s", authService.ObjectMeta.Name, authService.ObjectMeta.Namespace)
			// By default we keep any custom AuthServices that do not point at localhost
			authServices = append(authServices, authService)
			injectSyntheticAuth = false
		}
	}

	// TODO if there are v3 authServices, still remove any that are not `v3`

	// Also loop over the annotations and remove authservices that are not v3. We do
	// this by looping over each entry in the annotations map, removing all the non-v3
	// AuthService entries, and then removing any keys that end up with empty lists.

	// OK. Loop over all the keys and their corrauthServicesesponding lists of annotations...
	if annotationsContainAuthService(sh.k8sSnapshot.Annotations) {
		for key, list := range sh.k8sSnapshot.Annotations {
			// ...and build up our edited list of things.
			editedList := snapshot.AnnotationList{}

			for _, obj := range list {
				switch annotationObj := obj.(type) {
				case *v3alpha1.AuthService:
					if IsLocalhost8500(annotationObj.Spec.AuthService) {
						// If it does point at localhost, make sure it is v3, otherwise we need to inject the synthetic AuthService
						if annotationObj.Spec.ProtocolVersion == "v3" {
							injectSyntheticAuth = false
							if annotationObj.ObjectMeta.Name == "synthetic-edge-stack-auth" {
								syntheticAuthExists = true
							} else {
								authServices = append(authServices, annotationObj)
								editedList = append(editedList, annotationObj)
							}
						} else {
							// In the event that there is an AuthService that does not have protocol_version: v3
							// Then we use the spec of that AuthService as the Synthetic v3 AuthService we will inject later
							syntheticAuth.Spec = annotationObj.Spec
							syntheticAuth.Spec.ProtocolVersion = "v3"
						}
					} else {
						// By default we keep any custom AuthServices that do not point at localhost
						authServices = append(authServices, annotationObj)
						editedList = append(editedList, annotationObj)
						injectSyntheticAuth = false
					}
				default:
					// This isn't an AuthService at all, so we'll keep it.
					editedList = append(editedList, annotationObj)
				}
			}

			// Once here, is our editedList is empty?
			if len(editedList) == 0 {
				// Yes. Delete the whole key for this list.
				delete(sh.k8sSnapshot.Annotations, key)
			} else {
				// Nope, not empty. Save the edited list.
				sh.k8sSnapshot.Annotations[key] = editedList
			}
		}
	}

	if injectSyntheticAuth {
		dlog.Debugf(ctx, "[WATCHER]: No valid AuthServices with protocol_version: v3 detected, injecting Synthetic AuthService")
		// There are no valid AuthServices with protocol_version: v3. A synthetic one needs to be injected.
		authServices = append(authServices, syntheticAuth)

		// loop through the deltas and remove any AuthService deltas adding other AuthServices before the Synthetic delta is inserted
		var newDeltas []*kates.Delta
		for _, delta := range *deltas {
			// Keep all the deltas that are not for AuthServices. The AuthService deltas can be kept as long as they are not an add delta.
			if (delta.Kind != "AuthService") || (delta.Kind == "AuthService" && delta.DeltaType != kates.ObjectAdd) {
				newDeltas = append(newDeltas, delta)
			}
		}
		newDeltas = append(newDeltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectAdd,
		})

		*deltas = newDeltas
		sh.k8sSnapshot.AuthServices = authServices
	} else if len(authServices) >= 1 {
		dlog.Debugf(ctx, "[WATCHER] setting authservices back to list of customer services.")
		// Write back the list of valid AuthServices.
		sh.k8sSnapshot.AuthServices = authServices

		// The synthetic AuthService needs to be removed since one or more valid AuthServices are present.
		if syntheticAuthExists {
			dlog.Debugf(ctx, "[WATCHER]: Valid AuthServices using protocol_version: v3 detected alongside the Synthetic AuthService, removing Synthetic...")
			// One or more Valid AuthServices are present. The synthetic AuthService exists and needs to be removed now.
			var newDeltas []*kates.Delta
			*deltas = append(*deltas, &kates.Delta{
				TypeMeta:   syntheticAuth.TypeMeta,
				ObjectMeta: syntheticAuth.ObjectMeta,
				DeltaType:  kates.ObjectDelete,
			})

			*deltas = newDeltas
		}
	}

	return nil
}
