package limiter

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
)

var ErrRateLimiterNoRedis = errors.New("need a redis pool")

// RateLimiter limits on sliding window.
type RateLimiterWindow struct {
	// A connection to the redis instance.
	// This is required for a rate limiter.
	redisPool *pool.Pool
	// Need to take an instance of the limiter since the license key can be
	// reloaded.
	limiter Limiter
	// limit is the actual limit to enforce
	limit *licensekeys.Limit
	// sliding time window for calculating rate limits
	windowInSeconds int64
}

// newRateLimiterWindow creates a new limit based on a rate of requests.
//
// redisPool: a connection to the redisPool
// limit: the actual limit value.
// crypto: the engine used for actually encrypting values in redis.
func newRateLimiterWindow(redisPool *pool.Pool, limiter Limiter, limit *licensekeys.Limit) (*RateLimiterWindow, error) {
	return &RateLimiterWindow{
		redisPool,
		limiter,
		limit,
		int64(1),
	}, nil
}

func (this *RateLimiterWindow) getUnderlyingValue() (int, error) {
	// Get the number of events in the sorted-set
	resp, err := this.redisPool.Cmd("ZCARD", this.limit.String()).Int()
	if err != nil {
		if err == redis.ErrRespNil {
			return 0, err
		}
		return -1, err
	}
	this.recordMaxValue(resp)
	return resp, nil
}

// recordMaxValue records the maximum observed value in the last 24 hours.
func (this *RateLimiterWindow) recordMaxValue(value int) {
	previousValue, _ := this.getMaxValue()
	// if the new value is greater than the previously recorded maxValue,
	// set a "limit-m" redis key with the new maxValue and have it expire in 24h.
	if value > previousValue {
		this.redisPool.Cmd("SET", this.limit.String()+"-m", value, "EX", "86400")
	}
}

// getMaxValue returns the maximum observed value.
func (this *RateLimiterWindow) getMaxValue() (int, error) {
	resp, err := this.redisPool.Cmd("GET", this.limit.String()+"-m").Str()
	if err != nil {
		if err == redis.ErrRespNil || resp == "" {
			return 0, err
		}
		return -1, err
	}
	return strconv.Atoi(resp)
}

func (this *RateLimiterWindow) attemptToChange(incrementing bool) (int, error) {
	rc, err := this.redisPool.Get()
	if err != nil {
		return -1, err
	}
	defer this.redisPool.Put(rc)

	currentTimeMs := time.Now().UnixNano() / int64(time.Millisecond)
	maxScoreMs := currentTimeMs - (this.windowInSeconds * 1000) // X seconds ago

	// Flush old events from the sorted-set, everything older than `maxScoreMs` is out the window.
	rc.Cmd("ZREMRANGEBYSCORE", this.limit.String(), 0, maxScoreMs) //.Int()

	// Are we going to exceed limits, and is that a problem?
	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)
	currentValue, _ := this.getUnderlyingValue()
	currentValue++
	if currentLimit != -1 && currentValue > currentLimit && incrementing && this.limiter.IsHardLimitAtPointInTime() {
		return -1, fmt.Errorf("rate-limit exceeded for feature %s", this.limit)
	}

	// Either limits are not exceeded or it's not a problem: add this event to the sorted-set
	rc.Cmd("ZADD", this.limit.String(), currentTimeMs, currentTimeMs+rand.Int63()) //.Int()
	rc.Cmd("EXPIRE", this.limit.String(), this.windowInSeconds)                    //.Int()

	return currentValue, nil
}

// IncrementUsage tracks an increment in the usage rate.
//
// Returns an error if we couldn't due to limits, or redis failure.
// If the error is present do not allow the increment.
func (this *RateLimiterWindow) IncrementUsage() error {
	if this.redisPool == nil {
		return ErrRateLimiterNoRedis
	}
	_, err := this.attemptToChange(true)
	return err
}

// IsExceedingAtPointInTime determines if we're exceeding at this point in
// time.
func (this *RateLimiterWindow) IsExceedingAtPointInTime() (bool, error) {
	if this.redisPool == nil {
		return false, ErrRateLimiterNoRedis
	}
	currentValue, err := this.getUnderlyingValue()
	if err != nil {
		return false, err
	}
	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)
	return currentValue > currentLimit, nil
}

func (this *RateLimiterWindow) GetUsageAtPointInTime() (int, error) {
	if this.redisPool == nil {
		return 0, ErrRateLimiterNoRedis
	}
	return this.getUnderlyingValue()
}

func (this *RateLimiterWindow) GetMaxUsage() (int, error) {
	return this.getMaxValue()
}
