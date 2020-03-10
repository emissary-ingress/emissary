package mocks

import (
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
)

type MockLimiter struct {
	customHardLimits map[licensekeys.Limit]int
	isHardLimit      bool
}

func NewMockLimiter() *MockLimiter {
	return &MockLimiter{
		customHardLimits: make(map[licensekeys.Limit]int),
		isHardLimit:      false,
	}
}

func NewMockLimiterWithCustomHardLimit(isHardLimit bool) *MockLimiter {
	return &MockLimiter{
		customHardLimits: make(map[licensekeys.Limit]int),
		isHardLimit:      isHardLimit,
	}
}

func NewMockLimiterWithCounts(theMap map[licensekeys.Limit]int, isHardLimit bool) *MockLimiter {
	return &MockLimiter{
		customHardLimits: theMap,
		isHardLimit:      isHardLimit,
	}
}

func (ml *MockLimiter) CanUseFeature(f licensekeys.Feature) bool {
	return true
}

func (ml *MockLimiter) SetClaims(newClaims *licensekeys.LicenseClaimsLatest) {
}

func (ml *MockLimiter) GetClaims() *licensekeys.LicenseClaimsLatest {
	return nil
}

func (ml *MockLimiter) GetFeaturesOverLimitAtPointInTime() []string {
	return []string{}
}

func (ml *MockLimiter) GetLimitValueAtPointInTime(toCheck *licensekeys.Limit) int {
	if val, ok := ml.customHardLimits[*toCheck]; ok {
		return val
	} else {
		return licensekeys.GetLimitDefault(*toCheck)
	}
}

func (ml *MockLimiter) GetFeatureUsageValueAtPointInTime(toCheck *licensekeys.Limit) int {
	return 0
}

func (ml *MockLimiter) GetFeatureMaxUsageValue(toCheck *licensekeys.Limit) int {
	return 0
}

func (ml *MockLimiter) IsHardLimitAtPointInTime() bool {
	return ml.isHardLimit
}

func (ml *MockLimiter) CreateCountLimiter(limit *licensekeys.Limit) (limiter.CountLimiter, error) {
	return NewMockLimitCounter(ml.GetLimitValueAtPointInTime(limit)), nil
}

func (ml *MockLimiter) CreateRateLimiter(limit *licensekeys.Limit) (limiter.RateLimiter, error) {
	return NewMockLimitRate(ml.GetLimitValueAtPointInTime(limit)), nil
}
