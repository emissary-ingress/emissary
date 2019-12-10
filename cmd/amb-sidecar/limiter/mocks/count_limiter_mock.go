package mocks

import (
	"github.com/pkg/errors"
)

type MockCounter struct {
	limitAt      int
	currentCount int
}

func NewMockLimitCounter(limit int) *MockCounter {
	return &MockCounter{
		limitAt:      limit,
		currentCount: 0,
	}
}

func (mc *MockCounter) GetUnderlyingValueAtPointInTime() (int, error) {
	return mc.currentCount, nil
}

func (mc *MockCounter) IsExceedingAtPointInTime() (bool, error) {
	return mc.currentCount > mc.limitAt, nil
}

func (mc *MockCounter) IncrementUsage(key string) error {
	if mc.currentCount >= mc.limitAt {
		return errors.New("Does not allow for one more use")
	} else {
		mc.currentCount++
		return nil
	}
}

func (mc *MockCounter) DecrementUsage(key string) error {
	mc.currentCount--
	return nil
}
