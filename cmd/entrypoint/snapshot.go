package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// NewKubernetesSnapshot creates a new, empty set of Ambassador inputs.
func NewKubernetesSnapshot() *snapshotTypes.KubernetesSnapshot {
	a := &snapshotTypes.KubernetesSnapshot{}
	a.FSSecrets = make(map[snapshotTypes.SecretRef]*kates.Secret)

	return a
}

// GetAmbId extracts the AmbassadorId from the kubernetes resource.
func GetAmbId(resource kates.Object) amb.AmbassadorID {
	switch r := resource.(type) {
	case *v3alpha1.Host:
		var id amb.AmbassadorID
		if r.Spec != nil {
			if len(r.Spec.AmbassadorID) > 0 {
				id = amb.AmbassadorID(r.Spec.AmbassadorID)
			}
		}
		return id

	case *v3alpha1.Mapping:
		return amb.AmbassadorID(r.Spec.AmbassadorID)
	case *v3alpha1.TCPMapping:
		return amb.AmbassadorID(r.Spec.AmbassadorID)
	case *amb.Module:
		return r.Spec.AmbassadorID
	case *amb.TLSContext:
		return r.Spec.AmbassadorID
	case *amb.AuthService:
		return r.Spec.AmbassadorID
	case *amb.RateLimitService:
		return r.Spec.AmbassadorID
	case *amb.LogService:
		return r.Spec.AmbassadorID
	case *amb.TracingService:
		return r.Spec.AmbassadorID
	case *amb.DevPortal:
		return r.Spec.AmbassadorID
	case *amb.ConsulResolver:
		return r.Spec.AmbassadorID
	case *amb.KubernetesEndpointResolver:
		return r.Spec.AmbassadorID
	case *amb.KubernetesServiceResolver:
		return r.Spec.AmbassadorID
	}

	ann := resource.GetAnnotations()
	idstr, ok := ann["getambassador.io/ambassador-id"]
	if ok {
		var id amb.AmbassadorID
		err := json.Unmarshal([]byte(idstr), &id)
		if err != nil {
			dlog.Errorf(context.TODO(), "%s: error parsing ambassador-id '%s'", location(resource), idstr)
		} else {
			return id
		}
	}

	return amb.AmbassadorID{}
}

func location(obj kates.Object) string {
	return fmt.Sprintf("%s %s in namespace %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(),
		obj.GetNamespace())
}
