package limiter

import (
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
)

// Limiter is what actually implements the limits defined inside of licensekeys
//
// It uses redis as the backing store.
type LimiterImpl struct {
	// The backing store for storing rates + used rates.
	redisPool *pool.Pool
	// licenseClaims defines the limits to enforce
	licenseClaims *licensekeys.LicenseClaimsLatest
	// cryptoEngine is used for encrypting the values store in redis.
	cryptoEngine *LimitCrypto
	// unregisteredLicense is used to track unregistered licenses and enforce hard limits
	unregisteredLicenseHardLimits bool
	// phoneHomeHardLimits is used to track Metriton response enforcing hard limits
	phoneHomeHardLimits bool
	// limiters is used to track instantiated UsageLimiters
	limiters map[string]UsageLimiter
}

// NewLimiter properly creates a limiter instance.
//
// redisPool: a potential redis connection
// claims: the potential claims the user has activated.
func NewLimiterImpl() *LimiterImpl {
	return &LimiterImpl{
		redisPool:                     nil,
		licenseClaims:                 nil,
		unregisteredLicenseHardLimits: false,
		phoneHomeHardLimits:           false,
		limiters:                      map[string]UsageLimiter{},
	}
}

// hasRedisConnection is a small wrapper for checking a redis connection.
//
// TODO(cynthia): do some sort of uptime/downtime detection here
//                in case redis is temporarily unavailable.
func (l *LimiterImpl) hasRedisConnection() bool {
	return l.redisPool != nil
}

func (l *LimiterImpl) registerLimiter(limit *licensekeys.Limit, limiter UsageLimiter) {
	l.limiters[limit.String()] = limiter
}

func (l *LimiterImpl) lookupLimiter(limit *licensekeys.Limit) (limiter UsageLimiter) {
	return l.limiters[limit.String()]
}

// CanUseFeature determines if a feature can be utilized.
//
// f: the feature to check if available.
func (l *LimiterImpl) CanUseFeature(f licensekeys.Feature) bool {
	return l.licenseClaims != nil && l.licenseClaims.RequireFeature(f) == nil
}

// SetRedisPool is useful for associating a redis connection pool after initialization.
func (l *LimiterImpl) SetRedisPool(newRedisPool *pool.Pool) {
	l.redisPool = newRedisPool
}

// SetClaims is useful for reloading a license key while the program is running.
func (l *LimiterImpl) SetClaims(newClaims *licensekeys.LicenseClaimsLatest) {
	l.licenseClaims = newClaims
	l.cryptoEngine = NewLimitCrypto(newClaims)
}

// GetClaims is useful for using the loaded claims when computing limits, even after they were reloaded
func (l *LimiterImpl) GetClaims() *licensekeys.LicenseClaimsLatest {
	return l.licenseClaims
}

// SetUnregisteredLicenseHardLimits is useful for toggling license-enforced hard limits
func (l *LimiterImpl) SetUnregisteredLicenseHardLimits(newUnregisteredLicenseHardLimits bool) {
	l.unregisteredLicenseHardLimits = newUnregisteredLicenseHardLimits
}

// SetUnregisteredLicenseHardLimits is useful for toggling metriton-enforced hard limits
func (l *LimiterImpl) SetPhoneHomeHardLimits(newPhoneHomeHardLimits bool) {
	l.phoneHomeHardLimits = newPhoneHomeHardLimits
}

// GetFeaturesOverLimitAtPointInTime returns the current features with usage above licensed limits.
// This is called point in time to communicate that this can change
// over time, and you should check it back often.
func (l *LimiterImpl) GetFeaturesOverLimitAtPointInTime() []string {
	licensedFeaturesOverLimit := []string{}
	for _, limitName := range licensekeys.ListKnownLimits() {
		limit, ok := licensekeys.ParseLimit(limitName)
		if ok {
			limitValue := l.GetLimitValueAtPointInTime(&limit)
			usageValue := l.GetFeatureUsageValueAtPointInTime(&limit)
			if usageValue >= limitValue {
				licensedFeaturesOverLimit = append(licensedFeaturesOverLimit, limitName)
			}
		}
	}
	return licensedFeaturesOverLimit
}

// GetLimitValueAtPointInTime returns the current limit value for the current
// license key. This is called point in time to communicate that this can change
// over time, and you should check it back often.
func (l *LimiterImpl) GetLimitValueAtPointInTime(toCheck *licensekeys.Limit) int {
	return l.licenseClaims.GetLimitValue(*toCheck)
}

// GetFeatureUsageValueAtPointInTime returns the feature's usage value.
// This is called point in time to communicate that this can change
// over time, and you should check it back often.
func (l *LimiterImpl) GetFeatureUsageValueAtPointInTime(toCheck *licensekeys.Limit) int {
	limiter := l.lookupLimiter(toCheck)
	if limiter != nil {
		value, _ := limiter.GetUsageAtPointInTime()
		return value
	}
	return 0
}

// GetFeatureMaxUsageValue returns the feature's maximum usage value.
func (l *LimiterImpl) GetFeatureMaxUsageValue(toCheck *licensekeys.Limit) int {
	limiter := l.lookupLimiter(toCheck)
	if limiter != nil {
		value, _ := limiter.GetMaxUsage()
		return value
	}
	return 0
}

// IsHardLimitAtPointInTime determines if at the point of time of calling this
// we should enforce hard limits.
func (l *LimiterImpl) IsHardLimitAtPointInTime() bool {
	return l.unregisteredLicenseHardLimits || l.phoneHomeHardLimits
}

// CreateCountLimiter creates a limiter that is capable of enforcing counts.
func (l *LimiterImpl) CreateCountLimiter(limit *licensekeys.Limit) (CountLimiter, error) {
	if limit.Type() != licensekeys.LimitTypeCount {
		return nil, errors.New("This limit is not a count type")
	}

	realInstance, err := newCountLimiterImpl(l.redisPool, l, limit, l.cryptoEngine)
	if err != nil {
		return newNoNoCounter(), nil
	} else {
		l.registerLimiter(limit, realInstance)
		return realInstance, nil
	}
}

// CreateRateLimiter creates a limiter that is capable of enforcing rates.
func (l *LimiterImpl) CreateRateLimiter(limit *licensekeys.Limit) (RateLimiter, error) {
	if limit.Type() != licensekeys.LimitTypeRate {
		return nil, errors.New("This limit is not a rate type")
	}

	realInstance, err := newRateLimiterWindow(l.redisPool, l, limit)
	if err != nil {
		return newNoNoRate(), nil
	} else {
		l.registerLimiter(limit, realInstance)
		return realInstance, nil
	}
}
