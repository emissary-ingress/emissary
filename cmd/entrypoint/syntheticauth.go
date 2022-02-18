package entrypoint

import (
	"context"
	//"github.com/datawire/dlib/derror"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
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
		return nil
	}

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
	for _, authService := range sh.k8sSnapshot.AuthServices {
		// Keep any AuthServices already using protocol_version: v3
		if authService.ObjectMeta.Name == "synthetic-edge-stack-auth" {
			syntheticAuthExists = true
		} else if authService.Spec.ProtocolVersion == "v3" {
			authServices = append(authServices, authService)
		}
	}

	// Also loop over the annotations and remove authservices that are not v3. We do
	// this by looping over each entry in the annotations map, removing all the non-v3
	// AuthService entries, and then removing any keys that end up with empty lists.
	//
	// keysToDelete tracks keys to delete, since I never trust deleting things from
	// collections while iterating over them.
	keysToDelete := make([]string, 0)

	// OK. Loop over all the keys and their corresponding lists of annotations...
	for key, list := range sh.k8sSnapshot.Annotations {
		// ...and build up our edited list of things.
		editedList := snapshot.AnnotationList{}

		for _, obj := range list {
			un := obj.(*kates.Unstructured)

			if un.GetKind() != "AuthService" {
				// This isn't an AuthService at all, so we'll keep it.
				//
				// XXX There's optimization to do here: if there are no AuthServices
				// in our list, we needn't edit it at all. Something to circle back to
				// later.
				editedList = append(editedList, un)
			} else {
				// This _is_ an AuthService, so we'll check its protocol version.
				// Anything other than v3 gets tossed.
				//
				// Note also that AuthServices other than getambassador.io/v3 cannot
				// have a protocol_version, but whatever -- that'll be something not
				// equal to v3, so it'll get tossed, and that's what we want.
				spec := un.Object["spec"].(map[string]interface{})

				if spec["protocol_version"] == "v3" {
					// Whoa, it's a v3! Keep it.
					editedList = append(editedList, un)
				}
			}
		}

		// Once here, is our editedList is empty?
		if len(editedList) == 0 {
			// Yes. Delete the whole key for this list.
			keysToDelete = append(keysToDelete, key)
		} else {
			// Nope, not empty. Save the edited list.
			sh.k8sSnapshot.Annotations[key] = editedList
		}
	}

	// Finally, delete any keys that ended up with empty lists.
	for _, key := range keysToDelete {
		delete(sh.k8sSnapshot.Annotations, key)
	}

	if len(authServices) == 0 && !syntheticAuthExists {
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
	} else if len(authServices) >= 1 && syntheticAuthExists {
		// One or more Valid AuthServices are present. The synthetic AuthService exists and needs to be removed now.
		sh.k8sSnapshot.AuthServices = authServices
		var newDeltas []*kates.Delta
		*deltas = append(*deltas, &kates.Delta{
			TypeMeta:   syntheticAuth.TypeMeta,
			ObjectMeta: syntheticAuth.ObjectMeta,
			DeltaType:  kates.ObjectDelete,
		})

		*deltas = newDeltas
	}

	return nil
}
