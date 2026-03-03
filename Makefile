# Shut up make
MAKEFLAGS += --no-print-directory

# Real early setup
OSS_HOME := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))# Do this *before* 'include'-ing anything else
include build-aux/init-sanitize-env.mk
include build-aux/init-configure-make-itself.mk
include build-aux/prelude.mk # In Haskell, "Prelude" is what they call the stdlib builtins that get get imported by default before anything else
include build-aux/tools.mk

# TEST_CLUSTER overrides the name of the k3d test cluster. It defaults
# to "emissary-test".
TEST_CLUSTER ?= emissary-test

# Default Envoy image to use when building Emissary.
ENVOY_IMAGE ?= envoyproxy/envoy:distroless-v1.36.2

# Bootstrapping the build env
#
# I actually kind of hate this code. What's happening is that mostly we
# require VERSION to be set, but we have a fallback mechanism to
# calculate it if it's not set.
ifneq ($(MAKECMDGOALS),$(OSS_HOME)/build-aux/go-version.txt)
ifneq ($(filter tools/bin/goversion,$(MAKECMDGOALS)),tools/bin/goversion)
  ifneq ($(filter $(shell go env GOROOT),$(subst :, ,$(shell go env GOPATH))),)
    $(error Your $$GOPATH (where *your* Go stuff goes) and $$GOROOT (where Go *itself* is installed) are both set to the same directory ($(shell go env GOROOT)); it is remarkable that it has not blown up catastrophically before now)
  endif
  ifneq ($(foreach gopath,$(subst :, ,$(shell go env GOPATH)),$(filter $(gopath)/%,$(CURDIR))),)
    $(error Your emissary.git checkout is inside of your $$GOPATH ($(shell go env GOPATH)); Emissary-ingress uses Go modules and so GOPATH need not be pointed at it (in a post-modules world, the only role of GOPATH is to store the module download cache); and indeed some of the Kubernetes tools will get confused if GOPATH is pointed at it)
  endif

  # Ensure goversion is built before we try to use it
  print-version:
	@echo $(VERSION)

  _goversion_check := $(shell test -x $(OSS_HOME)/tools/bin/goversion || $(MAKE) -C $(OSS_HOME) tools/bin/goversion >&2)
  VERSION := $(or $(VERSION),$(shell $(OSS_HOME)/tools/bin/goversion))
  $(if $(or $(filter v4.%,$(VERSION)),$(filter v0.40.%,$(VERSION)),$(filter v0.0.0-%,$(VERSION))),\
  ,\ $(error VERSION variable is invalid: It must be v4.*, v0.40.* or v0.0.0-$$tag, but is '$(VERSION)'))
  $(if $(findstring +,$(VERSION)),\
    $(error VERSION variable is invalid: It must not contain + characters, but is '$(VERSION)'),)    
  export VERSION

  ARCH := $(or $(ARCH),$(shell uname -m))
  ifeq ($(ARCH),x86_64)
	ARCH := amd64
  endif
  ifeq ($(ARCH),aarch64)
	ARCH := arm64
  endif
  export ARCH

  # By default, we'll build for the same processor architecture as this Makefile
  # is running on, but you can change this. Unsetting it will build for all amd64
  # and arm64.
  BUILD_ARCH ?= linux/$(ARCH)

  # This is a bit hackish, at the moment, but let's run with it for now
  # and see how far we get.
  CHART_VERSION := $(VERSION)
  export CHART_VERSION

#   CHART_VERSION := $(or $(CHART_VERSION),$(shell go run ./tools/src/goversion --dir-prefix=chart))
#   $(if $(or $(filter v4.%,$(CHART_VERSION)),$(filter v0.0.0-%,$(CHART_VERSION))),\
#     ,$(error CHART_VERSION variable is invalid: It must be v4.* or v0.0.0-$$tag, but is '$(CHART_VERSION)'))
#   export CHART_VERSION

  ifeq ($(shell test "$(VERBOSE)" -gt 0 2>/dev/null && echo true),true)
    $(info [make] VERSION=$(VERSION))
    $(info [make] CHART_VERSION=$(CHART_VERSION))
    $(info [make] ARCH=$(ARCH))
  endif
endif
endif

# If SOURCE_DATE_EPOCH isn't set, AND the tree isn't dirty, then set
# SOURCE_DATE_EPOCH to the commit timestamp.
#
# if [[ -z "$SOURCE_DATE_EPOCH" ]] && [[ -z "$(git status --porcelain)" ]]; then
ifeq ($(SOURCE_DATE_EPOCH)$(shell git status --porcelain),)
  SOURCE_DATE_EPOCH := $(shell git log -1 --pretty=%ct)
endif
ifneq ($(SOURCE_DATE_EPOCH),)
  ifeq ($(shell test "$(VERBOSE)" -gt 0 2>/dev/null && echo true),true)
    export SOURCE_DATE_EPOCH
    $(info [make] SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH))
  endif
endif

# Everything else...

EMISSARY_NAME ?= emissary

_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring 'private',$(_git_remote_urls))

include $(OSS_HOME)/build-aux/charts.mk
include $(OSS_HOME)/build-aux/ci.mk
include $(OSS_HOME)/build-aux/deps.mk
include $(OSS_HOME)/build-aux/main.mk
include $(OSS_HOME)/build-aux/builder.mk
include $(OSS_HOME)/build-aux/check.mk
include $(OSS_HOME)/releng/release.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux/generate.mk
include $(OSS_HOME)/build-aux/lint.mk

.PHONY: print-envoy-image
print-envoy-image:
	@echo $(ENVOY_IMAGE)

.PHONY: info
info:
	@echo "VERSION=$(VERSION)"
	@echo "CHART_VERSION=$(CHART_VERSION)"
	@echo "ENVOY_IMAGE=$(ENVOY_IMAGE)"
	@echo "ARCH=$(ARCH)"
	@echo "BUILD_ARCH=$(BUILD_ARCH)"

.git/hooks/prepare-commit-msg:
	ln -s $(OSS_HOME)/tools/hooks/prepare-commit-msg $(OSS_HOME)/.git/hooks/prepare-commit-msg

githooks: .git/hooks/prepare-commit-msg

## Helper target for setting up local dev environment when working with python components
## such as pytest, diagd, etc...
.PHONY: python-dev-setup
python-dev-setup:
# recreate venv and upgrade pip
	rm -rf venv
	python3 -m venv venv
	venv/bin/python3 -m pip install --upgrade pip

# install deps, dev deps and diagd
	./venv/bin/pip install -r python/requirements.txt
	./venv/bin/pip install -r python/requirements-dev.txt
	./venv/bin/pip install -e python

# activate venv
	@echo "run 'source ./venv/bin/activate' to activate venv in local shell"

# re-generate docs
.PHONY: clean-changelog
clean-changelog:
	rm -f CHANGELOG.md

.PHONY: generate-changelog
generate-changelog: clean-changelog $(PWD)/CHANGELOG.md


.PHONY: list-target-names
list-target-names:
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'
