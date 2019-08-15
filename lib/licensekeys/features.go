package licensekeys

import (
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys/internal"
)

// The actual implementation of the "Feature" type is moved to a
// separate "internal/features.go", so that it is is hard to have
// inconsistent feature strings.  It's a hack to get the Go compiler
// to do more checking for us.

// Feature is a hacky approximation of an "enum" (since Go doesn't
// have enums).  It implements fmt.Stringer, encoding/json.Marshaler,
// and encoding/json.Unmarshaler.
type Feature = internal.Feature

// This is the exhaustive list of values that a Feature may take.
var (
	FeatureUnrecognized = internal.FeatureUnrecognized
	FeatureTraffic      = internal.AddFeature("traffic")
	FeatureRateLimit    = internal.AddFeature("ratelimit")
	FeatureFilter       = internal.AddFeature("filter")
	FeatureDevPortal    = internal.AddFeature("devportal")
)

// ParseFeature turns a feature string in to one of the recognized
// Feature enum values.  If is a recognized feature string, it returns
// (FeatureThatFeature, true); or else it returns
// (FeatureUnrecognized, false).
func ParseFeature(str string) (feature Feature, ok bool) {
	return internal.ParseFeature(str)
}

// RequireFeature returns an error if this license key does not grant
// access to the requested feature.
func (cl *LicenseClaimsLatest) RequireFeature(feature Feature) error {
	for _, straw := range cl.EnabledFeatures {
		if straw == feature {
			return nil
		}
	}
	return errors.Errorf("license key does not grant the %q feature", feature)
}

// ListKnownFeatures returns a list of known feature names (strings
// that are parsable by ParseFeature).  This is stringly-typed because
// it only exists so that "apictl-key create --help" can print a list
// of known features.
func ListKnownFeatures() []string {
	return internal.ListKnownFeatures()
}
