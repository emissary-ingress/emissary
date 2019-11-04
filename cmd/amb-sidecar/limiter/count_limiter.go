package limiter

import (
	"github.com/pkg/errors"
)

// CountLimiter is a limiter that works on a "Count" limit type.
type CountLimiter interface {
	// IsExceedingAtPointInTime determines if at this point in time
	// the count limiter is exceeding it's usage.
	IsExceedingAtPointInTime() (bool, error)
	// IncrementUsage will increment the usage of the counter, and return an
	// error if hard limit + limits exceeded, or if failure writing to redis.
	IncrementUsage() error
	// DecrementUsage will decrement the usage of the counter, and return an
	// error if there is a failure writing to redis.
	DecrementUsage() error
}

type NoNoCounter struct {
}

func newNoNoCounter() *NoNoCounter {
	return &NoNoCounter{}
}

func (*NoNoCounter) IsExceedingAtPointInTime() (bool, error) {
	return true, nil
}

func (*NoNoCounter) IncrementUsage() error {
	return errors.New("Does not allow for one more use")
}

func (*NoNoCounter) DecrementUsage() error {
	return errors.New("Does not allow for decrementing value")
}
