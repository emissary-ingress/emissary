package entrypoint

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dlog"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/kates/k8s_resource_types"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// checkSecret checks whether a secret is valid, and adds it to the list of secrets
// in this snapshot if so.
func checkSecret(
	ctx context.Context,
	sh *SnapshotHolder,
	what string,
	ref snapshotTypes.SecretRef,
	secret *v1.Secret) {
	forceSecretValidation, _ := strconv.ParseBool(os.Getenv("AMBASSADOR_FORCE_SECRET_VALIDATION"))
	// Make it more convenient to consistently refer to this secret.
	secretName := fmt.Sprintf("%s secret %s.%s", what, ref.Name, ref.Namespace)

	if secret == nil {
		// This is "impossible". Arguably it should be a panic...
		dlog.Debugf(ctx, "%s not found", secretName)
		return
	}

	// Assume that the secret is valid...
	isValid := true

	// ...and that we have no errors.
	var errs derror.MultiError

	// OK, do we have a TLS private key?
	privKeyPEMBytes, ok := secret.Data[v1.TLSPrivateKeyKey]

	if ok && len(privKeyPEMBytes) > 0 {
		// Yes. We need to be able to decode it.
		caKeyBlock, _ := pem.Decode(privKeyPEMBytes)

		if caKeyBlock != nil {
			dlog.Debugf(ctx, "%s has private key, block type %s", secretName, caKeyBlock.Type)

			// First try PKCS1.
			_, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)

			if err != nil {
				// Try PKCS8? (No, = instead of := is not a typo here: we're overwriting the
				// earlier error.)
				_, err = x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
			}

			if err != nil {
				// Try EC? (No, = instead of := is not a typo here: we're overwriting the
				// earlier error.)
				_, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
			}

			// Any issues here?
			if err != nil {
				errs = append(errs,
					fmt.Errorf("%s %s cannot be parsed as PKCS1, PKCS8, or EC: %s", secretName, v1.TLSPrivateKeyKey, err.Error()))
				isValid = false
			}
		} else {
			errs = append(errs,
				fmt.Errorf("%s %s is not a PEM-encoded key", secretName, v1.TLSPrivateKeyKey))
			isValid = false
		}
	}

	// How about a TLS cert bundle?
	caCertPEMBytes, ok := secret.Data[v1.TLSCertKey]

	if ok && len(caCertPEMBytes) > 0 {
		caCertBlock, _ := pem.Decode(caCertPEMBytes)

		if caCertBlock != nil {
			dlog.Debugf(ctx, "%s has public key, block type %s", secretName, caCertBlock.Type)

			_, err := x509.ParseCertificate(caCertBlock.Bytes)

			if err != nil {
				errs = append(errs,
					fmt.Errorf("%s %s cannot be parsed as x.509: %s", secretName, v1.TLSCertKey, err.Error()))
				isValid = false
			}
		} else {
			errs = append(errs,
				fmt.Errorf("%s %s is not a PEM-encoded certificate", secretName, v1.TLSCertKey))
			isValid = false
		}
	}

	if isValid || !forceSecretValidation {
		dlog.Debugf(ctx, "taking %s", secretName)
		sh.k8sSnapshot.Secrets = append(sh.k8sSnapshot.Secrets, secret)
	}
	if !isValid {
		// This secret is invalid, but we're not going to log about it -- instead, it'll go into the
		// list of Invalid resources.
		dlog.Debugf(ctx, "%s is not valid, skipping: %s", secretName, errs.Error())

		// We need to add this to our set of invalid resources. Sadly, this means we need to convert it
		// to an Unstructured and redact various bits.
		secretBytes, err := json.Marshal(secret)

		if err != nil {
			// This we'll log about, since it's impossible.
			dlog.Errorf(ctx, "unable to marshal invalid %s: %s", secretName, err)
			return
		}

		var unstructuredSecret kates.Unstructured
		err = json.Unmarshal(secretBytes, &unstructuredSecret)

		if err != nil {
			// This we'll log about, since it's impossible.
			dlog.Errorf(ctx, "unable to unmarshal invalid %s: %s", secretName, err)
			return
		}

		// Construct a redacted version of things in the original data map.
		redactedData := map[string]interface{}{}

		for key := range secret.Data {
			redactedData[key] = "-redacted-"
		}

		unstructuredSecret.Object["data"] = redactedData

		// We have to toss the last-applied-configuration as well... and we may as well toss the
		// managedFields.

		metadata, ok := unstructuredSecret.Object["metadata"].(map[string]interface{})

		if ok {
			delete(metadata, "managedFields")

			annotations, ok := metadata["annotations"].(map[string]interface{})

			if ok {
				delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

				if len(annotations) == 0 {
					delete(metadata, "annotations")
				}
			}

			if len(metadata) == 0 {
				delete(unstructuredSecret.Object, "metadata")
			}
		}

		// Finally, mark it invalid.
		sh.validator.addInvalid(ctx, &unstructuredSecret, errs.Error())
	}
}

// ReconcileSecrets figures out which secrets we're actually using,
// since we don't want to send secrets to Ambassador unless we're
// using them, since any secret we send will be saved to disk.
func ReconcileSecrets(ctx context.Context, sh *SnapshotHolder) error {
	envAmbID := GetAmbassadorID()

	// Start by building up a list of all the K8s objects that are
	// allowed to mention secrets. Note that we vet the ambassador_id
	// for all of these before putting them on the list.
	var resources []kates.Object

	// Annotations are straightforward, although honestly we should
	// be filtering annotations by type here (or, even better, unfold
	// them earlier so that we can treat them like any other resource
	// here).

	for _, list := range sh.k8sSnapshot.Annotations {
		for _, a := range list {
			if _, isInvalid := a.(*kates.Unstructured); isInvalid {
				continue
			}
			if GetAmbID(ctx, a).Matches(envAmbID) {
				resources = append(resources, a)
			}
		}
	}

	// Hosts are a little weird, because we have two ways to find the
	// ambassador_id. Sorry about that.
	for _, h := range sh.k8sSnapshot.Hosts {
		var id amb.AmbassadorID
		if len(h.Spec.AmbassadorID) > 0 {
			id = h.Spec.AmbassadorID
		}
		if id.Matches(envAmbID) {
			resources = append(resources, h)
		}
	}

	// TLSContexts, Modules, and Ingresses are all straightforward.
	for _, t := range sh.k8sSnapshot.TLSContexts {
		if t.Spec.AmbassadorID.Matches(envAmbID) {
			resources = append(resources, t)
		}
	}
	for _, m := range sh.k8sSnapshot.Modules {
		if m.Spec.AmbassadorID.Matches(envAmbID) {
			resources = append(resources, m)
		}
	}
	for _, i := range sh.k8sSnapshot.Ingresses {
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

	// We _always_ have an implicit references to the cloud-connec-token secret...
	secretRef(GetCloudConnectTokenResourceNamespace(), GetCloudConnectTokenResourceName(), false, action)

	// We _always_ have an implicit references to the fallback cert secret...
	secretRef(GetAmbassadorNamespace(), "fallback-self-signed-cert", false, action)

	isEdgeStack, err := IsEdgeStack()
	if err != nil {
		return err
	}
	if isEdgeStack {
		// ...and for Edge Stack, we _always_ have an implicit reference to the
		// license secret.
		secretRef(GetLicenseSecretNamespace(), GetLicenseSecretName(), false, action)
		// We also want to grab any secrets referenced by Edge-Stack filters for use in Edge-Stack
		// the Filters are unstructured because Emissary does not have their type definition
		for _, f := range sh.k8sSnapshot.Filters {
			err := findFilterSecret(f, action)
			if err != nil {
				dlog.Errorf(ctx, "Error gathering secret reference from Filter: %v", err)
			}
		}
	}

	// OK! After all that, go copy all the matching secrets from FSSecrets and
	// K8sSecrets to Secrets.
	//
	// The way this works is kind of simple: first we check everything in
	// FSSecrets. Then, when we check K8sSecrets, we skip any secrets that are
	// also in FSSecrets. End result: FSSecrets wins if there are any conflicts.
	sh.k8sSnapshot.Secrets = make([]*kates.Secret, 0, len(refs))

	for ref, secret := range sh.k8sSnapshot.FSSecrets {
		if refs[ref] {
			checkSecret(ctx, sh, "FSSecret", ref, secret)
		}
	}

	for _, secret := range sh.k8sSnapshot.K8sSecrets {
		ref := snapshotTypes.SecretRef{Namespace: secret.GetNamespace(), Name: secret.GetName()}

		_, found := sh.k8sSnapshot.FSSecrets[ref]
		if found {
			dlog.Debugf(ctx, "Conflict! skipping K8sSecret %#v", ref)
			continue
		}

		if refs[ref] {
			checkSecret(ctx, sh, "K8sSecret", ref, secret)
		}
	}
	return nil
}

// Returns secretName, secretNamespace from a provided (unstructured) filter if it contains a secret
// Returns empty strings when the secret name and/or namespace could not be found
func findFilterSecret(filter *unstructured.Unstructured, action func(snapshotTypes.SecretRef)) error {
	// Just making extra sure this is actually a Filter
	if filter.GetKind() != "Filter" {
		return fmt.Errorf("non-Filter object in Snapshot.Filters: %s", filter.GetKind())
	}
	// Only OAuth2 Filters have secrets, although they don't need to have them.
	// This is overly contrived because Filters are unstructured to Emissary since we don't have the type definitions
	// Yes this is disgusting. It is what it is...
	filterContents := filter.UnstructuredContent()
	filterSpec := filterContents["spec"]
	if filterSpec != nil {
		mapOAuth, ok := filterSpec.(map[string]interface{})
		// We need to check if all these type assertions fail since we shouldnt rely on CRD validation to protect us from a panic state
		// I cant imagine a scenario where this would realisticly happen, but we generate a unique log message for tracability and skip processing it
		if !ok {
			// We bail early any time we detect bogus contents for any of these fields
			// and let the APIServer, apiext, and amb-sidecar handle the error reporting
			return nil
		}
		oAuthFilter := mapOAuth["OAuth2"]
		if oAuthFilter != nil {
			secretName, secretNamespace := "", ""
			// Check if we have a secretName
			mapOAuth, ok := oAuthFilter.(map[string]interface{})
			if !ok {
				return nil
			}
			sName := mapOAuth["secretName"]
			if sName == nil {
				return nil
			}
			secretName, ok = sName.(string)
			// This is a weird check, but we have to handle the case where secretName is not provided, and when its explicitly set to ""
			if !ok || secretName == "" {
				// Bail out early since there is no secret
				return nil
			}
			sNamespace := mapOAuth["secretNamespace"]
			if sNamespace == nil {
				secretNamespace = filter.GetNamespace()
			} else {
				secretNamespace, ok = sNamespace.(string)
				if !ok {
					return nil
				} else if secretNamespace == "" {
					secretNamespace = filter.GetNamespace()
				}
			}
			secretRef(secretNamespace, secretName, false, action)
		}
	}
	return nil
}

// Find all the secrets a given Ambassador resource references.
func findSecretRefs(ctx context.Context, resource kates.Object, secretNamespacing bool, action func(snapshotTypes.SecretRef)) {
	switch r := resource.(type) {
	case *amb.Host:
		// The Host resource is a little odd. Host.spec.tls, Host.spec.tlsSecret, and
		// host.spec.acmeProvider.privateKeySecret can all refer to secrets.
		if r.Spec == nil {
			return
		}

		if r.Spec.TLS != nil {
			// Host.spec.tls.caSecret is the thing to worry about here.
			secretRef(r.GetNamespace(), r.Spec.TLS.CASecret, secretNamespacing, action)

			if r.Spec.TLS.CRLSecret != "" {
				secretRef(r.GetNamespace(), r.Spec.TLS.CRLSecret, secretNamespacing, action)
			}
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

		if r.Spec.CRLSecret != "" {
			if r.Spec.SecretNamespacing != nil {
				secretNamespacing = *r.Spec.SecretNamespacing
			}
			secretRef(r.GetNamespace(), r.Spec.CRLSecret, secretNamespacing, action)
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

	case *k8s_resource_types.Ingress:
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
