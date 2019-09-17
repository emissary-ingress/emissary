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

// Pin versions of external commands (i.e. things that we don't use as
// libraries).  We explicitly 'replace' them so that `go mod tidy`
// can't make the file forget which version we want.
replace (
	// Go a few commits after v0.0.14 to get the
	// github.com/{lyft→envoyproxy}/protoc-gen-validate rename
	github.com/envoyproxy/protoc-gen-validate => github.com/envoyproxy/protoc-gen-validate v0.0.15-0.20190405222122-d6164de49109

	github.com/gogo/protobuf => github.com/gogo/protobuf v1.2.1
)
