package entrypoint

import (
	"context"
	"strings"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

// ReconcileSecrets figures out which secrets we're actually using,
// since we don't want to send secrets to Ambassador unless we're
// using them, since any secret we send will be saved to disk.
func ReconcileSecrets(ctx context.Context, s *snapshotTypes.KubernetesSnapshot) {
	// Start by building up a list of all the K8s objects that are
	// allowed to mention secrets. Note that we vet the ambassador_id
	// for all of these before putting them on the list.
	var resources []kates.Object

	// Annotations are straightforward, although honestly we should
	// be filtering annotations by type here (or, even better, unfold
	// them earlier so that we can treat them like any other resource
	// here).

	for _, a := range s.Annotations {
		if include(GetAmbId(ctx, a)) {
			resources = append(resources, a)
		}
	}

	// Hosts are a little weird, because we have two ways to find the
	// ambassador_id. Sorry about that.
	for _, h := range s.Hosts {
		var id amb.AmbassadorID
		if len(h.Spec.AmbassadorID) > 0 {
			id = amb.AmbassadorID(h.Spec.AmbassadorID)
		}
		if include(id) {
			resources = append(resources, h)
		}
	}

	// TLSContexts, Modules, and Ingresses are all straightforward.
	for _, t := range s.TLSContexts {
		if include(t.Spec.AmbassadorID) {
			resources = append(resources, t)
		}
	}
	for _, m := range s.Modules {
		if include(m.Spec.AmbassadorID) {
			resources = append(resources, m)
		}
	}
	for _, i := range s.Ingresses {
		resources = append(resources, i)
	}

	// OK. Once that's done, we can check to see if we should be
	// doing secret namespacing or not -- this requires a look into
	// the Ambassador Module, if it's present.
	//
	// XXX Linear searches suck, but whatever, it's just not gonna
	// be all that many things. We won't bother optimizing this unless
	// a profiler shows that it's a problem.

	secretNamespacing := true
	for _, resource := range resources {
		mod, ok := resource.(*amb.Module)
		// We don't need to recheck ambassador_id on this Module because
		// the Module can't have made it into the resources list without
		// its ambassador_id being checked.

		if ok && mod.GetName() == "ambassador" {
			// XXX ModuleSecrets is a _godawful_ hack. See the comment on
			// ModuleSecrets itself for more.
			secs := ModuleSecrets{}
			err := convert(mod.Spec.Config, &secs)
			if err != nil {
				dlog.Errorf(ctx, "error parsing module: %v", err)
				continue
			}
			secretNamespacing = secs.Defaults.TLSSecretNamespacing
			break
		}
	}

	// Once we have our list of secrets, go figure out the names of all
	// the secrets we need. We'll use this "refs" map to hold all the names...
	refs := map[snapshotTypes.SecretRef]bool{}

	// ...and, uh, this "action" function is really just a closure to avoid
	// needing to pass "refs" to find SecretRefs. Shrug. Arguably more
	// complex than needed, but meh.
	action := func(ref snapshotTypes.SecretRef) {
		refs[ref] = true
	}

	// So. Walk the list of resources...
	for _, resource := range resources {
		// ...and for each resource, dig out any secrets being referenced.
		findSecretRefs(ctx, resource, secretNamespacing, action)
	}

	// We _always_ have an implicit references to the fallback cert secret...
	secretRef(GetAmbassadorNamespace(), "fallback-self-signed-cert", false, action)

	if IsEdgeStack() {
		// ...and for Edge Stack, we _always_ have an implicit reference to the
		// license secret.
		secretRef(GetLicenseSecretNamespace(), GetLicenseSecretName(), false, action)
	}

	// OK! After all that, go copy all the matching secrets from FSSecrets and
	// K8sSecrets to Secrets.
	//
	// The way this works is kind of simple: first we check everything in
	// FSSecrets. Then, when we check K8sSecrets, we skip any secrets that are
	// also in FSSecrets. End result: FSSecrets wins if there are any conflicts.
	s.Secrets = make([]*kates.Secret, 0, len(refs))

	for ref, secret := range s.FSSecrets {
		if refs[ref] {
			dlog.Debugf(ctx, "Taking FSSecret %#v", ref)
			s.Secrets = append(s.Secrets, secret)
		}
	}

	for _, secret := range s.K8sSecrets {
		ref := snapshotTypes.SecretRef{Namespace: secret.GetNamespace(), Name: secret.GetName()}

		_, found := s.FSSecrets[ref]
		if found {
			dlog.Debugf(ctx, "Conflict! skipping K8sSecret %#v", ref)
			continue
		}

		if refs[ref] {
			dlog.Debugf(ctx, "Taking K8sSecret %#v", ref)
			s.Secrets = append(s.Secrets, secret)
		}
	}
}

// Find all the secrets a given Ambassador resource references.
func findSecretRefs(ctx context.Context, resource kates.Object, secretNamespacing bool, action func(snapshotTypes.SecretRef)) {
	switch r := resource.(type) {
	case *v3alpha1.Host:
		// The Host resource is a little odd. Host.spec.tls, Host.spec.tlsSecret, and
		// host.spec.acmeProvider.privateKeySecret can all refer to secrets.
		if r.Spec == nil {
			return
		}

		if r.Spec.TLS != nil {
			// Host.spec.tls.caSecret is the thing to worry about here.
			secretRef(r.GetNamespace(), r.Spec.TLS.CASecret, secretNamespacing, action)
		}

		// Host.spec.tlsSecret and Host.spec.acmeProvider.privateKeySecret are native-Kubernetes-style
		// `core.v1.LocalObjectReference`s, not Ambassador-style `{name}.{namespace}` strings.  If we
		// ever decide that they should support cross-namespace references, we would do it by adding a
		// `namespace:` field (i.e. changing them to `core.v1.SecretReference`s) rather than by
		// adopting the `{name}.{namespace}` notation.
		if r.Spec.TLSSecret != nil && r.Spec.TLSSecret.Name != "" {
			secretRef(r.GetNamespace(), r.Spec.TLSSecret.Name, false, action)
		}

		if r.Spec.AcmeProvider != nil && r.Spec.AcmeProvider.PrivateKeySecret != nil &&
			r.Spec.AcmeProvider.PrivateKeySecret.Name != "" {
			secretRef(r.GetNamespace(), r.Spec.AcmeProvider.PrivateKeySecret.Name, false, action)
		}

	case *amb.TLSContext:
		// TLSContext.spec.secret and TLSContext.spec.ca_secret are the things to worry about --
		// but note well that TLSContexts can override the global secretNamespacing setting.
		if r.Spec.Secret != "" {
			if r.Spec.SecretNamespacing != nil {
				secretNamespacing = *r.Spec.SecretNamespacing
			}
			secretRef(r.GetNamespace(), r.Spec.Secret, secretNamespacing, action)
		}

		if r.Spec.CASecret != "" {
			if r.Spec.SecretNamespacing != nil {
				secretNamespacing = *r.Spec.SecretNamespacing
			}
			secretRef(r.GetNamespace(), r.Spec.CASecret, secretNamespacing, action)
		}

	case *amb.Module:
		// This whole thing is a hack. We probably _should_ check to make sure that
		// this is an Ambassador Module or a TLS Module, but, well, those're the only
		// supported kinds now, anyway...
		//
		// XXX ModuleSecrets is a godawful hack. See its comment for more.
		secs := ModuleSecrets{}
		err := convert(r.Spec.Config, &secs)
		if err != nil {
			// XXX
			dlog.Errorf(ctx, "error extracting secrets from module: %v", err)
			return
		}

		// XXX Technically, this is wrong -- _any_ element named in the module can
		// refer to a secret. Hmmm.
		if secs.Upstream.Secret != "" {
			secretRef(r.GetNamespace(), secs.Upstream.Secret, secretNamespacing, action)
		}
		if secs.Server.Secret != "" {
			secretRef(r.GetNamespace(), secs.Server.Secret, secretNamespacing, action)
		}
		if secs.Client.Secret != "" {
			secretRef(r.GetNamespace(), secs.Client.Secret, secretNamespacing, action)
		}

	case *kates.Ingress:
		// Ingress is pretty straightforward, too, just look in spec.tls.
		for _, itls := range r.Spec.TLS {
			if itls.SecretName != "" {
				secretRef(r.GetNamespace(), itls.SecretName, secretNamespacing, action)
			}
		}
	}
}

// Mark a secret as one we reference, handling secretNamespacing correctly.
func secretRef(namespace, name string, secretNamespacing bool, action func(snapshotTypes.SecretRef)) {
	if secretNamespacing {
		parts := strings.Split(name, ".")
		if len(parts) > 1 {
			namespace = parts[len(parts)-1]
			name = strings.Join(parts[:len(parts)-1], ".")
		}
	}

	action(snapshotTypes.SecretRef{Namespace: namespace, Name: name})
}

// ModuleSecrets is... a hack. It's sort of a mashup of the chunk of the Ambassador
// Module and the chunk of the TLS Module that are common, because they're able to
// specify secrets. However... first, I don't think the TLS Module actually supported
// tls_secret_namespacing. Second, the Ambassador Module at least supports arbitrary
// origination context names -- _any_ key in the TLS dictionary will get turned into
// an origination context.
//
// I seriously doubt that either of these will actually affect anyone at this remove,
// but... yeah.
type ModuleSecrets struct {
	Defaults struct {
		TLSSecretNamespacing bool `json:"tls_secret_namespacing"`
	} `json:"defaults"`
	Upstream struct {
		Secret string `json:"secret"`
	} `json:"upstream"`
	Server struct {
		Secret string `json:"secret"`
	} `json:"server"`
	Client struct {
		Secret string `json:"secret"`
	} `json:"client"`
}
