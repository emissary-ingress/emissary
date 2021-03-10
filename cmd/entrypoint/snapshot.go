package entrypoint

import (
	"encoding/json"
	"fmt"
	"log"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
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
	case *amb.Host:
		var id amb.AmbassadorID
		if r.Spec != nil {
			if len(r.Spec.AmbassadorID) > 0 {
				id = r.Spec.AmbassadorID
			} else {
				id = r.Spec.DeprecatedAmbassadorID
			}
		}
		return id

	case *amb.Mapping:
		return r.Spec.AmbassadorID
	case *amb.TCPMapping:
		return r.Spec.AmbassadorID
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
			log.Printf("%s: error parsing ambassador-id '%s'", location(resource), idstr)
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
