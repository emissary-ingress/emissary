package limiter

import (
	"github.com/datawire/apro/lib/licensekeys"
)

// Limiter defines a common implementation for limiter so we can mock it in tests.
type Limiter interface {
	// Determine if we can use a particular feature.
	CanUseFeature(f licensekeys.Feature) bool
	// Set the license key claims (to support reloading).
	SetClaims(newClaims *licensekeys.LicenseClaimsLatest)
	// Get the license key claims.
	GetClaims() *licensekeys.LicenseClaimsLatest
	// Get a particular limit value at a point in time.
	GetFeaturesOverLimitAtPointInTime() []string
	// Get a particular limit value at a point in time.
	GetLimitValueAtPointInTime(toCheck *licensekeys.Limit) int
	// Get a particular feature usage at a point in time.
	GetFeatureUsageValueAtPointInTime(toCheck *licensekeys.Limit) int
	// Get a particular feature maximum recorded usage.
	GetFeatureMaxUsageValue(toCheck *licensekeys.Limit) int
	// Are we enforcing hard limits right now?
	IsHardLimitAtPointInTime() bool
	// Create a limiter that can handle counts.
	CreateCountLimiter(limit *licensekeys.Limit) (CountLimiter, error)
	// Create a limiter that can handle rates.
	CreateRateLimiter(limit *licensekeys.Limit) (RateLimiter, error)
}
