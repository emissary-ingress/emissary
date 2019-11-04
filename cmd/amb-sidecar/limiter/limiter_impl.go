package limiter

import (
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
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
}

// NewLimiter properly creates a limiter instance.
//
// redisPool: a potential redis connection
// claims: the potential claims the user has activated.
func NewLimiterImpl(redisPool *pool.Pool, claims *licensekeys.LicenseClaimsLatest) *LimiterImpl {
	return &LimiterImpl{
		redisPool:     redisPool,
		licenseClaims: claims,
	}
}

// hasRedisConnection is a small wrapper for checking a redis connection.
//
// TODO(cynthia): do some sort of uptime/downtime detection here
//                in case redis is temporarily unavailable.
func (l *LimiterImpl) hasRedisConnection() bool {
	return l.redisPool != nil
}

// CanUseFeature determines if a feature can be utilized.
//
// f: the feature to check if available.
func (l *LimiterImpl) CanUseFeature(f licensekeys.Feature) bool {
	return l.licenseClaims != nil && l.licenseClaims.RequireFeature(f) == nil
}

// SetClaims is useful for reloading a license key while the program is running.
func (l *LimiterImpl) SetClaims(newClaims *licensekeys.LicenseClaimsLatest) {
	l.licenseClaims = newClaims
}

// GetLimitValueAtPointInTime returns the current limit value for the current
// license key. This is called point in time to communicate that this can change
// over time, and you should check it back often.
func (l *LimiterImpl) GetLimitValueAtPointInTime(toCheck licensekeys.Limit) int {
	return l.licenseClaims.GetLimitValue(toCheck)
}

// IsHardLimitAtPointInTime determines if at the point of time of calling this
// we should enforce hard limits.
func (l *LimiterImpl) IsHardLimitAtPointInTime() bool {
	// TODO: integrate with phone home.
	return false
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
		return realInstance, nil
	}
}
