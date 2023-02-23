// Copyright 2010 The Go Authors. All rights reserved.
// Copyright (C) 2023  Luke Shumaker  <lukeshu@lukeshu.com>
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This file contains code copied verbatim from Go 1.20.1 net/textproto/textproto.go

// Package pytextproto is a lightly modified version of net/textproto.
//
// textproto.Reader.ReadMIMEHeader turns newlines (0x0A) in to spaces (0x20) when reading a
// multi-line value.  In Python package metadata, newlines are significant in multi-line values.  So
// I want through a whole lot of trouble to make this single line change:
//
//	-r.buf = append(r.buf, ' ')
//	+r.buf = append(r.buf, '\n') // MODIFIED
package pytextproto

import (
	"net/textproto"
)

type (
	ProtocolError = textproto.ProtocolError
	MIMEHeader    = textproto.MIMEHeader
)

func canonicalMIMEHeaderKey(s []byte) (string, bool) {
	for _, b := range s {
		if !validHeaderFieldByte(b) {
			return "", false
		}
	}
	return textproto.CanonicalMIMEHeaderKey(string(s)), true
}

// isASCIILetter is borrowed from net/textproto/textproto.go.
func isASCIILetter(b byte) bool {
	b |= 0x20 // make lower case
	return 'a' <= b && b <= 'z'
}
