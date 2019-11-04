package limiter

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
)

// CountLimiter limits on counts.
type CountLimiterImpl struct {
	// A connection to the redis instance.
	//
	// This is required for a count limiter.
	redisPool *pool.Pool
	// Need to take an instance of the limiter since the license key can be
	// reloaded.
	limiter Limiter
	// limit is the actual limit to enforce
	limit *licensekeys.Limit
	// cryptoEngine is used for encrypting the values store in redis.
	cryptoEngine *LimitCrypto
}

// newCountLimiter creates a new limit based on a count of items.
//
// redisPool: a connection to the redisPool
// limit: the actual limit value.
// crypto: the engine used for actually encrypting values in redis.
func newCountLimiterImpl(redisPool *pool.Pool, limiter Limiter, limit *licensekeys.Limit, cryptoEngine *LimitCrypto) (*CountLimiterImpl, error) {
	if redisPool == nil {
		return nil, errors.New("Need a redis pool to enforce a counter limit")
	}

	return &CountLimiterImpl{
		redisPool,
		limiter,
		limit,
		cryptoEngine,
	}, nil
}

// getUnderlyingValue returns the current underlying value.
//
// NOTE: for speed this is done without taking a lock which means
// it may be about to change.
//
// When shooting for perfect accuracy make sure to use a method
// that takes the redis lock.
func (this *CountLimiterImpl) getUnderlyingValue() (int, error) {
	resp, err := this.redisPool.Cmd("GET", this.limit.String()).Str()
	if err != nil {
		return -1, err
	}
	decryptedValue, err := this.cryptoEngine.DecryptString(resp)
	if err != nil {
		return -1, err
	}
	val, err := strconv.ParseInt(decryptedValue, 16, 32)
	if err != nil {
		return -1, err
	}

	return int(val), nil
}

// attemptAcquireLock attempts to acquire a lock for a redis client.
func (this *CountLimiterImpl) attemptAcquireLock(rc *redis.Client) (bool, string) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return false, ""
	}
	randStr := uuid.String()

	didSet := false
	for i := 0; i < 3; i++ {
		resp, err := rc.Cmd("SET", this.limit.String()+"-lock", randStr, "NX", "PX", "30000").Str()
		if err == nil && resp == "OK" {
			didSet = true
			break
		}

		time.Sleep(3000 * time.Millisecond)
		continue
	}
	return didSet, randStr
}

// releaseLock is a best effort release lock. the key will expire if we fail here
// so it's okay to be best effort.
func (this *CountLimiterImpl) releaseLock(rc *redis.Client, randStr string) {
	randVal, err := rc.Cmd("GET", this.limit.String()+"-lock").Str()
	// Let key expire
	if err != nil {
		return
	}
	if randVal == randStr {
		// Attempt to delete key
		rc.Cmd("DEL", this.limit.String()+"-lock")
	}
}

// attemptToIncrement implements all the logic for attempting to increment values
// taking into account limits
func (this *CountLimiterImpl) attemptToChange(incrementing bool) (int, error) {
	rc, err := this.redisPool.Get()
	if err != nil {
		return -1, err
	}
	defer this.redisPool.Put(rc)

	didSet, randStr := this.attemptAcquireLock(rc)
	if !didSet {
		return -1, errors.New("Failed to acquire lock")
	}
	defer this.releaseLock(rc, randStr)

	val, err := rc.Cmd("GET", this.limit.String()).Str()
	if err != nil {
		return -1, err
	}
	decryptedStr, err := this.cryptoEngine.DecryptString(val)
	if err != nil {
		return -1, errors.New("Invalid current limit")
	}
	currentUsage, err := strconv.Atoi(decryptedStr)
	if err != nil {
		return -1, err
	}

	newUsage := currentUsage
	if incrementing {
		newUsage = newUsage + 1
	} else {
		newUsage = newUsage - 1
	}
	// Did we overflow/underflow?
	if newUsage < 0 {
		return -1, errors.New("Int32 overflow")
	}
	// Are we going to exceed limits, and is that a problem?
	currentLimit := this.limiter.GetLimitValueAtPointInTime(*this.limit)
	if currentLimit != -1 && newUsage > currentLimit && this.limiter.IsHardLimitAtPointInTime() {
		// If we're decrementing than just ignore the limit value increase.
		// If you wanna get closer to your hard limit that's more than fine.
		if incrementing {
			return -1, errors.New("No room for that service")
		}
	}

	newEncryptedValue, err := this.cryptoEngine.EncryptString(strconv.Itoa(newUsage))
	err = rc.Cmd("SET", this.limit.String(), newEncryptedValue).Err
	return newUsage, err
}

// IncrementUsage tracks an increment in the usage count.
//
// Returns an error if we couldn't due to limits, or redis failure.
// If the error is present do not allow the increment.
func (this *CountLimiterImpl) IncrementUsage() error {
	_, err := this.attemptToChange(true)
	return err
}

// DecrementUsage tracks a decrement in the usage count.
func (this *CountLimiterImpl) DecrementUsage() error {
	_, err := this.attemptToChange(false)
	return err
}

// IsExceedingAtPointInTime determines if we're exceeding at this point in
// time.
func (this *CountLimiterImpl) IsExceedingAtPointInTime() (bool, error) {
	currentValue, err := this.getUnderlyingValue()
	if err != nil {
		return false, err
	}
	currentLimit := this.limiter.GetLimitValueAtPointInTime(*this.limit)
	return currentValue > currentLimit, nil
}
