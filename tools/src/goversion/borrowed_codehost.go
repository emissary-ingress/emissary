// This file is a verbatim subset of Go 1.17 cmd/go/internal/modfetch/codehost/codehost.go

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/fs"
)

// AllHex reports whether the revision rev is entirely lower-case hexadecimal digits.
func AllHex(rev string) bool {
	for i := 0; i < len(rev); i++ {
		c := rev[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

// ShortenSHA1 shortens a SHA1 hash (40 hex digits) to the canonical length
// used in pseudo-versions (12 hex digits).
func ShortenSHA1(rev string) string {
	if AllHex(rev) && len(rev) == 40 {
		return rev[:12]
	}
	return rev
}

// UnknownRevisionError is an error equivalent to fs.ErrNotExist, but for a
// revision rather than a file.
type UnknownRevisionError struct {
	Rev string
}

func (e *UnknownRevisionError) Error() string {
	return "unknown revision " + e.Rev
}
func (UnknownRevisionError) Is(err error) bool {
	return err == fs.ErrNotExist
}
