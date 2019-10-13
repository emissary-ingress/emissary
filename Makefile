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

CI_DEBUG_KAT_BRANCH=

SHELL = bash

# Welcome to the Ambassador Makefile...

.PHONY: \
    clean version setup-develop print-vars \
    docker-push docker-images
.SECONDARY:

GIT_DIRTY ?= $(if $(shell git status --porcelain),dirty)

# This is only "kinda" the git branch name:
#
#  - if checked out is the synthetic merge-commit for a PR, then use
#    the PR's branch name (even though the merge commit we have
#    checked out isn't part of the branch")
#  - if this is a CI run for a tag (not a branch or PR), then use the
#    tag name
#  - if none of the above, then use the actual git branch name
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_BRANCH ?= $(or $(TRAVIS_PULL_REQUEST_BRANCH),$(TRAVIS_BRANCH),$(shell git rev-parse --abbrev-ref HEAD))

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

# This commands prints the tag of this commit or "undefined".
GIT_TAG ?= $(shell git name-rev --tags --name-only $(GIT_COMMIT))

GIT_BRANCH_SANITIZED := $(shell printf $(GIT_BRANCH) | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-zA-Z0-9]/-/g' -e 's/-\{2,\}/-/g')

# This gives the _previous_ tag, plus a git delta, like
# 0.36.0-436-g8b8c5d3
GIT_DESCRIPTION := $(shell git describe --tags $(GIT_COMMIT))

# IS_PRIVATE: empty=false, nonempty=true
# Default is true if any of the git remotes have the string "private" in any of their URLs.
_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

# RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
# will upload/release the image as.  It does NOT include a leading 'v'
# (trimming the 'v' from the git tag is what the 'patsubst' is for).
# If this is an RC or EA, then it includes the '-rcN' or '-eaN'
# suffix.
#
# BUILD_VERSION is of the same format, but is the version number that
# we build into the image.  Because an image built as a "release
# candidate" will ideally get promoted to be the GA image, we trim off
# the '-rcN' suffix.
RELEASE_VERSION = $(patsubst v%,%,$(or $(TRAVIS_TAG),$(shell git describe --tags --always)))$(if $(GIT_DIRTY),-dirty)
BUILD_VERSION = $(shell echo '$(RELEASE_VERSION)' | sed 's/-rc[0-9]*$$//')

ifndef DOCKER_REGISTRY
$(error DOCKER_REGISTRY must be set. Use make DOCKER_REGISTRY=- for a purely local build.)
endif

AMBASSADOR_DOCKER_REPO ?= $(if $(filter-out -,$(DOCKER_REGISTRY)),$(DOCKER_REGISTRY)/)ambassador$(if $(IS_PRIVATE),-private)

ifneq ($(DOCKER_EXTERNAL_REGISTRY),)
AMBASSADOR_EXTERNAL_DOCKER_REPO ?= $(DOCKER_EXTERNAL_REGISTRY)/ambassador$(if $(IS_PRIVATE),-private)
else
AMBASSADOR_EXTERNAL_DOCKER_REPO ?= $(AMBASSADOR_DOCKER_REPO)
endif

DOCKER_OPTS =

# This is the branch from ambassador-docs to pull for "make pull-docs".
# Override if you need to.
PULL_BRANCH ?= master

AMBASSADOR_DOCKER_TAG ?= $(RELEASE_VERSION)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)
AMBASSADOR_EXTERNAL_DOCKER_IMAGE ?= $(AMBASSADOR_EXTERNAL_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

YES_I_AM_UPDATING_THE_BASE_IMAGES ?=

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make docker-update-base`.
  # Increment BASE_RUNTIME_RELVER on changes to `Dockerfile.base-runtime`
  BASE_RUNTIME_RELVER ?= 1
  # Increment BASE_PY_RELVER on changes to `Dockerfile.base-py` or `python/requirements.txt`
  BASE_PY_RELVER      ?= 1

  BASE_DOCKER_REPO   ?= quay.io/datawire/ambassador-base$(if $(IS_PRIVATE),-private)
  BASE_IMAGE.runtime ?= $(BASE_DOCKER_REPO):runtime-$(BASE_RUNTIME_RELVER)
  BASE_IMAGE.py      ?= $(BASE_DOCKER_REPO):py-$(BASE_RUNTIME_RELVER).$(BASE_PY_RELVER)
# END LIST OF VARIABLES REQUIRING `make docker-update-base`.

#### Test service Dockerfile stuff.
# The test services live in subdirectories of test-services. TEST_SERVICE_ROOTS
# is the list of these directories.

TEST_SERVICE_ROOTS = $(notdir $(wildcard test-services/*))

# TEST_SERVICE_IMAGES maps each TEST_SERVICE_ROOT to test-$root.docker, since
# those are the names of the individual targets. We also add the auth-tls
# target here, by hand -- it has a special rule since it's also built from the
# test-services/auth directory.
TEST_SERVICE_IMAGES = $(patsubst %,test-%.docker,$(TEST_SERVICE_ROOTS) auth-tls)

# Set default tag values...
docker.tag.release    = $(AMBASSADOR_DOCKER_TAG)
docker.tag.release-rc = $(AMBASSADOR_EXTERNAL_DOCKER_REPO):$(RELEASE_VERSION) $(AMBASSADOR_EXTERNAL_DOCKER_REPO):$(BUILD_VERSION)-latest-rc
docker.tag.release-ea = $(AMBASSADOR_EXTERNAL_DOCKER_REPO):$(RELEASE_VERSION)
docker.tag.local      = $(AMBASSADOR_DOCKER_TAG)
docker.tag.base       = $(BASE_IMAGE.$(patsubst base-%.docker,%,$<))

TEST_SERVICE_VERSION ?= 0.0.3

# ...then set overrides for the test services.
test-%.docker.tag.release: docker.tag.release = quay.io/datawire/test_services:$(notdir $*)-$(TEST_SERVICE_VERSION)

LOCAL_REPO = $(if $(filter-out -,$(DOCKER_REGISTRY)),$(DOCKER_REGISTRY)/test_services,test_services)
test-%.docker.tag.local: docker.tag.local = $(LOCAL_REPO):$(notdir $*)-$(GIT_DESCRIPTION)

# ...and define some TEST_SERVICE_*_TAGS.
TEST_SERVICE_LOCAL_TAGS = $(addsuffix .tag.local,$(TEST_SERVICE_IMAGES))
TEST_SERVICE_RELEASE_TAGS = $(addsuffix .tag.release,$(TEST_SERVICE_IMAGES))

ifneq ($(DOCKER_REGISTRY), -)
TEST_SERVICE_LOCAL_PUSHES = $(addsuffix .push.local,$(TEST_SERVICE_IMAGES))
TEST_SERVICE_RELEASE_PUSHES = $(addsuffix .push.release,$(TEST_SERVICE_IMAGES))
endif

#### end test service stuff

# Default to _NOT_ using Kubernaut. At Datawire, we can set this to true,
# but outside, it works much better to assume that user has set up something
# and not try to override it.
USE_KUBERNAUT ?= false

KUBERNAUT=venv/bin/kubernaut
KUBERNAUT_VERSION=2018.10.24-d46c1f1
KUBERNAUT_CLAIM=$(KUBERNAUT) claims create --name $(CLAIM_NAME) --cluster-group main
KUBERNAUT_DISCARD=$(KUBERNAUT) claims delete $(CLAIM_NAME)

KUBECTL_VERSION = 1.16.1

# Only override KUBECONFIG if we're using Kubernaut
ifeq ($(USE_KUBERNAUT), true)
KUBECONFIG ?= $(shell pwd)/cluster.yaml
endif

SCOUT_APP_KEY=

KAT_CLIENT_DOCKER_REPO ?= $(if $(filter-out -,$(DOCKER_REGISTRY)),$(DOCKER_REGISTRY)/)kat-client$(if $(IS_PRIVATE),-private)
KAT_SERVER_DOCKER_REPO ?= $(if $(filter-out -,$(DOCKER_REGISTRY)),$(DOCKER_REGISTRY)/)kat-backend$(if $(IS_PRIVATE),-private)

KAT_CLIENT_DOCKER_IMAGE ?= $(KAT_CLIENT_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)
KAT_SERVER_DOCKER_IMAGE ?= $(KAT_SERVER_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

KAT_IMAGE_PULL_POLICY ?= Always

# This should maybe be replaced with a lighterweight dependency if we
# don't currently depend on go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

CLAIM_FILE=kubernaut-claim.txt
CLAIM_NAME=$(shell cat $(CLAIM_FILE))


# "make" by itself doesn't make the website. It takes too long and it doesn't
# belong in the inner dev loop.
all:
	$(MAKE) setup-develop
	$(MAKE) docker-push
	$(MAKE) test

go.bins.extra += github.com/datawire/teleproxy/cmd/kubestatus
go.bins.extra += github.com/datawire/teleproxy/cmd/teleproxy
go.bins.extra += github.com/datawire/teleproxy/cmd/watt
export CGO_ENABLED = 0

include build-aux/prelude.mk
include build-aux/var.mk
include build-aux/docker.mk
include build-aux/common.mk
include build-aux/go-mod.mk
include cxx/envoy.mk

# clean_docker_images could just be a fixed list of the images that we
# generate, but writing that fixed list would require a human to
# figure out what that complete list is.
clean_docker_images  = $(wildcard *.docker)
clean_docker_images += $(patsubst %.tag.release,%,$(wildcard *.docker.tag.release))
clean_docker_images += $(patsubst %.tag.local,%,$(wildcard *.docker.tag.local))
clean: clean-test $(addsuffix .clean,$(clean_docker_images))
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
	rm -f build/kat/client/kat_client
	rm -f build/kat/server/kat-server
	rm -f tools/sandbox/http_auth/docker-compose.yml
	rm -f tools/sandbox/grpc_auth/docker-compose.yml
	rm -f tools/sandbox/grpc_web/docker-compose.yaml tools/sandbox/grpc_web/*_pb.js
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
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
	-rm -f build/kat/client/teleproxy
	-rm -rf docs/node_modules
	-rm -rf venv && echo && echo "Deleted venv, run 'deactivate' command if your virtualenv is activated" || true

generate: pkg/api/kat/echo.pb.go
generate-clean: clobber
	rm -rf pkg/api
.PHONY: generate generate-clean

print-%:
	@printf "$($*)"

print-vars:
	@echo "AMBASSADOR_DOCKER_IMAGE          = $(AMBASSADOR_DOCKER_IMAGE)"
	@echo "AMBASSADOR_DOCKER_REPO           = $(AMBASSADOR_DOCKER_REPO)"
	@echo "AMBASSADOR_DOCKER_TAG            = $(AMBASSADOR_DOCKER_TAG)"
	@echo "AMBASSADOR_EXTERNAL_DOCKER_IMAGE = $(AMBASSADOR_EXTERNAL_DOCKER_IMAGE)"
	@echo "AMBASSADOR_EXTERNAL_DOCKER_REPO  = $(AMBASSADOR_EXTERNAL_DOCKER_REPO)"
	@echo "CI_DEBUG_KAT_BRANCH              = $(CI_DEBUG_KAT_BRANCH)"
	@echo "DOCKER_EPHEMERAL_REGISTRY        = $(DOCKER_EPHEMERAL_REGISTRY)"
	@echo "DOCKER_EXTERNAL_REGISTRY         = $(DOCKER_EXTERNAL_REGISTRY)"
	@echo "DOCKER_OPTS                      = $(DOCKER_OPTS)"
	@echo "DOCKER_REGISTRY                  = $(DOCKER_REGISTRY)"
	@echo "BASE_DOCKER_REPO                 = $(BASE_DOCKER_REPO)"
	@echo "GIT_BRANCH                       = $(GIT_BRANCH)"
	@echo "GIT_BRANCH_SANITIZED             = $(GIT_BRANCH_SANITIZED)"
	@echo "GIT_COMMIT                       = $(GIT_COMMIT)"
	@echo "GIT_DESCRIPTION                  = $(GIT_DESCRIPTION)"
	@echo "GIT_DIRTY                        = $(GIT_DIRTY)"
	@echo "GIT_TAG                          = $(GIT_TAG)"
	@echo "KAT_CLIENT_DOCKER_IMAGE          = $(KAT_CLIENT_DOCKER_IMAGE)"
	@echo "KAT_SERVER_DOCKER_IMAGE          = $(KAT_SERVER_DOCKER_IMAGE)"
	@echo "KUBECONFIG                       = $(KUBECONFIG)"
	@echo "USE_KUBERNAUT                    = $(USE_KUBERNAUT)"
	@echo "BUILD_VERSION                    = $(BUILD_VERSION)"
	@echo "RELEASE_VERSION                  = $(RELEASE_VERSION)"

export-vars:
	@echo "export AMBASSADOR_DOCKER_IMAGE='$(AMBASSADOR_DOCKER_IMAGE)'"
	@echo "export AMBASSADOR_DOCKER_REPO='$(AMBASSADOR_DOCKER_REPO)'"
	@echo "export AMBASSADOR_DOCKER_TAG='$(AMBASSADOR_DOCKER_TAG)'"
	@echo "export AMBASSADOR_EXTERNAL_DOCKER_IMAGE='$(AMBASSADOR_EXTERNAL_DOCKER_IMAGE)'"
	@echo "export AMBASSADOR_EXTERNAL_DOCKER_REPO='$(AMBASSADOR_EXTERNAL_DOCKER_REPO)'"
	@echo "export CI_DEBUG_KAT_BRANCH='$(CI_DEBUG_KAT_BRANCH)'"
	@echo "export DOCKER_EPHEMERAL_REGISTRY='$(DOCKER_EPHEMERAL_REGISTRY)'"
	@echo "export DOCKER_EXTERNAL_REGISTRY='$(DOCKER_EXTERNAL_REGISTRY)'"
	@echo "export DOCKER_OPTS='$(DOCKER_OPTS)'"
	@echo "export DOCKER_REGISTRY='$(DOCKER_REGISTRY)'"
	@echo "export BASE_DOCKER_REPO='$(BASE_DOCKER_REPO)'"
	@echo "export GIT_BRANCH='$(GIT_BRANCH)'"
	@echo "export GIT_BRANCH_SANITIZED='$(GIT_BRANCH_SANITIZED)'"
	@echo "export GIT_COMMIT='$(GIT_COMMIT)'"
	@echo "export GIT_DESCRIPTION='$(GIT_DESCRIPTION)'"
	@echo "export GIT_DIRTY='$(GIT_DIRTY)'"
	@echo "export GIT_TAG='$(GIT_TAG)'"
	@echo "export KAT_CLIENT_DOCKER_IMAGE='$(KAT_CLIENT_DOCKER_IMAGE)'"
	@echo "export KAT_SERVER_DOCKER_IMAGE='$(KAT_SERVER_DOCKER_IMAGE)'"
	@echo "export KUBECONFIG='$(KUBECONFIG)'"
	@echo "export USE_KUBERNAUT='$(USE_KUBERNAUT)'"
	@echo "export BUILD_VERSION='$(BUILD_VERSION)'"
	@echo "export RELEASE_VERSION='$(RELEASE_VERSION)'"

# All of this will likely fail horribly outside of CI, for the record.
docker-registry: $(KUBECONFIG)
ifneq ($(DOCKER_EPHEMERAL_REGISTRY),)
	@if [ "$(TRAVIS)" != "true" ]; then \
		echo "make docker-registry is only for CI" >&2 ;\
		exit 1 ;\
	fi
	@if [ -z "$(KUBECONFIG)" ]; then \
		echo "No KUBECONFIG" >&2 ;\
		exit 1 ;\
	fi
	@if [ ! -r .docker_port_forward ]; then \
		echo "Starting local Docker registry in Kubernetes" ;\
		kubectl apply -f releng/docker-registry.yaml ;\
		while [ -z "$$(kubectl get pods -n docker-registry -ojsonpath='{.items[0].status.containerStatuses[0].state.running}')" ]; do echo pod wait...; sleep 1; done ;\
		sh -c 'kubectl port-forward --namespace=docker-registry deployment/registry 31000:5000 > /tmp/port-forward-log & echo $$! > .docker_port_forward' ;\
	else \
		echo "Local Docker registry should be already running" ;\
	fi
	while ! curl -i http://localhost:31000/ 2>/dev/null; do echo curl wait...; sleep 1; done
endif

kill-docker-registry:
	@if [ -r .docker_port_forward ]; then \
		echo "Stopping local Docker registry" ;\
		kill $$(cat .docker_port_forward) ;\
		kubectl delete -f releng/docker-registry.yaml ;\
		rm -f .docker_port_forward ;\
	else \
		echo "Docker registry should not be running" ;\
	fi

base-%.docker: Dockerfile.base-% $(var.)BASE_IMAGE.% $(WRITE_IFCHANGED)
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

test-%.docker: test-services/%/Dockerfile $(MOVE_IFCHANGED) FORCE
	docker build --quiet --iidfile=$@.tmp test-services/$*
	$(MOVE_IFCHANGED) $@.tmp $@

test-auth-tls.docker: test-services/auth/Dockerfile $(MOVE_IFCHANGED) FORCE
	docker build --quiet --build-arg TLS=--tls --iidfile=$@.tmp test-services/auth
	$(MOVE_IFCHANGED) $@.tmp $@

test-services: $(TEST_SERVICE_IMAGES) $(TEST_SERVICE_LOCAL_TAGS) $(TEST_SERVICE_LOCAL_PUSHES)
test-services-release: $(TEST_SERVICE_IMAGES) $(TEST_SERVICE_RELEASE_TAGS) $(TEST_SERVICE_RELEASE_PUSHES)
.PHONY: test-services test-services-release

# XXX: Why doesn't just test-%.docker.push: test-%.docker.push.local work??
#
# This three-element form of $(addsuffix ...) is kind of an implicit foreach. 
# We're generating a rule for each word in $(TEST_SERVICE_IMAGES).
TEST_SERVICE_PUSH_TARGETS = $(addsuffix .push,$(TEST_SERVICE_IMAGES))
$(TEST_SERVICE_PUSH_TARGETS): %.push: %.push.local
.PHONY: $(TEST_SERVICE_PUSH_TARGETS)

docker-base-images: $(addsuffix .docker.tag.base,base-envoy base-runtime base-py)

docker-push-base-images: $(addsuffix .docker.push.base,base-envoy base-runtime base-py)

docker-update-base:
	$(MAKE) docker-base-images generate
	$(MAKE) docker-push-base-images

ambassador-docker-image: ambassador.docker
ambassador.docker: Dockerfile bin_linux_amd64/ambex bin_linux_amd64/watt bin_linux_amd64/kubestatus bin_linux_amd64/kubectl $(WRITE_IFCHANGED) python/ambassador/VERSION.py FORCE
	set -x; docker build $(DOCKER_OPTS) $($@.DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) .
	@docker image inspect $(AMBASSADOR_DOCKER_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@
ambassador.docker: base-runtime.docker base-py.docker
ambassador.docker.DOCKER_OPTS += --build-arg=BASE_RUNTIME_IMAGE=$$(cat base-runtime.docker)
ambassador.docker.DOCKER_OPTS += --build-arg=BASE_PY_IMAGE=$$(cat base-py.docker)
ambassador.docker: $(ENVOY_FILE) $(var.)ENVOY_FILE
ambassador.docker.DOCKER_OPTS += --build-arg=ENVOY_FILE=$(ENVOY_FILE)

kat-client-docker-image: kat-client.docker
.PHONY: kat-client-docker-image
kat-client.docker: build/kat/client/Dockerfile base-py.docker build/kat/client/teleproxy build/kat/client/kat_client $(WRITE_IFCHANGED) $(var.)KAT_CLIENT_DOCKER_IMAGE
	docker build --build-arg BASE_PY_IMAGE=$$(cat base-py.docker) $(DOCKER_OPTS) -t $(KAT_CLIENT_DOCKER_IMAGE) build/kat/client
	@docker image inspect $(KAT_CLIENT_DOCKER_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

# build/kat/client/teleproxy always uses the linux/amd64 architecture
build/kat/client/teleproxy: bin_linux_amd64/teleproxy
	cp $< $@

# build/kat/client/kat_client always uses the linux/amd64 architecture
build/kat/client/kat_client: bin_linux_amd64/kat-client
	cp $< $@

kat-server-docker-image: kat-server.docker
.PHONY:  kat-server-docker-image
kat-server.docker: $(wildcard build/kat/server/*) build/kat/server/kat-server $(var.)KAT_SERVER_DOCKER_IMAGE
	docker build $(DOCKER_OPTS) -t $(KAT_SERVER_DOCKER_IMAGE) build/kat/server
	@docker image inspect $(KAT_SERVER_DOCKER_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

# build/kat/server/kat-server always uses the linux/amd64 architecture
build/kat/server/kat-server: bin_linux_amd64/kat-server
	cp $< $@

docker-images: mypy ambassador-docker-image

docker-push: docker-images
ifeq ($(DOCKER_REGISTRY),-)
	@echo "No DOCKER_REGISTRY set"
else
	@echo 'PUSH $(AMBASSADOR_DOCKER_IMAGE)'
	@set -o pipefail; \
		docker push $(AMBASSADOR_DOCKER_IMAGE) | python releng/linify.py push.log
endif

docker-push-kat-client: kat-client-docker-image
	@echo 'PUSH $(KAT_CLIENT_DOCKER_IMAGE)'
	@set -o pipefail; \
		docker push $(KAT_CLIENT_DOCKER_IMAGE) | python releng/linify.py push.log

docker-push-kat-server: kat-server-docker-image
	@echo 'PUSH $(KAT_SERVER_DOCKER_IMAGE)'
	@set -o pipefail; \
		docker push $(KAT_SERVER_DOCKER_IMAGE) | python releng/linify.py push.log

docker-push-kat: docker-push-kat-client docker-push-kat-server

# TODO: validate version is conformant to some set of rules might be a good idea to add here
python/ambassador/VERSION.py: FORCE $(WRITE_IFCHANGED)
	$(call check_defined, BUILD_VERSION, BUILD_VERSION is not set)
	$(call check_defined, GIT_BRANCH, GIT_BRANCH is not set)
	$(call check_defined, GIT_COMMIT, GIT_COMMIT is not set)
	$(call check_defined, GIT_DESCRIPTION, GIT_DESCRIPTION is not set)
	@echo "Generating and templating version information -> $(BUILD_VERSION)"
	sed \
		-e 's!{{VERSION}}!$(BUILD_VERSION)!g' \
		-e 's!{{GITBRANCH}}!$(GIT_BRANCH)!g' \
		-e 's!{{GITDIRTY}}!$(GIT_DIRTY)!g' \
		-e 's!{{GITCOMMIT}}!$(GIT_COMMIT)!g' \
		-e 's!{{GITDESCRIPTION}}!$(GIT_DESCRIPTION)!g' \
		< VERSION-template.py | $(WRITE_IFCHANGED) $@

version: python/ambassador/VERSION.py

$(CLAIM_FILE):
	@if [ -z $${CI+x} ]; then \
		echo kat-$${USER} > $@; \
	else \
		echo kat-$${USER}-$(shell uuidgen) > $@; \
	fi

$(KUBERNAUT): $(var.)KUBERNAUT_VERSION $(var.)GOOS $(var.)GOARCH | venv/bin/activate
	curl -o $(KUBERNAUT) http://releases.datawire.io/kubernaut/$(KUBERNAUT_VERSION)/$(GOOS)/$(GOARCH)/kubernaut
	chmod +x $(KUBERNAUT)

bin_%/kubectl: $(var.)KUBECTL_VERSION
	mkdir -p $(@D)
	curl --fail -o $@ -L https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl
	chmod 755 $@

setup-develop: venv $(KUBERNAUT) bin_$(GOHOSTOS)_$(GOHOSTARCH)/kubestatus version

cluster.yaml: $(CLAIM_FILE) $(KUBERNAUT)
ifeq ($(USE_KUBERNAUT), true)
	$(KUBERNAUT_DISCARD)
	$(KUBERNAUT_CLAIM)
	cp ~/.kube/$(CLAIM_NAME).yaml cluster.yaml
else
ifneq ($(USE_KUBERNAUT),)
ifneq ($(USE_KUBERNAUT),false)
	@echo "USE_KUBERNAUT must be true, false, or unset" >&2
	false
endif
endif
endif
# Make is too dumb to understand equivalence between absolute and
# relative paths.
$(CURDIR)/cluster.yaml: cluster.yaml

setup-test: cluster.yaml
	rm -rf /tmp/k8s-*.yaml /tmp/kat-*.yaml

# "make shell" drops you into a dev shell, and tries to set variables, etc., as
# needed:
#
# If USE_KUBERNAUT is true, we'll set up for Kubernaut, otherwise we'll assume 
# that the current KUBECONFIG is good.
#
# XXX KLF HACK: The dev shell used to include setting
# 	AMBASSADOR_DEV=1 \
# but I've ripped that out, since moving the KAT client into the cluster makes it
# much complex for the AMBASSADOR_DEV stuff to work correctly. I'll open an
# issue to finish sorting this out, but right now I want to get our CI builds 
# into better shape without waiting for that.

shell: setup-develop
	env \
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	BASE_IMAGE.envoy="$(BASE_IMAGE.envoy)" \
	BASE_IMAGE.runtime="$(BASE_IMAGE.runtime)" \
	BASE_IMAGE.py="$(BASE_IMAGE.py)" \
	MAKE_KUBECONFIG="$(KUBECONFIG)" \
	bash --init-file releng/init.sh -i

clean-test:
	rm -f cluster.yaml
	test -x $(KUBERNAUT) && $(KUBERNAUT_DISCARD) || true
	rm -f $(CLAIM_FILE)

test: setup-develop
	cd python && env \
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	BASE_IMAGE.envoy="$(BASE_IMAGE.envoy)" \
	BASE_IMAGE.runtime="$(BASE_IMAGE.runtime)" \
	BASE_IMAGE.py="$(BASE_IMAGE.py)" \
	KUBECONFIG="$(KUBECONFIG)" \
	KAT_CLIENT_DOCKER_IMAGE="$(KAT_CLIENT_DOCKER_IMAGE)" \
	KAT_SERVER_DOCKER_IMAGE="$(KAT_SERVER_DOCKER_IMAGE)" \
	KAT_IMAGE_PULL_POLICY="$(KAT_IMAGE_PULL_POLICY)" \
	PATH="$(shell pwd)/venv/bin:$(PATH)" \
	bash ../releng/run-tests.sh

test-list: setup-develop
	cd python && PATH="$(shell pwd)/venv/bin":$(PATH) pytest --collect-only -q

update-aws:
ifeq ($(AWS_ACCESS_KEY_ID),)
	@echo 'AWS credentials not configured; not updating https://s3.amazonaws.com/datawire-static-files/ambassador/$(STABLE_TXT_KEY)'
	@echo 'AWS credentials not configured; not updating latest version in Scout'
else
	@if [ -n "$(STABLE_TXT_KEY)" ]; then \
        printf "$(RELEASE_VERSION)" > stable.txt; \
		echo "updating $(STABLE_TXT_KEY) with $$(cat stable.txt)"; \
        aws s3api put-object \
            --bucket datawire-static-files \
            --key ambassador/$(STABLE_TXT_KEY) \
            --body stable.txt; \
	fi
	@if [ -n "$(SCOUT_APP_KEY)" ]; then \
		printf '{"application":"ambassador","latest_version":"$(RELEASE_VERSION)","notices":[]}' > app.json; \
		echo "updating $(SCOUT_APP_KEY) with $$(cat app.json)"; \
        aws s3api put-object \
            --bucket scout-datawire-io \
            --key ambassador/$(SCOUT_APP_KEY) \
            --body app.json; \
	fi
endif

release-prep:
	bash releng/release-prep.sh

release:
	@if ! [[ '$(RELEASE_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		printf "'make release' can only be run for commit tagged with 'vX.Y.Z'!\n"; \
		exit 1; \
	fi
	docker pull $(AMBASSADOR_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest
	docker tag $(AMBASSADOR_DOCKER_REPO):$(RELEASE_VERSION)-rc-latest $(AMBASSADOR_DOCKER_REPO):$(RELEASE_VERSION)
	docker push $(AMBASSADOR_DOCKER_REPO):$(RELEASE_VERSION)
	$(MAKE) SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws

release-rc: ambassador.docker.push.release-rc
release-rc: SCOUT_APP_KEY = testapp.json
release-rc: STABLE_TXT_KEY = teststable.txt
release-rc: update-aws

release-ea: ambassador.docker.push.release-ea
release-ea: SCOUT_APP_KEY = earlyapp.json
release-ea: STABLE_TXT_KEY = earlystable.txt
release-ea: update-aws

# ------------------------------------------------------------------------------
# gRPC bindings for KAT
# ------------------------------------------------------------------------------

GRPC_WEB_VERSION = 1.0.3
GRPC_WEB_PLATFORM = $(GOOS)-x86_64

venv/bin/protoc-gen-grpc-web: $(var.)GRPC_WEB_VERSION $(var.)GRPC_WEB_PLATFORM | venv/bin/activate
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@

pkg/api/kat/echo.pb.go: api/kat/echo.proto venv/bin/protoc venv/bin/protoc-gen-gogofast
	mkdir -p $(@D)
	./venv/bin/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/venv/bin/protoc-gen-gogofast --gogofast_out=plugins=grpc:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_grpc_web_pb.js: api/kat/echo.proto venv/bin/protoc venv/bin/protoc-gen-grpc-web
	./venv/bin/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/venv/bin/protoc-gen-grpc-web --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_pb.js: api/kat/echo.proto venv/bin/protoc
	./venv/bin/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--js_out=import_style=commonjs:$(@D) \
		$(CURDIR)/$<

# ------------------------------------------------------------------------------
# KAT docker-compose sandbox
# ------------------------------------------------------------------------------

tools/sandbox/http_auth/docker-compose.yml tools/sandbox/grpc_auth/docker-compose.yml tools/sandbox/grpc_web/docker-compose.yaml: %: %.in kat-server.docker $(var.)KAT_SERVER_DOCKER_IMAGE
	sed 's,@KAT_SERVER_DOCKER_IMAGE@,$(KAT_SERVER_DOCKER_IMAGE),g' < $< > $@

tools/sandbox.http-auth: ## In docker-compose: run Ambassador, an HTTP AuthService, an HTTP backend service, and a TracingService
tools/sandbox.http-auth: tools/sandbox/http_auth/docker-compose.yml
	@echo " ---> cleaning HTTP auth tools/sandbox"
	@cd tools/sandbox/http_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting HTTP auth tools/sandbox"
	@cd tools/sandbox/http_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.http-auth

tools/sandbox.grpc-auth: ## In docker-compose: run Ambassador, a gRPC AuthService, an HTTP backend service, and a TracingService
tools/sandbox.grpc-auth: tools/sandbox/grpc_auth/docker-compose.yml
	@echo " ---> cleaning gRPC auth tools/sandbox"
	@cd tools/sandbox/grpc_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC auth tools/sandbox"
	@cd tools/sandbox/grpc_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.grpc-auth

tools/sandbox.web: ## In docker-compose: run Ambassador with gRPC-web enabled, and a gRPC backend service
tools/sandbox.web: tools/sandbox/grpc_web/docker-compose.yaml
tools/sandbox.web: tools/sandbox/grpc_web/echo_grpc_web_pb.js tools/sandbox/grpc_web/echo_pb.js
	@echo " ---> cleaning gRPC web tools/sandbox"
	@cd tools/sandbox/grpc_web && npm install && npx webpack
	@cd tools/sandbox/grpc_web && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC web tools/sandbox"
	@cd tools/sandbox/grpc_web && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.web

# ------------------------------------------------------------------------------
# Virtualenv
# ------------------------------------------------------------------------------

venv: version venv/bin/ambassador

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

mypy-server: venv
	@if ! venv/bin/dmypy status >/dev/null; then \
		venv/bin/dmypy start -- --use-fine-grained-cache --follow-imports=skip --ignore-missing-imports ;\
		echo "Started mypy server" ;\
	fi

mypy: mypy-server
	time venv/bin/dmypy check python

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

pull-docs:
	@PS4=; set -ex; { \
	    git fetch https://github.com/datawire/ambassador-docs $(PULL_BRANCH); \
	    docs_head=$$(git rev-parse FETCH_HEAD); \
	    git subtree merge --prefix=docs "$${docs_head}"; \
	    git subtree split --prefix=docs --rejoin --onto="$${docs_head}"; \
	}
push-docs:
	@PS4=; set -ex; { \
	    git fetch https://github.com/datawire/ambassador-docs master; \
	    docs_old=$$(git rev-parse FETCH_HEAD); \
	    docs_new=$$(git subtree split --prefix=docs --rejoin --onto="$${docs_old}"); \
	    git push $(if $(GH_TOKEN),https://d6e-automaton:${GH_TOKEN}@github.com/,git@github.com:)datawire/ambassador-docs.git "$${docs_new}:refs/heads/$(or $(PUSH_BRANCH),master)"; \
	}
.PHONY: pull-docs push-docs

# ------------------------------------------------------------------------------
# Function Definitions
# ------------------------------------------------------------------------------

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = $(strip $(foreach 1,$1, $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),, $(error Undefined $1$(if $2, ($2))))
