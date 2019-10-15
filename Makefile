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

SHELL = bash

# IS_PRIVATE: empty=false, nonempty=true
# Default is true if any of the git remotes have the string "private" in any of their URLs.
_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

RELEASE_DOCKER_REPO ?= quay.io/datawire/ambassador$(if $(IS_PRIVATE),-private)
BASE_DOCKER_REPO    ?= quay.io/datawire/ambassador-base$(if $(IS_PRIVATE),-private)
DEV_DOCKER_REPO     ?=

ifeq ($(DEV_DOCKER_REPO),)
  $(error DEV_DOCKER_REPO must be set.  Use a nonsense value for a purely local build.)
endif

DOCKER_OPTS ?=

YES_I_AM_UPDATING_THE_BASE_IMAGES ?=

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
  # Increment BASE_RUNTIME_RELVER on changes to
  # `docker/base-runtime/Dockerfile`
  BASE_RUNTIME_RELVER ?= 1
  # Increment BASE_PY_RELVER on changes to `docker/base-py/Dockerfile`
  # or `python/requirements.txt`.  You may reset it to '1' whenever
  # you edit BASE_RUNTIME_RELVER.
  BASE_PY_RELVER      ?= 1

  BASE_VERSION.runtime ?= $(BASE_RUNTIME_RELVER)
  BASE_VERSION.py      ?= $(BASE_RUNTIME_RELVER).$(BASE_PY_RELVER)
# END LIST OF VARIABLES REQUIRING `make update-base`.

#
#

# Set default tag values...
docker.tag.release    = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)
docker.tag.release-rc = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION) $(RELEASE_REPO):$(BUILD_VERSION)-latest-rc
docker.tag.release-ea = $(RELEASE_DOCKER_REPO):$(RELEASE_VERSION)
docker.tag.dev        = $(DEV_DOCKER_REPO):$(notdir $*)-$(shell tr : - < $<)
BASE_IMAGE._          = $(BASE_DOCKER_REPO):$1-$(BASE_VERSION.$1)
BASE_IMAGE.envoy      = $(call BASE_IMAGE._,envoy)
BASE_IMAGE.runtime    = $(call BASE_IMAGE._,runtime)
BASE_IMAGE.py         = $(call BASE_IMAGE._,py)
docker.tag.base       = $(BASE_IMAGE.$(patsubst base-%.docker,%,$<))
# Tag groups used by older versions.  Remove the tail of this list
# when the commit making the change gets far enough in to the past.
#
# 2019-10-14
docker.tag.build-sys  = $(error The '.build-sys' Docker tag-goup is no longer used)
docker.tag.local      = $(error The '.local' Docker tag-goup is no longer used)

# All images we know how to build
images.all = $(patsubst docker/%/Dockerfile,%,$(wildcard docker/*/Dockerfile)) test-auth-tls ambassador
# Images that will end up inside of a cluster during `make test`
images.cluster = $(filter-out base-%,$(images.all))
# Base images that we cache more aggressively
images.base = $(filter base-%,$(images.all))
# Images made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-10-13
images.old += base-go

KUBECTL_VERSION = 1.16.1

go.bins.extra += github.com/datawire/teleproxy/cmd/kubestatus
go.bins.extra += github.com/datawire/teleproxy/cmd/teleproxy
go.bins.extra += github.com/datawire/teleproxy/cmd/watt
export CGO_ENABLED = 0

include build-aux/prelude.mk
include build-aux/var.mk
include build-aux/docker.mk
include build-aux/common.mk
include build-aux/go-mod.mk
include build-aux/help.mk
include cxx/envoy.mk
include build-aux-local/kat.mk
include build-aux-local/docs.mk
include build-aux-local/release.mk
include build-aux-local/version.mk
.DEFAULT_GOAL = help

clean: $(addsuffix .docker.clean,$(images.all) $(images.old))
	rm -rf docs/_book docs/_site docs/package-lock.json
	rm -rf helm/*.tgz
	rm -rf app.json
	rm -rf venv/bin/ambassador
	rm -rf python/ambassador/VERSION.py*
	rm -f *.docker
	rm -rf python/build python/dist python/ambassador.egg-info python/__pycache__
	find . \( -name .coverage -o -name .cache -o -name __pycache__ \) -print0 | xargs -0 rm -rf
	find . \( -name *.log \) -print0 | xargs -0 rm -rf
	rm -rf log.txt
	find python/tests \
		\( -name '*.out' -o -name 'envoy.json' -o -name 'intermediate.json' \) -print0 \
		| xargs -0 rm -f
	rm -f docker/kat-client/kat_client
	rm -f docker/kat-client/teleproxy
	rm -f docker/kat-server/kat-server
	rm -f tools/sandbox/http_auth/docker-compose.yml
	rm -f tools/sandbox/grpc_auth/docker-compose.yml
	rm -f tools/sandbox/grpc_web/docker-compose.yaml tools/sandbox/grpc_web/*_pb.js
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-10-13
	rm -f build/kat/client/kat_client
	rm -f build/kat/client/teleproxy
	rm -f build/kat/server/kat-server
# 2019-10-13
	if [ -r .docker_port_forward ]; then kill $$(cat .docker_port_forward) || true; fi
	rm -f .docker_port_forward
# 2019-10-13
	rm -f cluster.yaml kubernaut-claim.txt
# 2019-10-13
	rm -f ambex kubestatus watt
	rm -f cmd/ambex/ambex
	rm -f venv/bin/kat_client venv/bin/teleproxy
# 2019-09-23
	rm -f kat-server-docker-image/kat-server
	rm -f kat-sandbox/grpc_auth/docker-compose.yml
	rm -f kat-sandbox/grpc_web/docker-compose.yaml
	rm -f kat-sandbox/grpc_web/*_pb.js
	rm -f kat-sandbox/http_auth/docker-compose.yml
# 2019-04-05 0388efe75c16540c71223320596accbbe3fe6ac2
	rm -f kat/kat/client

clobber: clean kill-docker-registry
	-rm -rf docs/node_modules
	-rm -rf venv && echo && echo "Deleted venv, run 'deactivate' command if your virtualenv is activated" || true

generate: ## Update generated sources that get committed to git
generate: pkg/api/kat/echo.pb.go
generate-clean: ## Delete generated sources that get committed to git (implies `make clobber`)
generate-clean: clobber
	rm -rf pkg/api
.PHONY: generate generate-clean

#
# Informational

print-%: ## Print the arbitrary Make expression '%'
	@printf "$($*)"
.PHONY: print-%

print-vars: ## Print variables of interest (in a human-friendly format)
	@echo "DOCKER_OPTS                      = $(DOCKER_OPTS)"
	@echo
	@echo "GIT_BRANCH                       = $(GIT_BRANCH)"
	@echo "GIT_COMMIT                       = $(GIT_COMMIT)"
	@echo "GIT_DIRTY                        = $(GIT_DIRTY)"
	@echo "GIT_DESCRIPTION                  = $(GIT_DESCRIPTION)"
	@echo
	@echo "RELEASE_DOCKER_REPO              = $(RELEASE_DOCKER_REPO)"
	@echo "BASE_DOCKER_REPO                 = $(BASE_DOCKER_REPO)"
	@echo "DEV_DOCKER_REPO                  = $(DEV_DOCKER_REPO)"
	@echo
	@echo "BUILD_VERSION                    = $(BUILD_VERSION)"
	@echo "RELEASE_VERSION                  = $(RELEASE_VERSION)"
	@echo "BASE_VERSION.envoy               = $(BASE_VERSION.envoy)"
	@echo "BASE_VERSION.runtime             = $(BASE_VERSION.runtime)"
	@echo "BASE_VERSION.py                  = $(BASE_VERSION.py)"
.PHONY: print-vars

export-vars: ## Print variables of interest (in a Bourne-shell format)
	@echo "export DOCKER_OPTS='$(DOCKER_OPTS)'"
	@echo
	@echo "export GIT_BRANCH='$(GIT_BRANCH)'"
	@echo "export GIT_COMMIT='$(GIT_COMMIT)'"
	@echo "export GIT_DIRTY='$(GIT_DIRTY)'"
	@echo "export GIT_DESCRIPTION='$(GIT_DESCRIPTION)'"
	@echo
	@echo "export RELEASE_DOCKER_REPO='$(RELEASE_DOCKER_REPO)'"
	@echo "export BASE_DOCKER_REPO='$(BASE_DOCKER_REPO)'"
	@echo "export DEV_DOCKER_REPO='$(DEV_DOCKER_REPO)'"
	@echo
	@echo "export BUILD_VERSION='$(BUILD_VERSION)'"
	@echo "export RELEASE_VERSION='$(RELEASE_VERSION)'"
.PHONY: export-vars

#
# Docker build

base-%.docker: docker/base-%/Dockerfile $(var.)BASE_IMAGE.% $(WRITE_IFCHANGED)
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
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
	docker image inspect $(BASE_IMAGE.$*) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

base-runtime.docker.DOCKER_OPTS =

base-py.docker: base-runtime.docker
base-py.docker.DOCKER_OPTS = --build-arg=BASE_RUNTIME_IMAGE=$$(cat base-runtime.docker)

test-%.docker: docker/test-%/Dockerfile $(MOVE_IFCHANGED) FORCE
	docker build --quiet --iidfile=$@.tmp $(<D)
	$(MOVE_IFCHANGED) $@.tmp $@

test-auth-tls.docker: docker/test-auth/Dockerfile $(MOVE_IFCHANGED) FORCE
	docker build --quiet --build-arg TLS=--tls --iidfile=$@.tmp $(<D)
	$(MOVE_IFCHANGED) $@.tmp $@

ambassador.docker: Dockerfile bin_linux_amd64/ambex bin_linux_amd64/watt bin_linux_amd64/kubestatus bin_linux_amd64/kubectl $(MOVE_IFCHANGED) python/ambassador/VERSION.py FORCE
	set -x; docker build $(DOCKER_OPTS) $($@.DOCKER_OPTS) --iidfile=$@.tmp .
	$(MOVE_IFCHANGED) $@.tmp $@
ambassador.docker: base-runtime.docker base-py.docker
ambassador.docker.DOCKER_OPTS += --build-arg=BASE_RUNTIME_IMAGE=$$(cat base-runtime.docker)
ambassador.docker.DOCKER_OPTS += --build-arg=BASE_PY_IMAGE=$$(cat base-py.docker)
ambassador.docker: $(ENVOY_FILE) $(var.)ENVOY_FILE
ambassador.docker.DOCKER_OPTS += --build-arg=ENVOY_FILE=$(ENVOY_FILE)

kat-client.docker: docker/kat-client/Dockerfile base-py.docker docker/kat-client/teleproxy docker/kat-client/kat_client $(MOVE_IFCHANGED)
	docker build --build-arg BASE_PY_IMAGE=$$(cat base-py.docker) $(DOCKER_OPTS) --iidfile=$@.tmp $(<D)
	$(MOVE_IFCHANGED) $@.tmp $@
docker/kat-client/teleproxy: docker/kat-client/%: bin_linux_amd64/%
	cp $< $@
docker/kat-client/kat_client: bin_linux_amd64/kat-client
	cp $< $@

kat-server.docker: $(wildcard docker/kat-server/*) docker/kat-server/kat-server $(MOVE_IFCHANGED)
	docker build $(DOCKER_OPTS) --iidfile=$@.tmp $(<D)
	$(MOVE_IFCHANGED) $@.tmp $@
docker/kat-server/kat-server: docker/kat-server/%: bin_linux_amd64/%
	cp $< $@

#
# Workflow

update-base: ## Run this whenever the base images (ex Envoy, ./docker/base-*/*) change
	$(MAKE) $(addsuffix .docker.tag.base,$(images.base))
	$(MAKE) generate
	$(MAKE) $(addsuffix .docker.push.base,$(images.base))
.PHONY: update-base

docker-push: ## Build and push the main Ambassador image to DEV_DOCKER_REPO
docker-push: ambassador.docker.push.dev
.PHONY: docker-push

lint: mypy

bin_%/kubectl: $(var.)KUBECTL_VERSION
	mkdir -p $(@D)
	curl --fail -o $@ -L https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl
	chmod 755 $@

setup-develop: ## TODO: document me
setup-develop: venv bin_$(GOHOSTOS)_$(GOHOSTARCH)/kubestatus python/ambassador/VERSION.py
.PHONY: setup-develop

setup-test: ## Perform setup for `make test`
setup-test: setup-develop $(addsuffix .docker.push.dev,$(images.cluster))
	rm -rf /tmp/k8s-*.yaml /tmp/kat-*.yaml
.PHONY: setup-test

# "make shell" drops you into a dev shell, and tries to set variables, etc., as
# needed:
#
# XXX KLF HACK: The dev shell used to include setting
# 	AMBASSADOR_DEV=1 \
# but I've ripped that out, since moving the KAT client into the cluster makes it
# much complex for the AMBASSADOR_DEV stuff to work correctly. I'll open an
# issue to finish sorting this out, but right now I want to get our CI builds 
# into better shape without waiting for that.

shell: ## Run a shell with the the virtualenv and such activated
shell: setup-develop
	bash --init-file releng/init.sh -i
.PHONY: shell

test: ## Run the test-suite
test: setup-test mypy
	cd python && env PATH="$(shell pwd)/venv/bin:$(PATH)" ../releng/run-tests.sh
.PHONY: test

test-list: ## List the tests in the test-suite
test-list: setup-develop
	cd python && PATH="$(shell pwd)/venv/bin":$(PATH) pytest --collect-only -q
.PHONY: test

# ------------------------------------------------------------------------------
# Virtualenv
# ------------------------------------------------------------------------------

venv: python/ambassador/VERSION.py venv/bin/ambassador

venv/bin/ambassador: venv/bin/activate python/requirements.txt
	@releng/install-py.sh dev requirements python/requirements.txt
	@releng/install-py.sh dev install python/requirements.txt
	@releng/fix_kube_client

venv/bin/activate: dev-requirements.txt
	test -d venv || virtualenv venv --python python3
	@releng/install-py.sh dev requirements $^
	@releng/install-py.sh dev install $^
	touch venv/bin/activate
	@releng/fix_kube_client

mypy-server-stop: venv
	venv/bin/dmypy stop
.PHONY: mypy-server-stop

mypy-server: venv
	@if ! venv/bin/dmypy status >/dev/null; then \
		venv/bin/dmypy start -- --use-fine-grained-cache --follow-imports=skip --ignore-missing-imports ;\
		echo "Started mypy server" ;\
	fi
.PHONY: mypy-server

mypy: mypy-server
	time venv/bin/dmypy check python
.PHONY: mypy
