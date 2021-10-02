# This file deals with installing programs used by the build.
#
# It depends on:
#  - The `go` binary being installed in PATH.
#  - OSS_HOME being set.
# That should be it.

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
tools/docker-promote      = $(tools.bindir)/docker-promote
tools/move-ifchanged      = $(tools.bindir)/move-ifchanged
tools/tap-driver          = $(tools.bindir)/tap-driver
tools/write-dockertagfile = $(tools.bindir)/write-dockertagfile
tools/write-ifchanged     = $(tools.bindir)/write-ifchanged
$(tools.bindir)/%: $(tools.srcdir)/%.sh
	mkdir -p $(@D)
	install $< $@

# `go get`-able things
# ====================
#
tools/chart-doc-gen  = $(tools.bindir)/chart-doc-gen
tools/controller-gen = $(tools.bindir)/controller-gen
tools/go-bindata     = $(tools.bindir)/go-bindata
tools/golangci-lint  = $(tools.bindir)/golangci-lint
tools/kubestatus     = $(tools.bindir)/kubestatus
tools/protoc-gen-go  = $(tools.bindir)/protoc-gen-go
tools/yq             = $(tools.bindir)/yq
$(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(tools.srcdir)/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# Let these use the main Emissary go.mod instead of having their own go.mod.
tools.main-gomod += $(tools/protoc-gen-go)  # ensure runtime libraries are consistent
tools.main-gomod += $(tools/controller-gen) # ensure runtime libraries are consistent
tools.main-gomod += $(tools/kubestatus)     # is actually part of Emissary
$(tools.main-gomod): $(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(OSS_HOME)/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# Local Go sources
# ================
#
tools/crds2schemas    = $(tools.bindir)/crds2schemas
tools/dsum            = $(tools.bindir)/dsum
tools/fix-crds        = $(tools.bindir)/fix-crds
tools/flock           = $(tools.bindir)/flock
tools/go-mkopensource = $(tools.bindir)/go-mkopensource
tools/gotest2tap      = $(tools.bindir)/gotest2tap
tools/py-mkopensource = $(tools.bindir)/py-mkopensource
tools/schema-fmt      = $(tools.bindir)/schema-fmt
$(tools.bindir)/.%.stamp: $(tools.srcdir)/%/main.go FORCE
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) .
$(tools.bindir)/%: $(tools.bindir)/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@

# Snowflakes
# ==========
#

tools/telepresence   = $(tools.bindir)/telepresence
TELEPRESENCE_VERSION = 2.4.2
$(tools.bindir)/telepresence: $(tools.mk)
	mkdir -p $(@D)
	curl -s --fail -L https://app.getambassador.io/download/tel2/$(GOHOSTOS)/$(GOHOSTARCH)/$(TELEPRESENCE_VERSION)/telepresence -o $@
	chmod a+x $@

# PROTOC_VERSION must be at least 3.8.0 in order to contain the fix so that it doesn't generate
# invalid Python if you name an Enum member the same as a Python keyword.
tools/protoc    = $(tools.bindir)/protoc
PROTOC_VERSION  = 3.8.0
PROTOC_ZIP     := protoc-$(PROTOC_VERSION)-$(patsubst darwin,osx,$(GOHOSTOS))-$(shell uname -m).zip
$(tools.dir)/downloads/$(PROTOC_ZIP):
	mkdir -p $(@D)
	curl --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP) -o $@
$(tools.dir)/bin/protoc $(tools.dir)/include: $(tools.dir)/%: $(tools.dir)/downloads/$(PROTOC_ZIP) $(tools.mk)
	bsdtar -C $(tools.dir) -x -m -f $< $*
$(tools.dir)/bin/protoc: $(tools.dir)/include

tools/protoc-gen-grpc-web  = $(tools.bindir)/protoc-gen-grpc-web
GRPC_WEB_VERSION           = 1.0.3
GRPC_WEB_PLATFORM         := $(GOHOSTOS)-$(shell uname -m)
$(tools.bindir)/protoc-gen-grpc-web: $(tools.mk)
	mkdir -p $(@D)
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@
