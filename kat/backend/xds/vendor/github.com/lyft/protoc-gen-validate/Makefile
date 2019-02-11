empty :=
space := $(empty) $(empty)
PACKAGE := github.com/lyft/protoc-gen-validate

# protoc-gen-go parameters for properly generating the import path for PGV
VALIDATE_IMPORT := Mvalidate/validate.proto=${PACKAGE}/validate
GO_IMPORT_SPACES := ${VALIDATE_IMPORT},\
	Mgoogle/protobuf/any.proto=github.com/golang/protobuf/ptypes/any,\
	Mgoogle/protobuf/duration.proto=github.com/golang/protobuf/ptypes/duration,\
	Mgoogle/protobuf/struct.proto=github.com/golang/protobuf/ptypes/struct,\
	Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp,\
	Mgoogle/protobuf/wrappers.proto=github.com/golang/protobuf/ptypes/wrappers,\
	Mgoogle/protobuf/descriptor.proto=github.com/golang/protobuf/protoc-gen-go/descriptor,\
	Mgogoproto/gogo.proto=${PACKAGE}/gogoproto
GO_IMPORT:=$(subst $(space),,$(GO_IMPORT_SPACES))

# protoc-gen-gogo parameters
GOGO_IMPORT_SPACES := ${VALIDATE_IMPORT},\
	Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
	Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/types,\
	Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto
GOGO_IMPORT:=$(subst $(space),,$(GOGO_IMPORT_SPACES))

.PHONY: build
build: validate/validate.pb.go
	# generates the PGV binary and installs it into $$GOPATH/bin
	go install .

.PHONY: bazel
bazel:
	# generate the PGV plugin with Bazel
	bazel build //tests/...

.PHONY: gazelle
gazelle:
	# runs gazelle against the codebase to generate Bazel BUILD files
	bazel run //:gazelle

.PHONY: lint
lint:
	# lints the package for common code smells
	which golint || go get -u github.com/golang/lint/golint
	test -z "$(gofmt -d -s ./*.go)" || (gofmt -d -s ./*.go && exit 1)
	# golint -set_exit_status
	go tool vet -all -shadow -shadowstrict *.go

.PHONY: quick
quick:
	# runs all tests without the race detector or coverage percentage
	go test

.PHONY: tests
tests:
	# runs all tests against the package with race detection and coverage percentage
	go test -race -cover
	# tests validate proto generation
	bazel build //validate:go_default_library \
		&& diff $$(bazel info bazel-genfiles)/validate/validate.pb.go validate/validate.pb.go

.PHONY: cover
cover:
	# runs all tests against the package, generating a coverage report and opening it in the browser
	go test -race -covermode=atomic -coverprofile=cover.out
	go tool cover -html cover.out -o cover.html
	open cover.html

gogofast:
	go build -o $@ vendor/github.com/gogo/protobuf/protoc-gen-gogofast/main.go

.PHONY: harness
harness: tests/harness/go/harness.pb.go tests/harness/gogo/harness.pb.go tests/harness/go/main/go-harness tests/harness/gogo/main/go-harness tests/harness/cc/cc-harness
 	# runs the test harness, validating a series of test cases in all supported languages
	go run ./tests/harness/executor/*.go

.PHONY: bazel-harness
bazel-harness:
	# runs the test harness via bazel
	bazel run //tests/harness/executor:executor

.PHONY: kitchensink
kitchensink: gogofast
	# generates the kitchensink test protos
	rm -r tests/kitchensink/go || true
	mkdir -p tests/kitchensink/go
	rm -r tests/kitchensink/gogo || true
	mkdir -p tests/kitchensink/gogo
	cd tests/kitchensink && \
	protoc \
		-I . \
		-I ../.. \
		--go_out="${GO_IMPORT}:./go" \
		--validate_out="lang=go:./go" \
		--plugin=protoc-gen-gogofast=$(shell pwd)/gogofast \
		--gogofast_out="${GOGO_IMPORT}:./gogo" \
		--validate_out="lang=gogo:./gogo" \
		`find . -name "*.proto"`
	cd tests/kitchensink/go && go build .
	cd tests/kitchensink/gogo && go build .

.PHONY: testcases
testcases: gogofast
	# generate the test harness case protos
	rm -r tests/harness/cases/go || true
	mkdir tests/harness/cases/go
	rm -r tests/harness/cases/other_package/go || true
	mkdir tests/harness/cases/other_package/go
	rm -r tests/harness/cases/gogo || true
	mkdir tests/harness/cases/gogo
	rm -r tests/harness/cases/other_package/gogo || true
	mkdir tests/harness/cases/other_package/gogo
	# protoc-gen-go makes us go a package at a time
	cd tests/harness/cases/other_package && \
	protoc \
		-I . \
		-I ../../../.. \
		--go_out="${GO_IMPORT}:./go" \
		--plugin=protoc-gen-gogofast=$(shell pwd)/gogofast \
		--gogofast_out="${GOGO_IMPORT}:./gogo" \
		--validate_out="lang=go:./go" \
		--validate_out="lang=gogo:./gogo" \
		./*.proto
	cd tests/harness/cases && \
	protoc \
		-I . \
		-I ../../.. \
		--go_out="Mtests/harness/cases/other_package/embed.proto=${PACKAGE}/tests/harness/cases/other_package/go,${GO_IMPORT}:./go" \
		--plugin=protoc-gen-gogofast=$(shell pwd)/gogofast \
		--gogofast_out="Mtests/harness/cases/other_package/embed.proto=${PACKAGE}/tests/harness/cases/other_package/gogo,${GOGO_IMPORT}:./gogo" \
		--validate_out="lang=go:./go" \
		--validate_out="lang=gogo:./gogo" \
		./*.proto

.PHONY: update-vendor
update-vendor:
	# updates the vendored dependencies using the Go Dep tool
	dep ensure -update
	$(MAKE) gazelle

tests/harness/go/harness.pb.go:
	# generates the test harness protos
	cd tests/harness && protoc -I . \
		--go_out="${GO_IMPORT}:./go" harness.proto

tests/harness/gogo/harness.pb.go: gogofast
	# generates the test harness protos
	cd tests/harness && protoc -I . \
		--plugin=protoc-gen-gogofast=$(shell pwd)/gogofast \
		--gogofast_out="${GOGO_IMPORT}:./gogo" harness.proto

.PHONY: tests/harness/go/main/go-harness
tests/harness/go/main/go-harness:
	# generates the go-specific test harness
	go build -o ./tests/harness/go/main/go-harness ./tests/harness/go/main

.PHONY: tests/harness/gogo/main/go-harness
tests/harness/gogo/main/go-harness:
	# generates the gogo-specific test harness
	go build -o ./tests/harness/gogo/main/go-harness ./tests/harness/gogo/main

tests/harness/cc/cc-harness: tests/harness/cc/harness.cc
	# generates the C++-specific test harness
	# use bazel which knows how to pull in the C++ common proto libraries
	bazel build //tests/harness/cc:cc-harness
	cp bazel-bin/tests/harness/cc/cc-harness $@
	chmod 0755 $@

.PHONY: ci
ci: lint build tests kitchensink testcases harness bazel-harness
