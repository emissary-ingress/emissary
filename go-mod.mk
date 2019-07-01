# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet to build Go programs using Go 1.11 modules.
#
## Eager inputs ##
#  - File: ./go.mod
#  - Variable: go.DISABLE_GO_TEST ?=
#  - Variable: go.PLATFORMS ?= $(GOOS)_$(GOARCH)
#
## Lazy inputs ##
#  - Variable: go.GOBUILD ?= go build
#  - Variable: go.LDFLAGS ?=
#  - Variable: go.GOLANG_LINT_VERSION ?= …
#  - Variable: go.GOLANG_LINT_FLAGS ?= …$(wildcard .golangci.yml .golangci.toml .golangci.json)…
#  - Variable: CI ?=
#
## Outputs ##
#  - Variable: export GO111MODULE = on
#  - Variable: NAME ?= $(notdir $(go.module))
#
#  - Variable: go.goversion = $(patsubst go%,%,$(filter go1%,$(shell go version)))
#  - Variable: go.module = EXAMPLE.COM/YOU/YOURREPO
#  - Variable: go.bins = List of "main" Go packages
#  - Variable: go.pkgs ?= ./...
#
#  - Function: go.list = like $(shell go list $1), but ignores nested Go modules and doesn't download things
#  - Function: go.bin.rule = Only use this if you know what you are doing
#
#  - Targets: bin_$(OS)_$(ARCH)/$(CMD)
#  - Targets: bin_$(OS)_$(ARCH)/$(CMD).opensource.tar.gz
#  - .PHONY Target: go-get
#  - .PHONY Target: go-build
#  - .PHONY Target: go-lint
#  - .PHONY Target: go-fmt
#  - .PHONY Target: go-test
#
## common.mk targets ##
#  - build
#  - lint
#  - format
#  - check
#  - clean
#  - clobber
#
# `go.PLATFORMS` is a list of OS_ARCH pairs that specifies which
# platforms `make build` should compile for.  Unlike most variables,
# it must be specified *before* including go-workspace.mk.
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_go-mod.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_go-mod.mk))common.mk

#
# Configure the `go` command

go.goversion = $(call lazyonce,go.goversion,$(patsubst go%,%,$(filter go1%,$(shell go version))))

export GO111MODULE = on

# Disable parallel builds on Go 1.11; the module cache is not
# concurrency-safe.  This is fixed in 1.12.
ifneq ($(filter 1.11.%,$(go.goversion)),)
.NOTPARALLEL:
endif

#
# Set default values for input variables

go.GOBUILD ?= go build
go.DISABLE_GO_TEST ?=
go.LDFLAGS ?=
go.PLATFORMS ?= $(GOOS)_$(GOARCH)
go.GOLANG_LINT_VERSION ?= 1.17.1
go.GOLANG_LINT_FLAGS ?= $(if $(wildcard .golangci.yml .golangci.toml .golangci.json),,--disable-all --enable=gofmt --enable=govet)
CI ?=

#
# Set output variables and functions

NAME ?= $(notdir $(go.module))

go.module := $(shell GO111MODULE=on go mod edit -json | jq -r .Module.Path)
ifneq ($(words $(go.module)),1)
  $(error Could not extract $$(go.module) from ./go.mod)
endif

# It would be simpler to create this list if we could use module-aware
# `go list`:
#
#     go.bins := $(shell GO111MODULE=on go list -f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)
#
# But alas, we can't do that, as that would cause the module system go
# ahead and download dependencies.  We don't want Go to do that at
# Makefile-parse-time; what if we're running `make clean`?
#
# So instead, we must deal with this abomination.
_go.submods := $(patsubst %/go.mod,%,$(shell git ls-files '*/go.mod'))
go.list = $(call path.addprefix,$(go.module),\
                                $(filter-out $(foreach d,$(_go.submods),$d $d/%),\
                                             $(call path.trimprefix,_$(CURDIR),\
                                                                    $(shell GOPATH=/bogus GO111MODULE=off go list $1))))
go.bins := $(call go.list,-f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)

go.pkgs ?= ./...

#
# Rules

go-get: ## (Go) Download Go dependencies
	go mod download
.PHONY: go-get

vendor: go-get FORCE
	go mod vendor
	@test -d $@
vendor.hash: vendor
	find vendor -type f -exec sha256sum {} + | sort | sha256sum | $(WRITE_IFCHANGED) $@

$(dir $(_go-mod.mk))go1%.src.tar.gz:
	curl -o $@ --fail https://dl.google.com/go/$(@F)

_go.mkopensource = $(dir $(_go-mod.mk))go-opensource

# Usage: $(eval $(call go.bin.rule,BINNAME,GOPACKAGE))
define go.bin.rule
bin_%/.$1.stamp: go-get FORCE
	$$(go.GOBUILD) $$(if $$(go.LDFLAGS),--ldflags $$(call quote.shell,$$(go.LDFLAGS))) -o $$@ $2
bin_%/$1: bin_%/.$1.stamp
	$$(COPY_IFCHANGED) $$< $$@

bin_%/.$1.deps: bin_%/$1
	go list -deps -f='{{.Module}}' $2 | LC_COLLATE=C sort -u > $$@
bin_%/$1.opensource.tar.gz: bin_%/.$1.deps vendor.hash $$(_go.mkopensource) $$(dir $$(_go-mod.mk))go$$(go.goversion).src.tar.gz
	$$(if $$(CI),@set -e; if test -e $$@; then echo 'This should not happen in CI: $$@ rebuild triggered by $$+' >&2; false; fi)
	$$(_go.mkopensource) --output=$$@ --package=$2 --depsfile=$$< --gotar=$$(dir $$(_go-mod.mk))go$$(go.goversion).src.tar.gz
endef

_go.bin.name = $(notdir $(_go.bin))
_go.bin.pkg = $(_go.bin)
$(foreach _go.bin,$(go.bins),$(eval $(call go.bin.rule,$(_go.bin.name),$(_go.bin.pkg))))
go-build: $(foreach _go.PLATFORM,$(go.PLATFORMS),$(foreach _go.bin,$(go.bins), bin_$(_go.PLATFORM)/$(_go.bin.name).opensource.tar.gz ))

go-build: ## (Go) Build the code with `go build`
.PHONY: go-build

$(dir $(_go-mod.mk))golangci-lint: $(_go-mod.mk)
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(@D) v$(go.GOLANG_LINT_VERSION)

go-lint: ## (Go) Check the code with `golangci-lint`
go-lint: $(dir $(_go-mod.mk))golangci-lint go-get
	$(dir $(_go-mod.mk))golangci-lint run $(go.GOLANG_LINT_FLAGS) $(go.pkgs)
.PHONY: go-lint

go-fmt: ## (Go) Fixup the code with `go fmt`
go-fmt: go-get
	go fmt $(go.pkgs)
.PHONY: go-fmt

go-test: ## (Go) Check the code with `go test`
go-test: go-build
ifeq ($(go.DISABLE_GO_TEST),)
	$(MAKE) $(dir $(_go-mod.mk))go-test.tap.summary
endif

$(dir $(_go-mod.mk))go-test.tap: FORCE
	@go test -json $(go.pkgs) 2>&1 | GO111MODULE=off go run $(dir $(_go-mod.mk))gotest2tap.go | tee $@ | $(dir $(_go-mod.mk))tap-driver stream -n go-test

#
# Hook in to common.mk

build: go-build
lint: go-lint
format: go-fmt
test-suite.tap: $(if $(go.DISABLE_GO_TEST),,$(dir $(_go-mod.mk))go-test.tap)

clean: _go-clean
_go-clean:
	rm -f $(dir $(_go-mod.mk))go-test.tap vendor.hash
	rm -rf vendor/
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-02-06
	rm -f $(dir $(_go-mod.mk))patter.go $(dir $(_go-mod.mk))patter.go.tmp
.PHONY: _go-clean

clobber: _go-clobber
_go-clobber:
	rm -f $(dir $(_go-mod.mk))golangci-lint $(dir $(_go-mod.mk))go1*.src.tar.gz
.PHONY: _go-clobber

#
endif
