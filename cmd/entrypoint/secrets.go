package entrypoint

import (
	"context"
	"log"
	"strings"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
)

func (s *AmbassadorInputs) ReconcileSecrets(ctx context.Context, client *kates.Client) {
	var resources []kates.Object

	for _, svc := range s.Services {
		ann, ok := svc.GetAnnotations()["getambassador.io/config"]
		if ok {
			objs, err := kates.ParseManifests(ann)
			if err != nil {
				log.Printf("error parsing annotations: %v", err)
			} else {
				for _, o := range objs {
					u, ok := o.(*kates.Unstructured)
					if ok {
						resources = append(resources, convertAnnotation(svc.GetNamespace(), u))
					} else {
						resources = append(resources, o)
					}
				}
			}
		}
	}

	for _, h := range s.Hosts {
		var id amb.AmbassadorID
		if len(h.Spec.AmbassadorID) > 0 {
			id = h.Spec.AmbassadorID
		} else {
			id = h.Spec.DeprecatedAmbassadorID
		}
		if include(id) {
			resources = append(resources, h)
		}
	}
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

	secretNamespacing := true
	for _, resource := range resources {
		mod, ok := resource.(*amb.Module)
		// XXX: ambassador_id!
		if ok && mod.GetName() == "ambassador" {
			secs := ModuleSecrets{}
			err := convert(mod.Spec.Config, &secs)
			if err != nil {
				log.Printf("error parsing module: %v", err)
				continue
			}
			secretNamespacing = secs.Defaults.TLSSecretNamespacing
			break
		}
	}

	refs := map[Ref]bool{}
	action := func(ref Ref) {
		refs[ref] = true
	}

	for _, resource := range resources {
		traverseSecretRefs(resource, secretNamespacing, action)
	}

	if IsEdgeStack() {
		secretRef(GetAmbassadorNamespace(), "fallback-self-signed-cert", false, action)
		secretRef(GetLicenseSecretNamespace(), GetLicenseSecretName(), false, action)
	}

	s.Secrets = make([]*kates.Secret, 0, len(refs))
	for _, secret := range s.AllSecrets {
		if refs[Ref{secret.GetNamespace(), secret.GetName()}] {
			s.Secrets = append(s.Secrets, secret)
		}
	}

	return
}

func include(id amb.AmbassadorID) bool {
	me := GetAmbassadorId()
	// XXX: is this right
	if len(id) == 0 && (me == "default" || me == "") {
		return true
	}

	for _, name := range id {
		if me == name {
			return true
		}
	}
	return false
}

func traverseSecretRefs(resource kates.Object, secretNamespacing bool, action func(Ref)) {
	switch r := resource.(type) {
	case *amb.Host:
		// Is this a kubernetes secret or something else, gonna assume it's something else right now:
		/*if host.TLS != nil {
			fmt.Println(host.TLS.CASecret)
		}*/
		if r.Spec.TLSSecret != nil && r.Spec.TLSSecret.Name != "" {
			secretRef(r.GetNamespace(), r.Spec.TLSSecret.Name, secretNamespacing, action)
		}
		if r.Spec.AcmeProvider != nil && r.Spec.AcmeProvider.PrivateKeySecret != nil &&
			r.Spec.AcmeProvider.PrivateKeySecret.Name != "" {
			secretRef(r.GetNamespace(), r.Spec.AcmeProvider.PrivateKeySecret.Name, secretNamespacing, action)
		}
	case *amb.TLSContext:
		if r.Spec.Secret != "" {
			if r.Spec.SecretNamespacing != nil {
				secretNamespacing = *r.Spec.SecretNamespacing
			}
			secretRef(r.GetNamespace(), r.Spec.Secret, secretNamespacing, action)
		}
	case *amb.Module:
		secs := ModuleSecrets{}
		err := convert(r.Spec.Config, &secs)
		if err != nil {
			// XXX
			log.Printf("error extracting secrets from module: %v", err)
			return
		}
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
		for _, itls := range r.Spec.TLS {
			if itls.SecretName != "" {
				secretRef(r.GetNamespace(), itls.SecretName, secretNamespacing, action)
			}
		}
	}

	return
}

func secretRef(namespace, name string, secretNamespacing bool, action func(Ref)) {
	if secretNamespacing {
		parts := strings.Split(name, ".")
		if len(parts) > 1 {
			namespace = parts[len(parts)-1]
			name = strings.Join(parts[:len(parts)-1], ".")
		}
	}

	action(Ref{namespace, name})
}

type Ref struct {
	Namespace string
	Name      string
}

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
