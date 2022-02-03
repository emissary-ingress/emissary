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

go-mod-tidy: $(patsubst $(tools.srcdir)/%/go.mod,go-mod-tidy/tools/%,$(wildcard $(tools.srcdir)/*/go.mod))

.PHONY: go-mod-tidy/tools/%
go-mod-tidy/tools/%:
	rm -f $(tools.srcdir)/$*/go.sum
	cd $(tools.srcdir)/$* && GOFLAGS=-mod=mod go mod tidy

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
tools/chart-doc-gen   = $(tools.bindir)/chart-doc-gen
tools/controller-gen  = $(tools.bindir)/controller-gen
tools/conversion-gen  = $(tools.bindir)/conversion-gen
tools/go-mkopensource = $(tools.bindir)/go-mkopensource
tools/golangci-lint   = $(tools.bindir)/golangci-lint
tools/kubestatus      = $(tools.bindir)/kubestatus
tools/protoc-gen-go   = $(tools.bindir)/protoc-gen-go
tools/yq              = $(tools.bindir)/yq
$(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(tools.srcdir)/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# Let these use the main Emissary go.mod instead of having their own go.mod.
tools.main-gomod += $(tools/protoc-gen-go)   # ensure runtime libraries are consistent
tools.main-gomod += $(tools/controller-gen)  # ensure runtime libraries are consistent
tools.main-gomod += $(tools/conversion-gen)  # ensure runtime libraries are consistent
tools.main-gomod += $(tools/go-mkopensource) # ensure it is consistent with py-mkopensource
tools.main-gomod += $(tools/kubestatus)      # is actually part of Emissary
$(tools.main-gomod): $(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(OSS_HOME)/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# Local Go sources
# ================
#
tools/dsum            = $(tools.bindir)/dsum
tools/fix-crds        = $(tools.bindir)/fix-crds
tools/flock           = $(tools.bindir)/flock
tools/gotest2tap      = $(tools.bindir)/gotest2tap
tools/goversion       = $(tools.bindir)/goversion
tools/testcert-gen    = $(tools.bindir)/testcert-gen
$(tools.bindir)/.%.stamp: $(tools.srcdir)/%/main.go FORCE
# If we build with `-mod=vendor` (which is the default if
# `vendor/modules.txt` exists), *and* our deps don't exist in $(go env
# GOMODCACHE), then the binary ends up with empty hashes for those
# packages.  In CI, (as long as `vendor/` is checked in to git) this
# means that they would be empty for the first `make generate` and
# non-empty for the second `make generate`, which would (rightfully)
# trip copy-ifchanged's "this should not happen in CI" checks.  I
# don't have the time to kill off `vendor/` yet, so for now this is
# addressed by explicitly setting `-mod=mod`.
	cd $(<D) && GOOS= GOARCH= go build -mod=mod -o $(abspath $@) .
$(tools.bindir)/%: $(tools.bindir)/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@

# Snowflakes
# ==========
#

# Telepresence would be `go get`-able, but it requires a few
# `replace`s that keeping in-sync would be more trouble than it's
# worth.
tools/telepresence   = $(tools.bindir)/telepresence
TELEPRESENCE_VERSION = 2.4.2
$(tools.bindir)/telepresence: $(tools.mk)
	mkdir -p $(@D)
	curl -s --fail -L https://app.getambassador.io/download/tel2/$(GOHOSTOS)/$(GOHOSTARCH)/$(TELEPRESENCE_VERSION)/telepresence -o $@
	chmod a+x $@

# k3d would be `go get`-able, but it requires Go 1.16, and Emissary is
# still stuck on Go 1.15.
tools/k3d   = $(tools.bindir)/k3d
K3D_VERSION = 4.4.8
$(tools.bindir)/k3d: $(tools.mk)
	mkdir -p $(@D)
	curl -s --fail -L https://github.com/rancher/k3d/releases/download/v$(K3D_VERSION)/k3d-$(GOHOSTOS)-$(GOHOSTARCH) -o $@
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

tools/kubectl = $(tools.bindir)/kubectl
KUBECTL_VERSION = 1.21.6
$(tools.bindir)/kubectl: $(tools.mk)
	mkdir -p $(@D)
	curl -o $@ -L --fail https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/$(GOHOSTOS)/$(GOHOSTARCH)/kubectl
	chmod 755 $@

endif
