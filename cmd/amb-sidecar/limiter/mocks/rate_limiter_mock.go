package mocks

import (
	"github.com/pkg/errors"
)

type MockRate struct {
	limitAt     int
	currentRate int
}

func NewMockLimitRate(limit int) *MockRate {
	return &MockRate{
		limitAt:     limit,
		currentRate: 0,
	}
}

func (mc *MockRate) IsExceedingAtPointInTime() (bool, error) {
	return mc.currentRate > mc.limitAt, nil
}

func (mc *MockRate) IncrementUsage() error {
	if mc.currentRate >= mc.limitAt {
		return errors.New("Does not allow for one more use")
	} else {
		mc.currentRate++
		return nil
	}
}
