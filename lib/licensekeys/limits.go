package licensekeys

import (
	"sort"

	"github.com/datawire/apro/lib/licensekeys/internal"
)

// The actual implementation of the "Limit" type is moved to a
// separate "internal/limits.go", so that it is is hard to have
// inconsistent limit strings.  It's a hack to get the Go compiler
// to do more checking for us.

// Limit is a hacky approximation of an "enum" (since Go doesn't
// have enums).  It implements fmt.Stringer, encoding/json.Marshaler,
// and encoding/json.Unmarshaler.
type Limit = internal.Limit

// This is the exhaustive list of values that a Limit may take.
var (
	LimitUnrecognized      = internal.LimitUnrecognized
	LimitDevPortalServices = addLimit("devportal-services", 5)
)

var limitDefaults = make(map[string]int)

func addLimit(name string, defautlValue int) Limit {
	limit := internal.AddLimit(name)
	limitDefaults[limit.String()] = defautlValue
	return limit
}

// ParseLimit turns a limit string in to one of the recognized
// Limit enum values.  If is a recognized limit string, it returns
// (LimitThatLimit, true); or else it returns
// (LimitUnrecognized, false).
func ParseLimit(str string) (limit Limit, ok bool) {
	return internal.ParseLimit(str)
}

// GetLimitValue returns the limit defaultValue if this license key does not specify
// the requested limit.
func (cl *LicenseClaimsLatest) GetLimitValue(limit Limit) int {
	for _, straw := range cl.EnforcedLimits {
		if straw.Name == limit {
			return straw.Value
		}
	}
	return GetLimitDefault(limit)
}

// GetLimitDefault returns the default value for the limit
func GetLimitDefault(limit Limit) int {
	return limitDefaults[limit.String()]
}

// ListKnownLimits returns a list of known limit names (strings
// that are parsable by ParseLimit).  This is stringly-typed because
// it only exists so that "apictl-key create --help" can print a list
// of known limits.
func ListKnownLimits() []string {
	ret := internal.ListKnownLimits()
	sort.Strings(ret)
	return ret
}
