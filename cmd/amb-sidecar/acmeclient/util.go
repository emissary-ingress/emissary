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

// NameEncode an arbitrary string (such as a qualified hostname) to
// make it usable as a Kubernetes resource name.  The scheme is
// urlencode-ish, but '-' instead of '%', and encodes all non
// ASCII-letter/ASCII-digit octets; since letter/digit/hyphen (LDH) is
// the set of characters that are always safe to use in the various
// kubernetes contexts.
func NameEncode(in string) string {
	out := new(strings.Builder)
	for _, b := range []byte(in) {
		switch {
		case 'A' <= b && b <= 'Z':
			out.WriteByte(b - 'A' + 'a')
		case ('a' <= b && b <= 'z') || ('0' <= b && b <= '9'):
			out.WriteByte(b)
		default:
			fmt.Fprintf(out, "-%02x", b)
		}
	}
	return out.String()
}

// *grumble grumble* k8s.io/code-generator/cmd/deepcopy-gen *grumble*
func deepCopyHost(in *ambassadorTypesV2.Host) *ambassadorTypesV2.Host {
	b, _ := json.Marshal(in)
	var out ambassadorTypesV2.Host
	_ = json.Unmarshal(b, &out)
	return &out
}

// ufnstructureMetadata marshals a *k8sTypesMetaV1.ObjectMeta for us
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
		// 'bs' is a valid JSON, we just generated it.  This
		// should never happen.
		panic(err)
	}
	return metadata
}

// hostsEqual returns whether 2 Host resources are equal.  Use this
// instead of `proto.Equal()` because Host is not (yet?) spec'ed as a
// protobuf (but HostSpec is, so we use `proto.Equal()` internally).
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
	return true
}

// secretsEqual returns whether 2 Secret resources are equal.  Use
// this instead of `proto.Equal()` because there seems to be a bug in
// (gogo/protobuf v1.3.0) `proto.Equal()` that causes it to panic when
// I give it secrets.  IDK.
func secretsEqual(a, b *k8sTypesCoreV1.Secret) bool {
	if a.GetNamespace() != b.GetNamespace() {
		return false
	}
	if a.GetName() != b.GetName() {
		return false
	}
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
