module github.com/datawire/ambassador

go 1.13

require (
	github.com/envoyproxy/protoc-gen-validate v0.0.15-0.20190405222122-d6164de49109
	github.com/gogo/googleapis v1.1.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.2.1-0.20190205222052-c823c79ea157 // indirect
	github.com/gorilla/websocket v1.4.0
	google.golang.org/grpc v1.19.1
	istio.io/gogo-genproto v0.0.0-20190614210408-e88dc8b0e4db
)

// This is go-control-plane.git's go.mod
require (
	github.com/envoyproxy/protoc-gen-validate v0.0.0-20190405222122-d6164de49109
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	golang.org/x/net v0.0.0-20190503192946-f4e77d36d62c // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	golang.org/x/sys v0.0.0-20190508220229-2d0786266e9c // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.19.1
	istio.io/gogo-genproto v0.0.0-20190731221249-06e20ada0df2
)

// But we need a newer istio.io/gogo-genproto; see the discussion in
// Makefile.
require (
	istio.io/gogo-genproto v0.0.0-20190904133402-ee07f2785480
)

// These are the versions mentioned protoc-gen-validate's Gopkg.lock;
// as reported by running `go mod init`.
require (
	github.com/gogo/protobuf v1.1.1
	github.com/golang/protobuf v1.2.0
	github.com/iancoleman/strcase v0.0.0-20180726023541-3605ed457bf7
	github.com/lyft/protoc-gen-star v0.4.4
	github.com/spf13/afero v1.1.2
	golang.org/x/net v0.0.0-20181023162649-9b4f9f5ad519
	golang.org/x/text v0.3.0
)
