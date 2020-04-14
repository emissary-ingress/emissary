// +build pin

// Package ignore is mostly ignored; it just serves to pin the
// packages of external commands (i.e. things that we don't use as
// libraries) in to the module, so that `go mod tidy` won't make the
// `go.mod` file forget which version we want.
package ignore

import (
	// protoc-gen-validate
	_ "github.com/envoyproxy/protoc-gen-validate"
	// list at least 1 package from each module mentioned in
	// protoc-gen-validate's Gopkg.lock
	_ "github.com/gogo/protobuf/proto"
	_ "github.com/golang/protobuf/proto"
	_ "github.com/iancoleman/strcase"
	_ "github.com/lyft/protoc-gen-star"
	_ "golang.org/x/net/context"

	// protoc-gen-gogofast
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"

	// protoc-gen-go-json
	_ "github.com/mitchellh/protoc-gen-go-json"

	// other
	_ "github.com/datawire/ambassador/pkg/ignore/sub"
)
