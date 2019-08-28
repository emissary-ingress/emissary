// This file is used in place of the generated "envoy.go" on non
// linux/amd64 platforms (since we only compile Envoy for
// linux/amd64), and for the linter (since the linter essentially
// takes forever on the massive 50MB envoy.go, despite being
// syntactically simple).

// +build !linux !amd64 lint

package main

// Because main.go will unlink() the envoyBytes file after starting
// it, we can't rely on the shell being able to read the contents of
// the script, so we need to cram everything for a good error message
// right in to the shebang itself.  Also note that on most unixen, the
// shebang only gets one argument; it isn't fully field-separated (the
// only exception that comes to mind is Cygwin); otherwise `sh -c ...`
// would be really convenient.

const envoyBytes = "" +
	"#!/bin/sh error:_Builds_of_certified-envoy_for_platforms_other_than_linux/amd64_are_non-functional\n"
