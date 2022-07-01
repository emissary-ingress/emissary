package snapshot

import (
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Currently, this only removes "sensitive" information, which, for now, is just Secrets.data and
// anything that's not object metadata from Invalid objects. (since we couldn't parse the things in
// "invalid", we actually don't know what they are so they could contain secrets.)
//
// TODO:(@acookin) Could also remove server generated bits from here, e.g. the last applied configuration
// annotation that the kube server applies. The benefit of that would be to reduce bits sent across
// the wire.
func (s *Snapshot) Sanitize() error {
	var err error
	if s.Kubernetes != nil {
		if err = s.Kubernetes.Sanitize(); err != nil {
			return err
		}
	}
	// this invalid stuff could contain secret secret things probably
	// so it's probably best to scrub the contents and just send along the object meta and
	// error?
	if len(s.Invalid) > 0 {
		scrubbedInvalid := []*kates.Unstructured{}
		for _, invalid := range s.Invalid {
			scrubbed := kates.NewUnstructured(invalid.GetKind(), invalid.GetAPIVersion())
			scrubbed.SetName(invalid.GetName())
			scrubbed.SetNamespace(invalid.GetNamespace())
			invalidErrs, hasErrors := invalid.Object["errors"]
			if hasErrors {
				rawContent := scrubbed.UnstructuredContent()
				rawContent["errors"] = invalidErrs
				scrubbed.SetUnstructuredContent(rawContent)
			}
			scrubbedInvalid = append(scrubbedInvalid, scrubbed)
		}
		s.Invalid = scrubbedInvalid
	}
	return nil
}

func (ambInputs *KubernetesSnapshot) Sanitize() error {
	// create new secrets so we only carry over info we want
	// secret values can live on in the last applied configuration annotation, for example

	// another option here is that we could have a `Sanatizable` interface, and have each object
	// that needs to be cleaned up a bit implement `Sanitize()`, but, imo, that's harder to
	// read
	sanitizedSecrets := []*kates.Secret{}
	for _, secret := range ambInputs.Secrets {
		sanitizedSecret := &kates.Secret{
			Type:     secret.Type,
			TypeMeta: secret.TypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.ObjectMeta.Name,
				Namespace: secret.ObjectMeta.Namespace,
			},
			Data: map[string][]byte{},
		}
		for k := range secret.Data {
			// adding the keys but removing the data, which is probably fine
			// note: the data will be base64 encoded during json serialization which is
			// fun. I still think <REDACTED> is better than an empty string because an
			// empty string could be a real value. Also just making this a garbage
			// random value would make things harder to debug, so I'd prefer not to
			sanitizedSecret.Data[k] = []byte(`<REDACTED>`)
		}
		sanitizedSecrets = append(sanitizedSecrets, sanitizedSecret)
	}
	ambInputs.Secrets = sanitizedSecrets

	return nil
}
