module github.com/datawire/ambassador

go 1.12

require (
	github.com/datawire/kat-backend v1.4.3 // indirect
	github.com/envoyproxy/data-plane-api v0.0.0-20190403155806-897f5b09bbe3
	github.com/envoyproxy/protoc-gen-validate v0.0.15-0.20190405222122-d6164de49109
	github.com/gogo/googleapis v1.1.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.2.1-0.20190205222052-c823c79ea157
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365 // indirect
	github.com/lyft/protoc-gen-star v0.4.10 // indirect
	github.com/sirupsen/logrus v1.0.4
	github.com/spf13/afero v1.2.2 // indirect
	golang.org/x/crypto v0.0.0-20180222182404-49796115aa4b
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd
	golang.org/x/sys v0.0.0-20180830151530-49385e6e1522
	golang.org/x/text v0.3.0
	google.golang.org/genproto v0.0.0-20180831171423-11092d34479b
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
