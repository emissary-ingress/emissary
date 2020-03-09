package licensekeys

import (
	"sort"

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
	FeatureUnrecognized        = internal.FeatureUnrecognized
	FeatureTraffic             = internal.AddFeature("traffic")
	FeatureRateLimit           = internal.AddFeature("ratelimit")
	FeatureFilter              = internal.AddFeature("filter")
	FeatureDevPortal           = internal.AddFeature("devportal")
	FeatureLocalDevPortal      = internal.AddFeature("local-devportal")
	FeatureCertifiedEnvoy      = internal.AddFeature("certified-envoy")
	FeatureSupportBusinessTier = internal.AddFeature("support-business-tier")
	FeatureSupport24x7Tier     = internal.AddFeature("support-24x7-tier")
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
	return errors.Errorf("Your license key does not grant the %q feature. Please contact sales@datawire.io for a free community license or discuss commercial options.", feature)
}

// ListKnownFeatures returns a list of known feature names (strings
// that are parsable by ParseFeature).  This is stringly-typed because
// it only exists so that "apictl-key create --help" can print a list
// of known features.
func ListKnownFeatures() []string {
	ret := internal.ListKnownFeatures()
	sort.Strings(ret)
	return ret
}
