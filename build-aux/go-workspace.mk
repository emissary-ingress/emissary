# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet to build Go programs using Go workspaces
# ("GOPATH").  Go workspaces are scheduled for deprecation in Go 1.13,
# which is scheduled for August 2019.
#
## Inputs ##
#  - Symlink: ./.go-workspace/src/EXAMPLE.COM/YOU/YOURREPO -> (git topdir)
## Outputs ##
#  - Variable: go.module = EXAMPLE.COM/YOU/YOURREPO
#  - Variable: go.bins = List of "main" Go packages
#  - Target: vendor/ (if `./glade.yaml` is present)
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
#  - clobber

go.module := $(patsubst src/%,%,$(shell cd .go-workspace && find src \( -name '.*' -prune \) -o -type l -print))
ifneq ($(words $(go.module)),1)
  # Print a helpful message
  ifeq ($(wildcard .go-workspace/),)
    $(info go-worksapce.mk: Directory `./go-workspace/` does not exist.)
    ifeq ($(wildcard go.mod),)
      $(info go-workspace.mk: Initialize it with:)
      $(info go-workspace.mk:     $$ mkdir -p .go-workspace/src/github.com/YOU)
      $(info go-workspace.mk:     $$ ln -srT . .go-workspace/src/github.com/YOU/REPONAME)
    else
      $(info go-workspace.mk: But `./go.mod` does.  Did you mean to use go-mod.mk?)
    endif
  else
    $(info go-workspace.mk: Did not find exactly 1 symlink under `./go-workspace/`)
  endif
  # And then error out
  $(error Could not extract $$(go.module) from ./.go-workspace/src/)
endif

export GO111MODULE = off
export GOPATH = $(CURDIR)/.go-workspace

go-get:
	go get -d $(go.bins)

ifneq ($(wildcard glide.yaml),)
vendor: glide.yaml $(wildcard glide.lock)
	rm -rf $@
	glide install || { r=$?; rm -rf $@; exit $?; }
go-get: vendor

_go-clobber-vendor:
	rm -rf vendor
.PHONY: _go-clobber-vendor
clobber: _go-clobber-vendor
endif

_go-clobber:
	find .go-workspace -exec chmod +w {} +
	rm -rf .go-workspace
	mkdir -p $(dir .go-workspace/src/$(go.module))
	ln -s $(call joinlist,$(patsubst %,..,$(subst /, ,$(dir .go-workspace/src/$(go.module)))),/) .go-workspace/src/$(go.module)
.PHONY: _go-clobber
clobber: _go-clobber

# .NOTPARALLEL is important, as having multiple `go install`s going at
# once can corrupt `$(GOPATH)/pkg`.  Setting .NOTPARALLEL is simpler
# than mucking with multi-target pattern rules.
.NOTPARALLEL:

include $(dir $(lastword $(MAKEFILE_LIST)))/_go-common.mk
