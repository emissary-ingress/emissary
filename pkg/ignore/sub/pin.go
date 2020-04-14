// Package ignore/sub is mostly ignored; it just serves to pin select
// packages so that `go mod tidy` won't make the `go.mod` file forget
// which version we want.  See the comment in `go.mod` for more
// information.
package sub

import (
	_ "golang.org/x/sys/unix"
	_ "github.com/gogo/protobuf/proto"
)
