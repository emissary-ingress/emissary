package acmeclient

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
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

// nameEncode an arbitrary string (such as a qualified hostname) to
// make it usable as a Kubernetes resource name.  The scheme is
// urlencode-ish, but '-' instead of '%', and encodes all non
// ASCII-letter/ASCII-digit octets; since letter/digit/hyphen (LDH) is
// the set of characters that are always safe to use in the various
// kubernetes contexts.
func nameEncode(in string) string {
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
func deepCopyHostSpec(in *ambassadorTypesV2.HostSpec) *ambassadorTypesV2.HostSpec {
	b, _ := json.Marshal(in)
	var out ambassadorTypesV2.HostSpec
	_ = json.Unmarshal(b, &out)
	return &out
}
