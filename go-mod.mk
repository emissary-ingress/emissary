# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet to build Go programs using Go 1.11 modules
#
## Eager inputs ##
#  - File: ./go.mod
#  - Variable: go.DISABLE_GO_TEST ?=
#  - Variable: go.PLATFORMS ?= $(GOOS)_$(GOARCH)
## Lazy inputs ##
#  - Variable: go.GOBUILD ?= go build
#  - Variable: go.LDFLAGS ?=
#  - Variable: go.GOLANG_LINT_VERSION ?= …
#  - Variable: go.GOLANG_LINT_FLAGS ?= …$(wildcard .golangci.yml .golangci.toml .golangci.json)…
#  - Variable: CI
## Outputs ##
#  - Variable: NAME ?= $(notdir $(go.module))
#  - Variable: go.module = EXAMPLE.COM/YOU/YOURREPO
#  - Variable: go.bins = List of "main" Go packages
#  - Variable: go.pkgs = ./...
#  - Function: go.list = $(shell go list $1), but ignores submodules and doesn't download things
#  - Targets: bin_$(OS)_$(ARCH)/$(CMD)
#  - .PHONY Target: go-get
#  - .PHONY Target: go-build
#  - .PHONY Target: go-lint
#  - .PHONY Target: go-fmt
#  - .PHONY Target: go-test
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
_go-common.mk := $(lastword $(MAKEFILE_LIST))
ifneq ($(go.module),)
$(error Only include one of go-mod.mk or go-workspace.mk)
endif

#
# 0. configure the `go` command

export GO111MODULE = on

# Disable parallel builds on Go 1.11; the module cache is not
# concurrency-safe.  This is fixed in 1.12.
ifneq ($(filter go1.11.%,$(shell go version)),)
.NOTPARALLEL:
endif

#
# 1. Set go.module

go.module := $(shell GO111MODULE=on go mod edit -json | jq -r .Module.Path)
ifneq ($(words $(go.module)),1)
  $(error Could not extract $$(go.module) from ./go.mod)
endif

#
# 2. Set go.pkgs

go.pkgs := ./...

#
# 3. Recipe for go-get

go-get:
	go mod download

#
# Include _go-common.mk

include $(dir $(_go-common.mk))common.mk

NAME ?= $(notdir $(go.module))

go.GOBUILD ?= go build
go.DISABLE_GO_TEST ?=
go.LDFLAGS ?=
go.PLATFORMS ?= $(GOOS)_$(GOARCH)
go.GOLANG_LINT_VERSION ?= 1.15.0
go.GOLANG_LINT_FLAGS ?= $(if $(wildcard .golangci.yml .golangci.toml .golangci.json),,--disable-all --enable=gofmt --enable=govet)

# It would be simpler to create this list if we could use Go modules:
#
#     go.bins := $(shell $(GO) list -f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)
#
# But alas, we can't do that *even if* the build is using Go modules,
# as that would cause the module system go ahead and download
# dependencies.  We don't want Go to do that at Makefile-parse-time;
# what if we're running `make clean`?
#
# So instead, we must deal with this abomination.  At least that means
# we can share it between go-mod.mk and go-workspace.mk.
_go.submods := $(patsubst %/go.mod,%,$(shell git ls-files '*/go.mod'))
go.list = $(call path.addprefix,$(go.module),\
                                $(filter-out $(foreach d,$(_go.submods),$d $d/%),\
                                             $(call path.trimprefix,_$(CURDIR),\
                                                                    $(shell GOPATH=/bogus GO111MODULE=off go list $1))))
go.bins := $(call go.list,-f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)

#
# (_go-common.mk) Rules

# go-FOO.mk is responsible for implementing go-get
go-get: ## (Go) Download Go dependencies
.PHONY: go-get

define _go.bin.rule
bin_%/.cache.$(notdir $(go.bin)): go-get FORCE
	$$(go.GOBUILD) $$(if $$(go.LDFLAGS),--ldflags $$(call quote.shell,$$(go.LDFLAGS))) -o $$@ $(go.bin)
bin_%/$(notdir $(go.bin)): bin_%/.cache.$(notdir $(go.bin))
	@{ \
		PS4=''; set -x; \
		if ! cmp -s $$< $$@; then \
			$(if $(CI),if test -e $$@; then false This should not happen in CI: $$@ should not change; fi &&) \
			cp -f $$< $$@; \
		fi; \
	}
endef
$(foreach go.bin,$(go.bins),$(eval $(_go.bin.rule)))

go-build: $(foreach _go.PLATFORM,$(go.PLATFORMS),$(addprefix bin_$(_go.PLATFORM)/,$(notdir $(go.bins))))
.PHONY: go-build

$(dir $(_go-common.mk))golangci-lint: $(_go-common.mk)
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(@D) v$(go.GOLANG_LINT_VERSION)

go-lint: ## (Go) Check the code with `golangci-lint`
go-lint: $(dir $(_go-common.mk))golangci-lint go-get
	$(dir $(_go-common.mk))golangci-lint run $(go.GOLANG_LINT_FLAGS) $(go.pkgs)
.PHONY: go-lint

go-fmt: ## (Go) Fixup the code with `go fmt`
go-fmt: go-get
	go fmt $(go.pkgs)
.PHONY: go-fmt

go-test: ## (Go) Check the code with `go test`
go-test: go-build
ifeq ($(go.DISABLE_GO_TEST),)
	$(MAKE) $(dir $(_go-common.mk))go-test.tap.summary
endif

$(dir $(_go-common.mk))go-test.tap: FORCE
	@go test -json $(go.pkgs) 2>&1 | GO111MODULE=off go run $(dir $(_go-common.mk))gotest2tap.go | tee $@ | $(dir $(_go-common.mk))tap-driver stream -n go-test

#
# (_go-common.mk) Hook in to common.mk

build: go-build
lint: go-lint
format: go-fmt
test-suite.tap: $(if $(go.DISABLE_GO_TEST),,$(dir $(_go-common.mk))go-test.tap)

clean: _clean-go-common
_clean-go-common:
	rm -f $(dir $(_go-common.mk))go-test.tap
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-02-06
	rm -f $(dir $(_go-common.mk))patter.go $(dir $(_go-common.mk))patter.go.tmp
.PHONY: _clean-go-common

clobber: _clobber-go-common
_clobber-go-common:
	rm -f $(dir $(_go-common.mk))golangci-lint
.PHONY: _clobber-go-common

#
endif
