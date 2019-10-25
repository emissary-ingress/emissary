NAME ?= ambassador

OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

include $(OSS_HOME)/builder/builder.mk

$(call module,ambassador,$(OSS_HOME))

generate: ## Update generated sources that get committed to git
generate: pkg/api/kat/echo.pb.go
generate: pkg/api/getambassador.io/v2/Host.pb.go
generate: python/ambassador/proto/v2/Host_pb2.py
generate-clean: ## Delete generated sources that get committed to git
generate-clean:
	rm -rf pkg/api
.PHONY: generate generate-clean

## Install protoc

GOOS=$(shell go env GOOS)
GOHOSTOS=$(shell go env GOHOSTOS)
GOHOSTARCH=$(shell go env GOHOSTARCH)

GRPC_WEB_VERSION = 1.0.3
GRPC_WEB_PLATFORM = $(GOOS)-x86_64

bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web:
	mkdir -p $(@D)
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@

# The version numbers of `protoc` (in this Makefile),
# `protoc-gen-gogofast` (in go.mod), and `protoc-gen-validate` (in
# go.mod) are based on
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/Dockerfile.ci
# (since that commit most closely corresponds to our ENVOY_COMMIT).
# Additionally, the package names of those programs are mentioned in
# ./go/pin.go, so that `go mod tidy` won't make the go.mod file forget
# about them.

PROTOC_VERSION = 3.5.1
PROTOC_PLATFORM = $(patsubst darwin,osx,$(GOHOSTOS))-$(patsubst amd64,x86_64,$(patsubst 386,x86_32,$(GOHOSTARCH)))

bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc:
	mkdir -p $(@D)
	set -o pipefail; curl --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip | bsdtar -x -f - -O bin/protoc > $@
	chmod 755 $@

bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast: go.mod
	mkdir -p $(@D)
	go build -o $@ github.com/gogo/protobuf/protoc-gen-gogofast

bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-validate: go.mod
	mkdir -p $(@D)
	go build -o $@ github.com/envoyproxy/protoc-gen-validate

clobber: _makefile_clobber
_makefile_clobber:
	rm -rf bin_*/
.PHONY: _makefile_clobber

## Generated sources

vendor:
	go mod vendor
.PHONY: vendor

pkg/api/kat/echo.pb.go: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast
	mkdir -p $(@D)
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast --gogofast_out=plugins=grpc:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_grpc_web_pb.js: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_pb.js: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--js_out=import_style=commonjs:$(@D) \
		$(CURDIR)/$<

pkg/api/getambassador.io/v2/Host.pb.go python/ambassador/proto/v2/Host_pb2.py: api/getambassador.io/v2/Host.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast vendor
	mkdir -p pkg/api/getambassador.io/v2 python/ambassador/proto/v2
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/vendor \
		--proto_path=$(CURDIR)/api/getambassador.io/v2 \
		--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast --gogofast_out=plugins=grpc:pkg/api/getambassador.io/v2 \
		--python_out=python/ambassador/proto/v2 \
		$(CURDIR)/$<

# Configure GNU Make itself
.SECONDARY:
.DELETE_ON_ERROR:
