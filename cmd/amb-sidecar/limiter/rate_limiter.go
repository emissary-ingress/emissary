package limiter

import (
	"github.com/pkg/errors"
)

// RateLimiter is a limiter that works on a "Rate Per X" limit type.
type RateLimiter interface {
	// IsExceedingAtPointInTime determines if at this point in time
	// the rate limiter is exceeding it's usage.
	IsExceedingAtPointInTime() (bool, error)
	// IncrementUsage will increment the usage of the rate limiter, and return an
	// error if hard limit + limits exceeded, or if failure writing to redis.
	IncrementUsage() error
}

type NoNoRate struct {
}

func newNoNoRate() *NoNoRate {
	return &NoNoRate{}
}

func (*NoNoRate) IsExceedingAtPointInTime() (bool, error) {
	return true, nil
}

func (*NoNoRate) IncrementUsage() error {
	return errors.New("Does not allow for one more use")
}
