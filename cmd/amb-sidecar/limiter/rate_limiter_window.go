package limiter

import (
	"fmt"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
)

// RateLimiter limits on sliding window.
type RateLimiterWindow struct {
	// A connection to the redis instance.
	//
	// This is required for a rate limiter.
	redisPool *pool.Pool
	// Need to take an instance of the limiter since the license key can be
	// reloaded.
	limiter Limiter
	// limit is the actual limit to enforce
	limit *licensekeys.Limit
	// cryptoEngine is used for encrypting the values store in redis.
	cryptoEngine *LimitCrypto
}

var count int

// newRateLimiterWindow creates a new limit based on a rate of requests.
//
// redisPool: a connection to the redisPool
// limit: the actual limit value.
// crypto: the engine used for actually encrypting values in redis.
func newRateLimiterWindow(redisPool *pool.Pool, limiter Limiter, limit *licensekeys.Limit, cryptoEngine *LimitCrypto) (*RateLimiterWindow, error) {
	if redisPool == nil {
		return nil, errors.New("Need a redis pool to enforce a rate limit")
	}

	return &RateLimiterWindow{
		redisPool,
		limiter,
		limit,
		cryptoEngine,
	}, nil
}

func (this *RateLimiterWindow) getUnderlyingValue() (int, error) {
	// TODO(alexgervais): impl
	return count, nil
}

func (this *RateLimiterWindow) attemptToChange(incrementing bool) (int, error) {
	// TODO(alexgervais): impl
	if incrementing {
		count++
	}
	// Are we going to exceed limits, and is that a problem?
	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)
	if currentLimit != -1 && count > currentLimit && this.limiter.IsHardLimitAtPointInTime() {
		// If we're decrementing than just ignore the limit value increase.
		// If you wanna get closer to your hard limit that's more than fine.
		if incrementing {
			return -1, fmt.Errorf("rate-limit exceeded for feature %s", this.limit)
		}
	}
	return count, nil
}

// IncrementUsage tracks an increment in the usage rate.
//
// Returns an error if we couldn't due to limits, or redis failure.
// If the error is present do not allow the increment.
func (this *RateLimiterWindow) IncrementUsage() error {
	_, err := this.attemptToChange(true)
	return err
}

// IsExceedingAtPointInTime determines if we're exceeding at this point in
// time.
func (this *RateLimiterWindow) IsExceedingAtPointInTime() (bool, error) {
	currentValue, err := this.getUnderlyingValue()
	if err != nil {
		return false, err
	}
	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)
	return currentValue > currentLimit, nil
}

func (this *RateLimiterWindow) GetUsageAtPointInTime() (int, error) {
	return this.getUnderlyingValue()
}
