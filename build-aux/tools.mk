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
go-mod-tidy/tools/%: $(OSS_HOME)/build-aux/go-version.txt
	rm -f $(tools.srcdir)/$*/go.sum
	cd $(tools.srcdir)/$* && GOFLAGS=-mod=mod go mod tidy -compat=$$(cut -d. -f1,2 < $<) -go=$$(cut -d. -f1,2 < $<)

# Shell scripts
# =============
#
tools/copy-ifchanged      = $(tools.bindir)/copy-ifchanged
tools/devversion          = $(tools.bindir)/devversion
tools/docker-promote      = $(tools.bindir)/docker-promote
tools/move-ifchanged      = $(tools.bindir)/move-ifchanged
tools/tap-driver          = $(tools.bindir)/tap-driver
tools/write-dockertagfile = $(tools.bindir)/write-dockertagfile
tools/write-ifchanged     = $(tools.bindir)/write-ifchanged
$(tools.bindir)/%: $(tools.srcdir)/%.sh
	mkdir -p $(@D)
	install $< $@

# Python scripts
# ==============
#
tools/py-list-deps = $(tools.bindir)/py-list-deps
$(tools.bindir)/%: $(tools.srcdir)/%.py
	mkdir -p $(@D)
	install $< $@

# `go get`-able things
# ====================
#
tools/chart-doc-gen      = $(tools.bindir)/chart-doc-gen
tools/controller-gen     = $(tools.bindir)/controller-gen
tools/conversion-gen     = $(tools.bindir)/conversion-gen
tools/crane              = $(tools.bindir)/crane
tools/go-mkopensource    = $(tools.bindir)/go-mkopensource
tools/golangci-lint      = $(tools.bindir)/golangci-lint
tools/kubestatus         = $(tools.bindir)/kubestatus
tools/ocibuild           = $(tools.bindir)/ocibuild
tools/protoc-gen-go      = $(tools.bindir)/protoc-gen-go
tools/protoc-gen-go-grpc = $(tools.bindir)/protoc-gen-go-grpc
tools/yq                 = $(tools.bindir)/yq
$(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(tools.srcdir)/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
# Let these use the main Emissary go.mod instead of having their own go.mod.
tools.main-gomod += $(tools/controller-gen)     # ensure runtime libraries are consistent
tools.main-gomod += $(tools/conversion-gen)     # ensure runtime libraries are consistent
tools.main-gomod += $(tools/protoc-gen-go)      # ensure runtime libraries are consistent
tools.main-gomod += $(tools/protoc-gen-go-grpc) # ensure runtime libraries are consistent
tools.main-gomod += $(tools/go-mkopensource)    # ensure it is consistent with py-mkopensource
tools.main-gomod += $(tools/kubestatus)         # is actually part of Emissary
$(tools.main-gomod): $(tools.bindir)/%: $(tools.srcdir)/%/pin.go $(OSS_HOME)/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# Local Go sources
# ================
#
tools/dsum            = $(tools.bindir)/dsum
tools/filter-yaml     = $(tools.bindir)/filter-yaml
tools/fix-crds        = $(tools.bindir)/fix-crds
tools/flock           = $(tools.bindir)/flock
tools/gotest2tap      = $(tools.bindir)/gotest2tap
tools/goversion       = $(tools.bindir)/goversion
tools/py-mkopensource = $(tools.bindir)/py-mkopensource
tools/py-split-tests  = $(tools.bindir)/py-split-tests
tools/testcert-gen    = $(tools.bindir)/testcert-gen
$(tools.bindir)/.%.stamp: $(tools.srcdir)/%/main.go FORCE
# If we build with `-mod=vendor` (which is the default if
# `vendor/modules.txt` exists), *and* our deps don't exist in $(go env
# GOMODCACHE), then the binary ends up with empty hashes for those
# packages; this can (rightfully) trip copy-ifchanged's "this should
# not happen in CI" checks.
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
TELEPRESENCE_VERSION = 2.6.6
$(tools.bindir)/telepresence: $(tools.mk)
	mkdir -p $(@D)
	curl -s --fail -L https://app.getambassador.io/download/tel2/$(GOHOSTOS)/$(GOHOSTARCH)/$(TELEPRESENCE_VERSION)/telepresence -o $@
	chmod a+x $@

# k3d is in theory `go get`-able, but... the tests fail when I do
# that.  IDK.  --lukeshu
tools/k3d   = $(tools.bindir)/k3d
K3D_VERSION = 5.4.7
$(tools.bindir)/k3d: $(tools.mk)
	mkdir -p $(@D)
	curl -s --fail -L https://github.com/rancher/k3d/releases/download/v$(K3D_VERSION)/k3d-$(GOHOSTOS)-$(GOHOSTARCH) -o $@
	chmod a+x $@

# PROTOC_VERSION must be at least 3.8.0 in order to contain the fix so that it doesn't generate
# invalid Python if you name an Enum member the same as a Python keyword.
tools/protoc    = $(tools.bindir)/protoc
PROTOC_VERSION  = 21.5
PROTOC_ZIP     := protoc-$(PROTOC_VERSION)-$(patsubst darwin,osx,$(GOHOSTOS))-$(patsubst arm64,aarch_64,$(shell uname -m)).zip
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

tools/ct = $(tools.bindir)/ct
$(tools/ct): $(tools.bindir)/%: $(tools.srcdir)/%/wrap.sh $(tools/ct).d/bin/ct $(tools/ct).d/bin/kubectl $(tools/ct).d/venv $(tools/ct).d/home
	install $< $@
$(tools/ct).d/bin/ct: $(tools.srcdir)/ct/pin.go $(tools.srcdir)/ct/go.mod
	@PS4=; set -ex; {\
	  cd $(<D); \
	  pkg=$$(sed -En 's,^import "(.*)".*,\1,p' pin.go); \
	  ver=$$(go list -f='{{ .Module.Version }}' "$$pkg"); \
	  GOOS= GOARCH= go build -o $(abspath $@) -ldflags="-X $$pkg/cmd.Version=$$ver" "$$pkg"; \
	}
$(tools/ct).d/bin/kubectl: $(tools/kubectl)
	mkdir -p $(@D)
	ln -s ../../kubectl $@
$(tools/ct).d/dir.txt: $(tools.srcdir)/ct/pin.go $(tools.srcdir)/ct/go.mod
	mkdir -p $(@D)
	cd $(<D) && GOFLAGS='-mod=readonly' go list -f='{{ .Module.Dir }}' "$$(sed -En 's,^import "(.*)".*,\1,p' pin.go)" >$(abspath $@)
$(tools/ct).d/venv: %/venv: %/dir.txt
	rm -rf $@
	python3 -m venv $@
	$@/bin/pip3 install \
	  yamllint==$$(sed -n 's/ARG yamllint_version=//p' "$$(cat $<)/Dockerfile") \
	  yamale==$$(sed -n 's/ARG yamale_version=//p' "$$(cat $<)/Dockerfile") \
	  || (rm -rf $@; exit 1)
$(tools/ct).d/home: %/home: %/dir.txt
	rm -rf $@
	mkdir $@ $@/.ct
	cp "$$(cat $<)"/etc/* $@/.ct || (rm -rf $@; exit 1)

# Inter-tool dependencies
# =======================
#
$(tools/devversion): $(tools/goversion)

endif
