// Package ignore can be ignored.  It exists to pin the ratelimit
// repo, so `go mod tidy` knows about it.
package ignore

import (
	_ "github.com/lyft/ratelimit/src/service_cmd/runner" // pin this in `go.mod`
)
