// +build pin

// Package ignore is mostly ignored; it just serves to pin the
// packages of external commands (i.e. things that we don't use as
// libraries) in to the module, so that `go mod tidy` won't make the
// `go.mod` file forget which version we want.
package ignore

import (
	// protoc-gen-go
	_ "github.com/golang/protobuf/protoc-gen-go"

	// controller-gen
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
