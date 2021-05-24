// MODIFIED: This file is a verbatim subset of Go 1.15.6 context/context.go,
// MODIFIED: except for lines marked "MODIFIED".
//
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dcontext // MODIFIED

import (
	. "context"           // MODIFIED
	reflectlite "reflect" // MODIFIED
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
