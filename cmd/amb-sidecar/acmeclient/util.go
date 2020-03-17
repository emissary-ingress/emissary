package acmeclient

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gogo/protobuf/proto"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func subjects(cert *x509.Certificate) []string {
	set := map[string]struct{}{}
	set[cert.Subject.CommonName] = struct{}{}
	for _, san := range cert.DNSNames {
		set[san] = struct{}{}
	}
	ret := make([]string, 0, len(set))
	for subject := range set {
		ret = append(ret, subject)
	}
	sort.Strings(ret)
	return ret
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NameEncode an arbitrary case-insensitive string to make it usable
// as a Kubernetes resource name.  The scheme is urlencode-ish, but
// '-' instead of '%', and encodes all non
// ASCII-letter/ASCII-digit/'.' octets; since as the characters
// allowed in a Kubernetes name are "digits (0-9), lower case letters
// (a-z), `-`, and `.`"[1].  Upper-case ASCII letters are simply
// translated to lower-case.
//
// For some Kubernetes resource types, `.` is also forbidden.  We
// don't deal with that case.
//
// [1]: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
func NameEncode(in string) string {
	out := new(strings.Builder)
	for _, b := range []byte(in) {
		switch {
		case 'A' <= b && b <= 'Z':
			out.WriteByte(b - 'A' + 'a')
		case ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || (b == '.'):
			out.WriteByte(b)
		default:
			fmt.Fprintf(out, "-%02x", b)
		}
	}
	return out.String()
}

// *grumble grumble* k8s.io/code-generator/cmd/deepcopy-gen *grumble*
func deepCopyHost(in *ambassadorTypesV2.Host) *ambassadorTypesV2.Host {
	bs, err := json.Marshal(in)
	if err != nil {
		// 'in' is a valid object.  This should never happen.
		panic(err)
	}

	var out ambassadorTypesV2.Host
	if err := json.Unmarshal(bs, &out); err != nil {
		// 'bs' is valid JSON, we just generated it.  This
		// should never happen.
		panic(err)
	}

	return &out
}

// *grumble grumble* k8s.io/code-generator/cmd/deepcopy-gen *grumble*
func deepCopyHostSpec(in *ambassadorTypesV2.HostSpec) *ambassadorTypesV2.HostSpec {
	bs, err := json.Marshal(in)
	if err != nil {
		// 'in' is a valid object.  This should never happen.
		panic(err)
	}

	var out ambassadorTypesV2.HostSpec
	if err := json.Unmarshal(bs, &out); err != nil {
		// 'bs' is valid JSON, we just generated it.  This
		// should never happen.
		panic(err)
	}

	return &out
}

// unstructureMetadata marshals a *k8sTypesMetaV1.ObjectMeta for use
// in a `*k8sTypesUnstructured.Unstructured`.
//
// `*k8sTypesUnstructured.Unstructured` requires that the "metadata"
// field be a `map[string]interface{}`.  Going through JSON is the
// easiest way to get from a typed `*k8sTypesMetaV1.ObjectMeta` to an
// untyped `map[string]interface{}`.  Yes, it's gross and stupid.
func unstructureMetadata(in *k8sTypesMetaV1.ObjectMeta) map[string]interface{} {
	var metadata map[string]interface{}
	bs, err := json.Marshal(in)
	if err != nil {
		// 'in' is a valid object.  This should never happen.
		panic(err)
	}

	if err := json.Unmarshal(bs, &metadata); err != nil {
		// 'bs' is valid JSON, we just generated it.  This
		// should never happen.
		panic(err)
	}

	return metadata
}

// unstructureHost returns a *k8sTypesUnstructured.Unstructured
// representation of an *ambassadorTypesV2.Host.  There are 2 reasons
// why we might want this:
//
//  1. For use with a k8sClientDynamic.Interface
//  2. For use as a k8sRuntime.Object
func unstructureHost(host *ambassadorTypesV2.Host) *k8sTypesUnstructured.Unstructured {
	return &k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "Host",
			"metadata":   unstructureMetadata(host.ObjectMeta),
			"spec":       host.Spec,
			"status":     host.Status,
		},
	}
}

// hostsEqual returns whether 2 Host resources are equal.  Use this
// instead of `proto.Equal()` because (gogo/protobuf v1.3.0)
// `proto.Equal()` panics on metav1.ObjectMeta.
func hostsEqual(a, b *ambassadorTypesV2.Host) bool {
	if a.GetNamespace() != b.GetNamespace() {
		return false
	}
	if a.GetName() != b.GetName() {
		return false
	}
	if !proto.Equal(a.Spec, b.Spec) {
		return false
	}
	if !proto.Equal(a.Status, b.Status) {
		return false
	}
	return true
}

// secretsEqual returns whether 2 Secret resources are equal.  Use
// this instead of `proto.Equal()` because (gogo/protobuf v1.3.0)
// `proto.Equal()` panics on metav1.ObjectMeta.
func secretsEqual(a, b *k8sTypesCoreV1.Secret) bool {
	// metadata
	if a.GetNamespace() != b.GetNamespace() {
		return false
	}
	if a.GetName() != b.GetName() {
		return false
	}
	if len(a.GetOwnerReferences()) != len(b.GetOwnerReferences()) {
		return false
	}
	for i := range a.GetOwnerReferences() {
		aRef := a.GetOwnerReferences()[i]
		bRef := b.GetOwnerReferences()[i]
		if aRef.APIVersion != bRef.APIVersion {
			return false
		}
		if aRef.Kind != bRef.Kind {
			return false
		}
		if aRef.Name != bRef.Name {
			return false
		}
		if aRef.UID != bRef.UID {
			return false
		}
		aController := aRef.Controller != nil && *aRef.Controller
		bController := bRef.Controller != nil && *bRef.Controller
		if aController != bController {
			return false
		}
		aBlockOwnerDeletion := aRef.BlockOwnerDeletion != nil && *aRef.BlockOwnerDeletion
		bBlockOwnerDeletion := bRef.BlockOwnerDeletion != nil && *bRef.BlockOwnerDeletion
		if aBlockOwnerDeletion != bBlockOwnerDeletion {
			return false
		}
	}
	// content
	if a.Type != b.Type {
		return false
	}
	if len(a.Data) != len(b.Data) {
		return false
	}
	for key, aVal := range a.Data {
		bVal, ok := b.Data[key]
		if !ok {
			return false
		}
		if !bytes.Equal(aVal, bVal) {
			return false
		}
	}
	return true
}

func secretIsOwnedBy(secret *k8sTypesCoreV1.Secret, owner *ambassadorTypesV2.Host) bool {
	for _, straw := range secret.ObjectMeta.OwnerReferences {
		if straw.APIVersion == owner.TypeMeta.APIVersion &&
			straw.Kind == owner.TypeMeta.Kind &&
			straw.Name == owner.GetName() &&
			straw.UID == owner.GetUID() {
			return true
		}
	}
	return false
}

func secretAddOwner(secret *k8sTypesCoreV1.Secret, owner *ambassadorTypesV2.Host) {
	secret.ObjectMeta.OwnerReferences = append(secret.ObjectMeta.OwnerReferences, k8sTypesMetaV1.OwnerReference{
		APIVersion: owner.TypeMeta.APIVersion,
		Kind:       owner.TypeMeta.Kind,
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	})
}

func (c *Controller) storeSecret(secretsGetter k8sClientCoreV1.SecretsGetter, secret *k8sTypesCoreV1.Secret) error {
	secretInterface := secretsGetter.Secrets(secret.GetNamespace())
	var err error
	var newSecret *k8sTypesCoreV1.Secret
	if secret.GetResourceVersion() == "" {
		newSecret, err = secretInterface.Create(secret)
	} else {
		newSecret, err = secretInterface.Update(secret)
	}
	if err != nil {
		return err
	}
	if newSecret.GetResourceVersion() != secret.GetResourceVersion() {
		c.knownChangedSecrets[ref{Name: secret.GetName(), Namespace: secret.GetNamespace()}] = struct{}{}
	}
	return nil
}

func strInArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}
