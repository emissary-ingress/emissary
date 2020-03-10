package limiter

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
)

const deleteScript = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`

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
	// local copy of the encrypted key value
	localValueCopy string
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
		"",
	}, nil
}

// getUnderlyingValue returns the current underlying value.
//
// NOTE: for speed this is done without taking a lock which means
// it may be about to change.
//
// When shooting for perfect accuracy make sure to use a method
// that takes the redis lock.
func (this *CountLimiterImpl) GetUnderlyingValueAtPointInTime() (int, error) {
	resp, err := this.redisPool.Cmd("GET", this.limit.String()).Str()
	if err != nil {
		if err == redis.ErrRespNil || resp == "" {
			return 0, err
		}
		return -1, err
	}

	decryptedValue := this.decryptString(resp)
	keys := strings.Split(decryptedValue, ",")
	value := len(keys)
	this.recordMaxValue(value)
	return value, nil
}

// recordMaxValue records the maximum observed value in the last 24 hours.
func (this *CountLimiterImpl) recordMaxValue(value int) {
	previousValue, _ := this.getMaxValue()
	// if the new value is greater than the previously recorded maxValue,
	// set a "limit-m" redis key with the new maxValue and have it expire in 24h.
	if value > previousValue {
		this.redisPool.Cmd("SET", this.limit.String()+"-m", value, "EX", "86400")
	}
}

// getMaxValue returns the maximum observed value.
func (this *CountLimiterImpl) getMaxValue() (int, error) {
	resp, err := this.redisPool.Cmd("GET", this.limit.String()+"-m").Str()
	if err != nil {
		if err == redis.ErrRespNil || resp == "" {
			return 0, err
		}
		return -1, err
	}
	return strconv.Atoi(resp)
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
	rc.Cmd("EVAL", deleteScript, "1", this.limit.String()+"-lock", randStr)
}

// encrypt the redis string value, and save a local copy in memory if we are ever unable to decrypt it.
func (this *CountLimiterImpl) encryptString(val string) (string, error) {
	this.localValueCopy = val
	return this.cryptoEngine.EncryptString(val)
}

// decrypt the redis string value, and use a local copy of a previous representation if we are unable to decrypt it.
func (this *CountLimiterImpl) decryptString(val string) string {
	decryptedStr, err := this.cryptoEngine.DecryptString(val)
	if err != nil {
		this.redisPool.Cmd("DEL", this.limit.String())
		return this.localValueCopy
	}
	return decryptedStr
}

// I still can't believe golang doesn't have a builtin check for
// contains in a slice.
func containsStrSlice(slice []string, item string) (bool, int) {
	for idx, s := range slice {
		if s == item {
			return true, idx
		}
	}

	return false, -1
}

func removeStrSlice(slice []string, idx int) []string {
	slice[len(slice)-1], slice[idx] = slice[idx], slice[len(slice)-1]
	return slice[:len(slice)-1]
}

// attemptToIncrement implements all the logic for attempting to increment values
// taking into account limits
func (this *CountLimiterImpl) attemptToChange(incrementing bool, key string) (int, error) {
	if strings.Contains(key, ",") {
		return -1, errors.New("Key cannot contain a ','")
	}

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
		if err != redis.ErrRespNil {
			return -1, err
		}
	}

	var keys []string
	if err == redis.ErrRespNil || val == "" {
		keys = make([]string, 0)
	} else {
		decryptedStr := this.decryptString(val)
		keys = strings.Split(decryptedStr, ",")
	}

	doesContain, idx := containsStrSlice(keys, key)
	if doesContain && !incrementing {
		keys = removeStrSlice(keys, idx)
	} else if !doesContain && incrementing {
		keys = append(keys, key)
	}

	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)

	// Are we going to exceed limits, and is that a problem?
	if len(keys) > currentLimit && this.limiter.IsHardLimitAtPointInTime() {
		// If we're decrementing than just ignore the limit value increase.
		// If you wanna get closer to your hard limit that's more than fine.
		if incrementing {
			return -1, errors.New("No room for that service")
		}
	}

	joinedKey := strings.Join(keys, ",")
	newEncryptedValue, err := this.encryptString(joinedKey)
	if err != nil {
		return -1, err
	}
	err = rc.Cmd("SET", this.limit.String(), newEncryptedValue).Err
	return len(keys), err
}

// IncrementUsage tracks an increment in the usage count.
//
// Returns an error if we couldn't due to limits, or redis failure.
// If the error is present do not allow the increment.
func (this *CountLimiterImpl) IncrementUsage(key string) error {
	_, err := this.attemptToChange(true, key)
	return err
}

// DecrementUsage tracks a decrement in the usage count.
func (this *CountLimiterImpl) DecrementUsage(key string) error {
	_, err := this.attemptToChange(false, key)
	return err
}

// IsExceedingAtPointInTime determines if we're exceeding at this point in
// time.
func (this *CountLimiterImpl) IsExceedingAtPointInTime() (bool, error) {
	currentValue, err := this.GetUnderlyingValueAtPointInTime()
	if err != nil {
		return false, err
	}
	currentLimit := this.limiter.GetLimitValueAtPointInTime(this.limit)
	return currentValue > currentLimit, nil
}

func (this *CountLimiterImpl) GetUsageAtPointInTime() (int, error) {
	return this.GetUnderlyingValueAtPointInTime()
}

func (this *CountLimiterImpl) GetMaxUsage() (int, error) {
	return this.getMaxValue()
}
