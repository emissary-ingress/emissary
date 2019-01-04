<<<<<<< HEAD
# This requires Go 1.11 or newer

# Note that the `export` directive *only* affects recipes, and does
# *not* affect $(shell …).  Because of this, you shoud not call `go`
# inside of $(shell …).
export GOBIN ?= $(CURDIR)
export GO111MODULE = on
# We only set GOPATH and GOCACHE to control where the module cache
# (`$GOPATH/pkg/mod`) and the build cache are; it isn't strictly
# nescessary to set them.  We care about where the caches are because
# - we'd like to be able to copy them in to Docker images.
# - we'd like `make clean` to actually clean build caches
# - we'd like to not vomit all over the user's global caches
export GOPATH = $(CURDIR)/.gocache/workspace
export GOCACHE = $(CURDIR)/.gocache/go-build

# We could more easily set $(go.module) and $(go.bins) using Go's
# built-in module handling facilities, but that would cause the module
# system go ahead and download dependencies.  We don't want Go to do
# that at Makefile-parse-time; what if we're running `make clean`?

# See: cmd/go/internal/modfile/read.go:ModulePath()
go.module := $(strip $(shell sed -n -e 's,//.*,,' -e '/^\s*module/{s/^\s*module//;p;q}' go.mod))
#go.module := $(shell $(GO) list -m)

go.bins := $(notdir $(shell GO111MODULE=off GOCACHE=off go list -f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...))
#go.bins := $(notdir $(shell $(GO) list -f='{{if eq .Name "main"}}{{.ImportPath}}{{end}}' $(go.module)/...))

# Add the Go "main" packages to the "build" target.
build: $(addprefix $(GOBIN)/,$(go.bins))
.PHONY: build

# This is a little awkward.  We can't specify each binary as a normal
# separate target, because Make will call `go install` separately for
# each of them, and if we have multiple `go install`s going at once,
# they could corrupt $GOCACHE.  We could just say .NOTPARALLEL:, but
# we can do better.  Use a multi-target pattern rule, where the
# pattern matches $GOBIN.
#
# https://www.gnu.org/software/make/manual/html_node/Pattern-Examples.html#Pattern-Examples
$(addprefix %/,$(go.bins)): %/. FORCE
	GOBIN=$(abspath $(@D)) go install $(go.module)/...

# Make trims the leading `./` before doing pattern-matching, so if
# GOBIN=$(CURDIR), then `./teleproxy` becomes `teleproxy`, and doesn't
# match `%/teleproxy`.
$(go.bins): %: $(CURDIR)/%

clean: clean-go
clean-go:
	rm -f -- $(go.bins)
	find .gocache/workspace -exec chmod +w -- {} + || true
	rm -rf .gocache
.PHONY: clean clean-go

# Some utility targets:
go-get:
	go get -t -u -d $(go.module)/...
.PHONY: go-get

go-fmt:
	go fmt $(go.module)/...
.PHONY: go-fmt

go-mod-tidy:
	go mod tidy
.PHONY: go-mod-tidy

go-mod-vendor:
	go mod vendor
.PHONY: go-mod-vendor

# The $(go.bins) aren't .PHONY--they're real files that will exist,
# but we should try to update them every run, and let `go` decide if
# they're up-to-date or not, rather than trying to teach Make to do
# it.  So instead, have them depend on a .PHONY target so that they'll
# always be considered out-of-date.
.PHONY: FORCE
=======
# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet to build Go programs using Go 1.11 modules
#
## Inputs ##
#  - File: ./go.mod
#  - Variable: go.DISABLE_GO_TEST ?=
## Outputs ##
#  - Variable: go.module = EXAMPLE.COM/YOU/YOURREPO
#  - Variable: go.bins = List of "main" Go packages
#  - Variable: NAME ?= $(notdir $(go.module))
#  - .PHONY Target: go-get
#  - .PHONY Target: go-build
#  - .PHONY Target: check-go-fmt
#  - .PHONY Target: go-vet
#  - .PHONY Target: go-fmt
#  - .PHONY Target: go-test
## common.mk targets ##
#  - build
#  - lint
#  - check
#  - format
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
ifneq ($(go.module),)
$(error Only include one of go-mod.mk or go-workspace.mk)
endif
include $(dir $(lastword $(MAKEFILE_LIST)))/common.mk

#
# 0. configure the `go` command

export GO111MODULE = on

#
# 1. Set go.module

# Why use this complex `sed` expression to parse go.mod, instead of
# just having `go list -m` do it?  Because `go list -m` will go ahead
# and download dependencies.  We don't want Go to do that at
# Makefile-parse-time; what if we're running `make clean`?
#
# See: cmd/go/internal/modfile/read.go:ModulePath()
go.module := $(strip $(shell sed -n -e 's,//.*,,' -e '/^\s*module/{s/^\s*module//;p;q;}' go.mod))
#go.module := $(shell $(GO) list -m)
ifneq ($(words $(go.module)),1)
  # Print a helpful message
  ifeq ($(wildcard go.mod),)
    $(info go-mod.mk: File `./go.mod` does not exist.)
    ifeq ($(wildcard .go-workspace/),)
      $(info go-mod.mk: Initalize it with `go mod init github.com/YOU/REPONAME`)
    else
      $(info go-mod.mk: But `./go-workspace/` does.  Did you mean to use go-workspace.mk?)
    endif
  else
    $(info go-mod.mk: File `./go.mod` seems to be malformed; could not parse.)
  endif
  # And then error out
  $(error Could not extract $$(go.module) from ./go.mod)
endif

#
# 2. Set go.pkgs

go.pkgs := ./...

#
# 3. Recipe for go-get

go-get:
	go list ./...

#
# Include _go-common.mk

include $(dir $(lastword $(MAKEFILE_LIST)))/_go-common.mk

#
endif
>>>>>>> origin
