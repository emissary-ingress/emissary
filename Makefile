# file: Makefile

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

# Welcome to the Ambassador Makefile...

# We'll set REGISTRY_ERR in builder.mk
DEV_REGISTRY ?= $(REGISTRY_ERR)

# IS_PRIVATE: empty=false, nonempty=true
# Default is true if any of the git remotes have the string "private" in any of their URLs.
_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

RELEASE_DOCKER_REPO ?= quay.io/datawire/ambassador$(if $(IS_PRIVATE),-private)
BASE_DOCKER_REPO    ?= quay.io/datawire/ambassador-base$(if $(IS_PRIVATE),-private)
DEV_DOCKER_REPO     ?= $(DEV_REGISTRY)/dev

DOCKER_OPTS ?=

YES_I_AM_UPDATING_THE_BASE_IMAGES ?=

docker.tag.dev        = $(DEV_DOCKER_REPO):$(notdir $*)-$(shell tr : - < $<)
# By default, don't allow .release, .release-rc, .release-ea, or .base tags...
docker.tag.release    = $(error The 'release' tag is only valid for the 'ambassador-release{,-rc,-ea}' images)
docker.tag.base       = $(error The 'base' tag is only valid for the 'base-envoy' image)
# ... except for on specific images
ambassador-release.docker.tag.release:    docker.tag.release = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)
ambassador-release-rc.docker.tag.release: docker.tag.release = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION) $(RELEASE_DOCKER_REPO):$(BUILD_VERSION)-rc-latest
ambassador-release-ea.docker.tag.release: docker.tag.release = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)
BASE_IMAGE.envoy = $(BASE_DOCKER_REPO):envoy-$(BASE_VERSION.envoy)
base-envoy.docker.tag.base:               docker.tag.base       = $(BASE_IMAGE.envoy)

# We'll set REGISTRY_ERR in builder.mk
docker.tag.dev = $(if $(DEV_REGISTRY),$(DEV_REGISTRY)/$*:$(patsubst sha256:%,%,$(shell cat $<)),$(REGISTRY_ERR))

# All Docker images that we know how to build
images.all =
# The subset of $(images.all) that we will deploy to the
# DEV_KUBECONFIG cluster.
images.cluster =
# The subset of $(images.all) that `make update-base` should update.
images.base =

images.all += $(patsubst docker/%/Dockerfile,%,$(wildcard docker/*/Dockerfile)) test-auth-tls
images.cluster += $(filter test-%,$(images.all))
images.base += $(filter base-%,$(images.all))

OSS_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux/var.mk
include $(OSS_HOME)/build-aux/docker.mk
include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/cxx/envoy.mk
include $(OSS_HOME)/build-aux-local/kat.mk
include $(OSS_HOME)/build-aux-local/docs.mk
include $(OSS_HOME)/build-aux-local/release.mk
include $(OSS_HOME)/build-aux-local/version.mk
.DEFAULT_GOAL = help

$(call module,ambassador,$(OSS_HOME))

sync: python/ambassador/VERSION.py

clean: _makefile_clean
clobber: _makefile_clobber
_makefile_clean:
	rm -f python/ambassador/VERSION.py
_makefile_clobber:
	rm -rf bin_*/
.PHONY: _makefile_clean _makefile_clobber

generate: ## Update generated sources that get committed to git
generate: pkg/api/kat/echo.pb.go
generate-clean: ## Delete generated sources that get committed to git (implies `make clobber`)
generate-clean: clobber
	rm -rf pkg/api
.PHONY: generate generate-clean

base-%.docker.stamp: docker/base-%/Dockerfile $(var.)BASE_IMAGE.%
	@PS4=; set -ex; { \
	    if ! docker run --rm --entrypoint=true $(BASE_IMAGE.$*); then \
	        if [ -z '$(YES_I_AM_UPDATING_THE_BASE_IMAGES)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: failed to pull $(BASE_IMAGE.$*), but $$YES_I_AM_UPDATING_THE_BASE_IMAGES is not set'; \
	            echo '       If you are trying to update the base images, then set that variable to a non-empty value.'; \
	            echo '       If you are not trying to update the base images, then check your network connection and Docker credentials.'; \
	            exit 1; \
	        fi; \
	        docker build $(DOCKER_OPTS) $($@.DOCKER_OPTS) -t $(BASE_IMAGE.$*) -f $< $(or $($@.DOCKER_DIR),.); \
	    fi; \
	}
	docker image inspect $(BASE_IMAGE.$*) --format='{{.Id}}' > $@

test-%.docker.stamp: docker/test-%/Dockerfile FORCE
	docker build --quiet --iidfile=$@ $(<D)
test-auth-tls.docker.stamp: docker/test-auth/Dockerfile FORCE
	docker build --quiet --build-arg TLS=--tls --iidfile=$@ $(<D)

update-base: ## Run this whenever the base images (ex Envoy, ./docker/base-*/*) change
	$(MAKE) $(addsuffix .docker.tag.base,$(images.base))
	$(MAKE) generate
	$(MAKE) $(addsuffix .docker.push.base,$(images.base))
.PHONY: update-base

export-vars:
	@echo "export BASE_DOCKER_REPO='$(BASE_DOCKER_REPO)'"
	@echo "export RELEASE_DOCKER_REPO='$(RELEASE_DOCKER_REPO)'"
.PHONY: export-vars

# Configure GNU Make itself
SHELL = bash
.SECONDARY:
.DELETE_ON_ERROR:
