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

include $(dir $(lastword $(MAKEFILE_LIST)))_go-common.mk

#
endif
