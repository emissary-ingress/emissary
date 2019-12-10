package limiter

import (
	"github.com/pkg/errors"
)

// CountLimiter is a limiter that works on a "Count" limit type.
type CountLimiter interface {
	// GetUnderlyingValueAtPointInTime grabs the underlying count value.
	GetUnderlyingValueAtPointInTime() (int, error)
	// IsExceedingAtPointInTime determines if at this point in time
	// the count limiter is exceeding it's usage.
	IsExceedingAtPointInTime() (bool, error)
	// IncrementUsage will increment the usage of the counter, and return an
	// error if hard limit + limits exceeded, or if failure writing to redis.
	//
	// Takes a key to ensure we don't count the same key.
	IncrementUsage(key string) error
	// DecrementUsage will decrement the usage of the counter, and return an
	// error if there is a failure writing to redis.
	DecrementUsage(key string) error
}

type NoNoCounter struct {
}

func newNoNoCounter() *NoNoCounter {
	return &NoNoCounter{}
}

func (*NoNoCounter) GetUnderlyingValueAtPointInTime() (int, error) {
	return -1, errors.New("NoNoCounter does not allow GetUnderlyingValueAtPointInTime")
}

func (*NoNoCounter) IsExceedingAtPointInTime() (bool, error) {
	return true, nil
}

func (*NoNoCounter) IncrementUsage(key string) error {
	return errors.New("Does not allow for one more use")
}

func (*NoNoCounter) DecrementUsage(key string) error {
	return errors.New("Does not allow for decrementing value")
}
