// MODIFIED: META: This file is a verbatim subset of Go 1.15.14 context/context.go,
// MODIFIED: META: except for lines marked "MODIFIED".

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dcontext // MODIFIED: FROM: package context

import (
	. "context"           // MODIFIED: ADDED
	reflectlite "reflect" // MODIFIED: FROM: "internal/reflectlite"
)

type stringer interface {
	String() string
}

func contextName(c Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	return reflectlite.TypeOf(c).String()
}
