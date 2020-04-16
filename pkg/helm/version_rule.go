package helm

import (
	"github.com/Masterminds/semver"
)

// A chart version rule defines the rules for allowed chart versions
type ChartVersionRule struct {
	s          string
	constraint *semver.Constraints
}

func NewChartVersionRule(ver string) (ChartVersionRule, error) {
	s := ver
	if len(ver) == 0 {
		s = "*"
	}
	constraint, err := semver.NewConstraint(s)
	if err != nil {
		return ChartVersionRule{}, err
	}
	return ChartVersionRule{
		constraint: constraint,
		s:          s,
	}, nil
}

// Allowed returns true if the version provided is allowed by the ChartVersionRule
func (cv ChartVersionRule) Allowed(s string) (bool, error) {
	test, err := semver.NewVersion(s)
	if err != nil {
		return false, err
	}

	if cv.constraint.Check(test) {
		return true, nil
	}

	return false, nil
}

func (cv ChartVersionRule) String() string {
	return cv.s
}

// MoreRecentThan returns True if a is more recent than b
func MoreRecentThan(a, b string) (bool, error) {
	aver, err := semver.NewVersion(a)
	if err != nil {
		return false, err
	}
	bver, err := semver.NewVersion(b)
	if err != nil {
		return false, err
	}

	return aver.GreaterThan(bver), nil
}

// Equal returns True if a is equal to b
func Equal(a, b string) (bool, error) {
	aver, err := semver.NewVersion(a)
	if err != nil {
		return false, err
	}
	bver, err := semver.NewVersion(b)
	if err != nil {
		return false, err
	}

	return aver.Equal(bver), nil
}
