package rfc6749

import (
	"strings"
)

// Scope represents an unordered list of scope-values as defined by §3.3.
type Scope map[string]struct{}

// String serializes the set of scope-values for use as a parameter, per §3.3.
func (scope Scope) String() string {
	strs := make([]string, 0, len(scope))
	for k := range scope {
		strs = append(strs, k)
	}
	return strings.Join(strs, " ")
}

// ParseScope de-serializes the set of scope-values from use as a parameter, per §3.3.
func ParseScope(str string) Scope {
	strs := strings.Split(str, " ")
	ret := make(Scope, len(strs))
	for _, s := range strs {
		if s != "" {
			ret[s] = struct{}{}
		}
	}
	return ret
}
