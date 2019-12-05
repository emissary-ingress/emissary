package rfc6749

import (
	rfc6749common "github.com/datawire/apro/common/rfc6749"
)

// Scope represents an unordered list of scope-values as defined by §3.3.
type Scope = rfc6749common.Scope

// ParseScope de-serializes the set of scope-values from use as a parameter, per §3.3.
func ParseScope(str string) Scope {
	return rfc6749common.ParseScope(str)
}
