# This file deals with installing programs used by the build.
#
# It depends on:
#  - The `go` binary being installed in PATH.
#  - OSS_HOME being set.
# That should be it.

ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)

tools.mk    := $(lastword $(MAKEFILE_LIST))
tools.dir    = tools
tools.bindir = $(tools.dir)/bin
tools.srcdir = $(tools.dir)/src

GOHOSTOS   := $(shell go env GOHOSTOS)
GOHOSTARCH := $(shell go env GOHOSTARCH)

clobber: clobber-tools

.PHONY: clobber-tools
clobber-tools:
	rm -rf $(tools.bindir) $(tools.dir)/include $(tools.dir)/downloads

# Shell scripts
# =============
#
tools/copy-ifchanged      = $(tools.bindir)/copy-ifchanged
$(tools.bindir)/%: build-aux/bin-sh/%.sh
	mkdir -p $(@D)
	install $< $@

# `go get`-able things
# ====================
#
tools/golangci-lint = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/golangci-lint
$(tools/golangci-lint): $(OSS_HOME)/build-aux/bin-go/golangci-lint/go.mod
	mkdir -p $(@D)
	cd $(<D) && go build -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

# Local Go sources
# ================
#
tools/schema-fmt      = $(tools.bindir)/schema-fmt
$(tools.bindir)/.%.stamp: $(tools.srcdir)/%/main.go FORCE
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) .
$(tools.bindir)/%: $(tools.bindir)/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@

endif
