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
    docker-push docker-images \
    teleproxy-restart teleproxy-stop
.SECONDARY:

# GIT_BRANCH on TravisCI needs to be set through some external custom logic. Default to a Git native mechanism or
# use what is defined.
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_DIRTY ?= $(shell test -z "$(shell git status --porcelain)" || printf "dirty")

GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

# This commands prints the tag of this commit or "undefined". Later we use GIT_TAG_SANITIZED and set it to "" if this
# string is "undefined" or blank.
GIT_TAG ?= $(shell git name-rev --tags --name-only $(GIT_COMMIT))

GIT_BRANCH_SANITIZED := $(shell printf $(GIT_BRANCH) | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-zA-Z0-9]/-/g' -e 's/-\{2,\}/-/g')
GIT_TAG_SANITIZED := $(shell \
	if [ "$(GIT_TAG)" = "undefined" -o -z "$(GIT_TAG)" ]; then \
		printf ""; \
	else \
		printf "$(GIT_TAG)" | sed -e 's/\^.*//g'; \
	fi \
)

# Trees get dirty sometimes by choice and sometimes accidently. If we are in a dirty tree then append "-dirty" to the
# GIT_COMMIT.
ifeq ($(GIT_DIRTY),dirty)
GIT_VERSION := $(GIT_BRANCH_SANITIZED)-$(GIT_COMMIT)-dirty
else
GIT_VERSION := $(GIT_BRANCH_SANITIZED)-$(GIT_COMMIT)
endif

# This gives the _previous_ tag, plus a git delta, like
# 0.36.0-436-g8b8c5d3
GIT_DESCRIPTION := $(shell git describe $(GIT_COMMIT))

# Note that for everything except RC builds, VERSION will be set to the version
# we'd use for a GA build. This is by design.
ifneq ($(GIT_TAG_SANITIZED),)
VERSION = $(shell printf "$(GIT_TAG_SANITIZED)" | sed -e 's/-.*//g')
else
VERSION = $(GIT_VERSION)
endif

# We need this for tagging in some situations.
LATEST_RC=$(VERSION)-rc-latest

ifndef DOCKER_REGISTRY
$(error DOCKER_REGISTRY must be set. Use make DOCKER_REGISTRY=- for a purely local build.)
endif

ifeq ($(DOCKER_REGISTRY), -)
AMBASSADOR_DOCKER_REPO ?= ambassador
else
AMBASSADOR_DOCKER_REPO ?= $(DOCKER_REGISTRY)/ambassador
endif

ifneq ($(DOCKER_EXTERNAL_REGISTRY),)
AMBASSADOR_EXTERNAL_DOCKER_REPO ?= $(DOCKER_EXTERNAL_REGISTRY)/ambassador
else
AMBASSADOR_EXTERNAL_DOCKER_REPO ?= $(AMBASSADOR_DOCKER_REPO)
endif

DOCKER_OPTS =

# This is the branch from ambassador-docs to pull for "make pull-docs".
# Override if you need to.
PULL_BRANCH ?= master

NETLIFY_SITE=datawire-ambassador

AMBASSADOR_DOCKER_TAG ?= $(GIT_VERSION)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)
AMBASSADOR_EXTERNAL_DOCKER_IMAGE ?= $(AMBASSADOR_EXTERNAL_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make docker-update-base`.
  ENVOY_REPO ?= git://github.com/datawire/envoy.git
  ENVOY_COMMIT ?= 8f57f7d765939552a999721e8dac9b5a9a5cbb8b
  ENVOY_COMPILATION_MODE ?= dbg

  ENVOY_FILE ?= envoy-bin/envoy-static-stripped

  # Increment BASE_ENVOY_RELVER on changes to `Dockerfile.base-envoy`, ENVOY_FILE, or Envoy recipes
  BASE_ENVOY_RELVER ?= 3
  # Increment BASE_GO_RELVER on changes to `Dockerfile.base-go`
  BASE_GO_RELVER    ?= 1
  # Increment BASE_PY_RELVER on changes to `Dockerfile.base-py`, `releng/*`, `multi/requirements.txt`, `ambassador/requirements.txt`
  BASE_PY_RELVER    ?= 1

  BASE_DOCKER_REPO ?= quay.io/datawire/ambassador-base
  BASE_ENVOY_IMAGE ?= $(BASE_DOCKER_REPO):envoy-$(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)
  BASE_GO_IMAGE    ?= $(BASE_DOCKER_REPO):go-$(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE).$(BASE_GO_RELVER)
  BASE_PY_IMAGE    ?= $(BASE_DOCKER_REPO):py-$(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE).$(BASE_PY_RELVER)
# END LIST OF VARIABLES REQUIRING `make docker-update-base`.

# Default to _NOT_ using Kubernaut. At Datawire, we can set this to true,
# but outside, it works much better to assume that user has set up something
# and not try to override it.
USE_KUBERNAUT ?= false

# Only override KUBECONFIG if we're using Kubernaut
ifeq ($(USE_KUBERNAUT), true)
KUBECONFIG ?= $(shell pwd)/cluster.yaml
endif

SCOUT_APP_KEY=

# Sets the kat-backend release which contains the kat-client use for E2e testing.
# For details https://github.com/datawire/kat-backend
#
# Note that this must not start with the 'v'. Sigh.
KAT_BACKEND_RELEASE = 1.5.0

# Allow overriding which watt we use.
WATT ?= watt
WATT_VERSION ?= 0.6.0

# "make" by itself doesn't make the website. It takes too long and it doesn't
# belong in the inner dev loop.
all:
	$(MAKE) setup-develop
	$(MAKE) docker-push
	$(MAKE) test

include build-aux/prelude.mk
include build-aux/var.mk

clean: clean-test
	rm -rf docs/_book docs/_site docs/package-lock.json
	rm -rf helm/*.tgz
	rm -rf app.json
	rm -rf venv/bin/ambassador
	rm -rf ambassador/ambassador/VERSION.py*
	rm -f *.docker
	rm -rf ambassador/build ambassador/dist ambassador/ambassador.egg-info ambassador/__pycache__
	find . \( -name .coverage -o -name .cache -o -name __pycache__ \) -print0 | xargs -0 rm -rf
	find . \( -name *.log \) -print0 | xargs -0 rm -rf
	find ambassador/tests \
		\( -name '*.out' -o -name 'envoy.json' -o -name 'intermediate.json' \) -print0 \
		| xargs -0 rm -f
	rm -rf envoy-bin
	rm -f envoy-build-image.txt

clobber: clean
	-rm -rf watt
	-$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf envoy envoy-src)
	-rm -rf docs/node_modules
	-rm -rf venv && echo && echo "Deleted venv, run 'deactivate' command if your virtualenv is activated" || true

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
	@echo "GIT_TAG_SANITIZED                = $(GIT_TAG_SANITIZED)"
	@echo "GIT_VERSION                      = $(GIT_VERSION)"
	@echo "KAT_BACKEND_RELEASE              = $(KAT_BACKEND_RELEASE)"
	@echo "KUBECONFIG                       = $(KUBECONFIG)"
	@echo "LATEST_RC                        = $(LATEST_RC)"
	@echo "USE_KUBERNAUT                    = $(USE_KUBERNAUT)"
	@echo "VERSION                          = $(VERSION)"

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
	@echo "export GIT_TAG_SANITIZED='$(GIT_TAG_SANITIZED)'"
	@echo "export GIT_VERSION='$(GIT_VERSION)'"
	@echo "export KAT_BACKEND_RELEASE='$(KAT_BACKEND_RELEASE)'"
	@echo "export KUBECONFIG='$(KUBECONFIG)'"
	@echo "export LATEST_RC='$(LATEST_RC)'"
	@echo "export USE_KUBERNAUT='$(USE_KUBERNAUT)'"
	@echo "export VERSION='$(VERSION)'"

# All of this will likely fail horribly outside of CI, for the record.
docker-registry:
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
		sh -c 'kubectl port-forward --namespace=docker-registry deployment/registry 31000:5000 & echo $$! > .docker_port_forward' ;\
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

envoy-src: FORCE
	@echo "Getting Envoy sources..."
	@if test -d envoy && ! test -d envoy-src; then PS4=; set -x; mv envoy envoy-src; fi
	@PS4=; set -x; \
	git init $@
	cd $@ && \
	if git remote get-url origin &>/dev/null; then \
		git remote set-url origin $(ENVOY_REPO); \
	else \
		git remote add origin $(ENVOY_REPO); \
	fi && \
	git fetch --tags origin && \
	$(if $(filter-out -,$(ENVOY_COMMIT)),git checkout $(ENVOY_COMMIT),if ! git rev-parse HEAD >/dev/null 2>&1; then git checkout origin/master; fi)

envoy-build-image.txt: FORCE envoy-src
	@echo "Making $@..."
	set -e; \
	cd envoy-src; . ci/envoy_build_sha.sh; cd ..; \
	echo docker.io/envoyproxy/envoy-build-ubuntu:$$ENVOY_BUILD_SHA > .tmp.$@.tmp; \
	if cmp -s .tmp.$@.tmp $@; then \
		rm -f .tmp.$@.tmp; \
	else \
		mv .tmp.$@.tmp $@; \
	fi

# We rsync to a persistent named-volume (instead of building directly
# on the bind-mount volume) because Docker for Mac's osxfs is very
# slow.
ENVOY_SYNC_HOST_TO_DOCKER = docker run --rm --volume=$(CURDIR)/envoy-src:/xfer:ro --volume=envoy-build:/root:rw $$(cat envoy-build-image.txt) rsync -Pav --delete /xfer/ /root/envoy
ENVOY_SYNC_DOCKER_TO_HOST = docker run --rm --volume=$(CURDIR)/envoy-src:/xfer:rw --volume=envoy-build:/root:ro $$(cat envoy-build-image.txt) rsync -Pav --delete /root/envoy/ /xfer

envoy-bin/envoy-static: FORCE envoy-build-image.txt
	$(ENVOY_SYNC_HOST_TO_DOCKER)
	@PS4=; set -ex; trap '$(ENVOY_SYNC_DOCKER_TO_HOST)' EXIT; { \
	    docker run --rm --volume=envoy-build:/root:rw --workdir=/root/envoy $$(cat envoy-build-image.txt) bazel build --verbose_failures -c $(ENVOY_COMPILATION_MODE) //source/exe:envoy-static; \
	    docker run --rm --volume=envoy-build:/root:rw $$(cat envoy-build-image.txt) chmod 755 /root /root/.cache; \
	}
	mkdir -p envoy-bin
	docker run --rm --volume=envoy-build:/root:ro --volume=$(CURDIR)/envoy-bin:/xfer:rw --user=$$(id -u):$$(id -g) $$(cat envoy-build-image.txt) rsync -Pav --delete /root/envoy/bazel-bin/source/exe/envoy-static /xfer/envoy-static
%-stripped: % envoy-build-image.txt
	docker run --rm --volume=$(abspath $(@D)):/xfer:rw $$(cat envoy-build-image.txt) strip /xfer/$(<F) -o /xfer/$(@F)

check-envoy: envoy-bin/envoy-static envoy-build-image.txt
	$(ENVOY_SYNC_HOST_TO_DOCKER)
	@PS4=; set -ex; trap '$(ENVOY_SYNC_DOCKER_TO_HOST)' EXIT; { \
	    docker run --rm --privileged --volume=envoy-build:/root:rw --workdir=/root/envoy $$(cat envoy-build-image.txt) bazel test --verbose_failures -c dbg --test_env=ENVOY_IP_TEST_VERSIONS=v4only //test/...; \
	}
.PHONY: check-envoy

envoy-shell: envoy-build-image.txt
	$(ENVOY_SYNC_HOST_TO_DOCKER)
	docker run --rm --privileged --volume=envoy-build:/root:rw --workdir=/root/envoy -it $$(cat envoy-build-image.txt) || true
	$(ENVOY_SYNC_DOCKER_TO_HOST)
.PHONY: envoy-shell

base-envoy.docker: Dockerfile.base-envoy $(var.)BASE_ENVOY_IMAGE $(WRITE_IFCHANGED)
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	@if ! docker run --rm --entrypoint=true $(BASE_ENVOY_IMAGE); then \
		echo "Building Envoy binary..." && \
		$(MAKE) $(ENVOY_FILE) && \
		echo "Building Envoy Docker image..." && \
		docker build --build-arg ENVOY_FILE=$(ENVOY_FILE) $(DOCKER_OPTS) -t $(BASE_ENVOY_IMAGE) -f $< .; \
	fi
	@docker image inspect $(BASE_ENVOY_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

base-py.docker: Dockerfile.base-py base-envoy.docker $(var.)BASE_PY_IMAGE $(WRITE_IFCHANGED)
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	@if ! docker run --rm --entrypoint=true $(BASE_PY_IMAGE); then \
		echo "Building $(BASE_PY_IMAGE)" && \
		docker build --build-arg BASE_ENVOY_IMAGE=$(BASE_ENVOY_IMAGE) $(DOCKER_OPTS) -t $(BASE_PY_IMAGE) -f $< .; \
	fi
	@docker image inspect $(BASE_PY_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

base-go.docker: Dockerfile.base-go base-envoy.docker $(var.)BASE_GO_IMAGE $(WRITE_IFCHANGED)
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	@if ! docker run --rm --entrypoint=true $(BASE_GO_IMAGE); then \
		echo "Building $(BASE_GO_IMAGE)" && \
		docker build --build-arg BASE_ENVOY_IMAGE=$(BASE_ENVOY_IMAGE) $(DOCKER_OPTS) -t $(BASE_GO_IMAGE) -f $< .; \
	fi
	@docker image inspect $(BASE_GO_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

docker-base-images:
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	$(MAKE) base-envoy.docker base-go.docker base-py.docker
	@echo "RESTART ANY DEV SHELLS to make sure they use your new images."

docker-push-base-images:
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	docker push $(BASE_ENVOY_IMAGE)
	docker push $(BASE_PY_IMAGE)
	docker push $(BASE_GO_IMAGE)
	@echo "RESTART ANY DEV SHELLS to make sure they use your new images."

docker-update-base:
	$(MAKE) docker-base-images go/apis/envoy
	$(MAKE) docker-push-base-images

ambassador-docker-image: ambassador.docker
ambassador.docker: Dockerfile base-go.docker base-py.docker $(WATT) $(WRITE_IFCHANGED) ambassador/ambassador/VERSION.py FORCE
	docker build --build-arg BASE_GO_IMAGE=$(BASE_GO_IMAGE) --build-arg BASE_PY_IMAGE=$(BASE_PY_IMAGE) $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) .
	@docker image inspect $(AMBASSADOR_DOCKER_IMAGE) --format='{{.Id}}' | $(WRITE_IFCHANGED) $@

docker-images: ambassador-docker-image

docker-push: docker-images
ifeq ($(DOCKER_PUSH_AS),)
	@echo "No DOCKER_PUSH_AS set"
else
	@if [ "$(GIT_DIRTY)" = "dirty" ]; then \
		printf "Git tree is dirty and therefore 'docker push' is not allowed!\n"; \
		exit 1; \
	fi
	@echo 'PUSH $(AMBASSADOR_DOCKER_IMAGE) as $(DOCKER_PUSH_AS)'
ifneq ($(DOCKER_PUSH_AS),$(AMBASSADOR_DOCKER_IMAGE))
	@docker tag $(AMBASSADOR_DOCKER_IMAGE) $(DOCKER_PUSH_AS)
endif
	@docker push $(DOCKER_PUSH_AS) | python releng/linify.py push.log
endif

# TODO: validate version is conformant to some set of rules might be a good idea to add here
ambassador/ambassador/VERSION.py: FORCE $(WRITE_IFCHANGED)
	$(call check_defined, VERSION, VERSION is not set)
	$(call check_defined, GIT_BRANCH, GIT_BRANCH is not set)
	$(call check_defined, GIT_COMMIT, GIT_COMMIT is not set)
	$(call check_defined, GIT_DESCRIPTION, GIT_DESCRIPTION is not set)
	@echo "Generating and templating version information -> $(VERSION)"
	sed \
		-e 's!{{VERSION}}!$(VERSION)!g' \
		-e 's!{{GITBRANCH}}!$(GIT_BRANCH)!g' \
		-e 's!{{GITDIRTY}}!$(GIT_DIRTY)!g' \
		-e 's!{{GITCOMMIT}}!$(GIT_COMMIT)!g' \
		-e 's!{{GITDESCRIPTION}}!$(GIT_DESCRIPTION)!g' \
		< VERSION-template.py | $(WRITE_IFCHANGED) $@

version: ambassador/ambassador/VERSION.py

TELEPROXY=venv/bin/teleproxy
TELEPROXY_VERSION=0.4.11

# This should maybe be replaced with a lighterweight dependency if we
# don't currently depend on go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(TELEPROXY): $(var.)TELEPROXY_VERSION $(var.)GOOS $(var.)GOARCH | venv/bin/activate
	curl -o $(TELEPROXY) https://s3.amazonaws.com/datawire-static-files/teleproxy/$(TELEPROXY_VERSION)/$(GOOS)/$(GOARCH)/teleproxy
	sudo chown 0:0 $(TELEPROXY)			# setting group 0 is very important for SUID on MacOS!	
	sudo chmod go-w,a+sx $(TELEPROXY)

kill_teleproxy = curl -s --connect-timeout 5 127.254.254.254/api/shutdown || true
run_teleproxy = $(TELEPROXY)

# This is for the docker image, so we don't use the current arch, we hardcode to linux/amd64
$(WATT): $(var.)WATT_VERSION
	curl -o $(WATT) https://s3.amazonaws.com/datawire-static-files/watt/$(WATT_VERSION)/linux/amd64/watt
	chmod go-w,a+x $(WATT)

CLAIM_FILE=kubernaut-claim.txt
CLAIM_NAME=$(shell cat $(CLAIM_FILE))

KUBERNAUT=venv/bin/kubernaut
KUBERNAUT_VERSION=2018.10.24-d46c1f1
KUBERNAUT_CLAIM=$(KUBERNAUT) claims create --name $(CLAIM_NAME) --cluster-group main
KUBERNAUT_DISCARD=$(KUBERNAUT) claims delete $(CLAIM_NAME)

$(CLAIM_FILE):
	@if [ -z $${CI+x} ]; then \
		echo kat-$${USER} > $@; \
	else \
		echo kat-$${USER}-$(shell uuidgen) > $@; \
	fi

$(KUBERNAUT): $(var.)KUBERNAUT_VERSION $(var.)GOOS $(var.)GOARCH | venv/bin/activate
	curl -o $(KUBERNAUT) http://releases.datawire.io/kubernaut/$(KUBERNAUT_VERSION)/$(GOOS)/$(GOARCH)/kubernaut
	chmod +x $(KUBERNAUT)

KAT_CLIENT=venv/bin/kat_client

venv/kat-backend-$(KAT_BACKEND_RELEASE).tar.gz: | venv/bin/activate
	curl -L -o $@ https://github.com/datawire/kat-backend/archive/v$(KAT_BACKEND_RELEASE).tar.gz
$(KAT_CLIENT): venv/kat-backend-$(KAT_BACKEND_RELEASE).tar.gz
	cd venv && tar -xzf $(<F) kat-backend-$(KAT_BACKEND_RELEASE)/client/bin/client_$(GOOS)_$(GOARCH)
	install -m0755 venv/kat-backend-$(KAT_BACKEND_RELEASE)/client/bin/client_$(GOOS)_$(GOARCH) $(CURDIR)/$(KAT_CLIENT)

setup-develop: venv $(KAT_CLIENT) $(TELEPROXY) $(KUBERNAUT) $(WATT) version

cluster.yaml: $(CLAIM_FILE)
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

setup-test: cluster-and-teleproxy

cluster-and-teleproxy: cluster.yaml $(TELEPROXY)
	rm -rf /tmp/k8s-*.yaml
	$(MAKE) teleproxy-restart
	@echo "Sleeping for Teleproxy cluster"
	sleep 10

teleproxy-restart: $(TELEPROXY)
	@echo "Killing teleproxy"
	$(kill_teleproxy)
	sleep 0.25 # wait for exit...
	$(run_teleproxy) -kubeconfig $(KUBECONFIG) 2> /tmp/teleproxy.log &
	sleep 0.5 # wait for start
	@if [ $$(ps -ef | grep venv/bin/teleproxy | grep -v grep | wc -l) -le 0 ]; then \
		echo "teleproxy did not start"; \
		cat /tmp/teleproxy.log; \
		exit 1; \
	fi
	@echo "Done"

teleproxy-stop:
	$(kill_teleproxy)
	sleep 0.25 # wait for exit...
	@if [ $$(ps -ef | grep venv/bin/teleproxy | grep -v grep | wc -l) -gt 0 ]; then \
		echo "teleproxy still running" >&2; \
		ps -ef | grep venv/bin/teleproxy | grep -v grep >&2; \
		false; \
	else \
		echo "teleproxy stopped" >&2; \
	fi

# "make shell" drops you into a dev shell, and tries to set variables, etc., as
# needed:
#
# If USE_KUBERNAUT is true, we'll set up for Kubernaut, otherwise we'll assume 
# that the current KUBECONFIG is good.

shell: setup-develop
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	BASE_PY_IMAGE="$(BASE_PY_IMAGE)" \
	BASE_GO_IMAGE="$(BASE_GO_IMAGE)" \
	MAKE_KUBECONFIG="$(KUBECONFIG)" \
	AMBASSADOR_DEV=1 \
	bash --init-file releng/init.sh -i

clean-test:
	rm -f cluster.yaml
	test -x $(KUBERNAUT) && $(KUBERNAUT_DISCARD) || true
	rm -f $(CLAIM_FILE)
	$(call kill_teleproxy)

test: setup-develop
	cd ambassador && \
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	BASE_PY_IMAGE="$(BASE_PY_IMAGE)" \
	BASE_GO_IMAGE="$(BASE_GO_IMAGE)" \
	KUBECONFIG="$(KUBECONFIG)" \
	PATH="$(shell pwd)/venv/bin:$(PATH)" \
	bash ../releng/run-tests.sh

test-list: setup-develop
	cd ambassador && PATH="$(shell pwd)/venv/bin":$(PATH) pytest --collect-only -q

update-aws:
ifeq ($(ASW_ACCESS_KEY_ID),)
	@echo 'AWS credentials not configured; not updating https://s3.amazonaws.com/datawire-static-files/ambassador/$(STABLE_TXT_KEY)'
	@echo 'AWS credentials not configured; not updating latest version in Scout'
else
	@if [ -n "$(STABLE_TXT_KEY)" ]; then \
        printf "$(VERSION)" > stable.txt; \
		echo "updating $(STABLE_TXT_KEY) with $$(cat stable.txt)"; \
        aws s3api put-object \
            --bucket datawire-static-files \
            --key ambassador/$(STABLE_TXT_KEY) \
            --body stable.txt; \
	fi
	@if [ -n "$(SCOUT_APP_KEY)" ]; then \
		printf '{"application":"ambassador","latest_version":"$(VERSION)","notices":[]}' > app.json; \
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
	@if [ "$(VERSION)" = "$(GIT_VERSION)" ]; then \
		printf "'make release' can only be run for a GA commit when VERSION is not the same as GIT_COMMIT!\n"; \
		exit 1; \
	fi
	docker pull $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC)
	docker tag $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC) $(AMBASSADOR_DOCKER_REPO):$(VERSION)
	docker push $(AMBASSADOR_DOCKER_REPO):$(VERSION)
	$(MAKE) SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws

# ------------------------------------------------------------------------------
# Go gRPC bindings
# ------------------------------------------------------------------------------

# The version numbers of `protoc` (in this Makefile),
# `protoc-gen-gogofast` (in go.mod), and `protoc-gen-validate` (in
# go.mod) are based on
# https://github.com/envoyproxy/go-control-plane/blob/master/Dockerfile.ci

PROTOC_VERSION = 3.5.1
PROTOC_PLATFORM = $(patsubst darwin,osx,$(GOOS))-$(patsubst amd64,x86_64,$(patsubst 386,x86_32,$(GOARCH)))

venv/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip: $(var.)PROTOC_VERSION | venv/bin/activate
	curl -o $@ --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(@F)
venv/bin/protoc: venv/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip
	bsdtar -xf $< -C venv bin/protoc

venv/bin/protoc-gen-gogofast: go.mod $(FLOCK) | venv/bin/activate
	$(FLOCK) go.mod go build -o $@ github.com/gogo/protobuf/protoc-gen-gogofast

venv/bin/protoc-gen-validate: go.mod $(FLOCK) | venv/bin/activate
	$(FLOCK) go.mod go build -o $@ github.com/envoyproxy/protoc-gen-validate

# Search path for .proto files
gomoddir = $(shell $(FLOCK) go.mod go list $1/... >/dev/null 2>/dev/null; $(FLOCK) go.mod go list -m -f='{{.Dir}}' $1)
# This list is based 'imports=()' in https://github.com/envoyproxy/go-control-plane/blob/master/build/generate_protos.sh
imports += $(CURDIR)/envoy-src/api
imports += $(call gomoddir,github.com/envoyproxy/protoc-gen-validate)
imports += $(call gomoddir,github.com/gogo/googleapis)
imports += $(call gomoddir,github.com/gogo/protobuf)/protobuf
imports += $(call gomoddir,istio.io/gogo-genproto)
imports += $(call gomoddir,istio.io/gogo-genproto)/prometheus

# Map from .proto files to Go package names
# This list is based 'mappings=()' in https://github.com/envoyproxy/go-control-plane/blob/master/build/generate_protos.sh
mappings += gogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto
mappings += google/api/annotations.proto=github.com/gogo/googleapis/google/api
mappings += google/protobuf/any.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/duration.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/empty.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/struct.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/timestamp.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/wrappers.proto=github.com/gogo/protobuf/types
mappings += google/rpc/status.proto=github.com/gogo/googleapis/google/rpc
mappings += metrics.proto=istio.io/gogo-genproto/prometheus
mappings += opencensus/proto/trace/v1/trace.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
mappings += opencensus/proto/trace/v1/trace_config.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
mappings += $(shell find $(CURDIR)/envoy-src/api/envoy -type f -name '*.proto' | sed -E 's,^$(CURDIR)/envoy-src/api/((.*)/[^/]*),\1=github.com/datawire/ambassador/go/apis/\2,')

joinlist=$(if $(word 2,$2),$(firstword $2)$1$(call joinlist,$1,$(wordlist 2,$(words $2),$2)),$2)
comma = ,

_imports = $(call lazyonce,_imports,$(imports))
_mappings = $(call lazyonce,_mappings,$(mappings))
go/apis/envoy: envoy-src $(FLOCK) venv/bin/protoc venv/bin/protoc-gen-gogofast venv/bin/protoc-gen-validate $(var.)_imports $(var.)_mappings
	rm -rf $@ $(@D).envoy.tmp
	mkdir -p $(@D).envoy.tmp
# go-control-plane `make generate`
	@set -e; find $(CURDIR)/envoy-src/api/envoy -type f -name '*.proto' | sed 's,/[^/]*$$,,' | uniq | while read -r dir; do \
		echo "Generating $$dir"; \
		./venv/bin/protoc \
			$(addprefix --proto_path=,$(_imports))  \
			--plugin=$(CURDIR)/venv/bin/protoc-gen-gogofast --gogofast_out='$(call joinlist,$(comma),plugins=grpc $(addprefix M,$(_mappings))):$(@D).envoy.tmp' \
			--plugin=$(CURDIR)/venv/bin/protoc-gen-validate --validate_out='lang=gogo:$(@D).envoy.tmp' \
			"$$dir"/*.proto; \
	done
# go-control-plane `make generate-patch`
# https://github.com/envoyproxy/go-control-plane/issues/173
	find $(@D).envoy.tmp -name '*.validate.go' -exec sed -E -i.bak 's,"(envoy/.*)"$$,"github.com/datawire/ambassador/go/apis/\1",' {} +
	find $(@D).envoy.tmp -name '*.bak' -delete
# move things in to place
	mkdir -p $(@D)
	mv $(@D).envoy.tmp/envoy $@
	rmdir $(@D).envoy.tmp

# ------------------------------------------------------------------------------
# Virtualenv
# ------------------------------------------------------------------------------

venv: version venv/bin/ambassador

venv/bin/ambassador: venv/bin/activate ambassador/requirements.txt
	@releng/install-py.sh dev requirements ambassador/requirements.txt
	@releng/install-py.sh dev install ambassador/requirements.txt
	@releng/fix_kube_client

venv/bin/activate: dev-requirements.txt multi/requirements.txt kat/requirements.txt
	test -d venv || virtualenv venv --python python3
	@releng/install-py.sh dev requirements $^
	@releng/install-py.sh dev install $^
	touch venv/bin/activate
	@releng/fix_kube_client

mypy-server-stop:
	dmypy stop

mypy-server:
	@if ! dmypy status >/dev/null; then \
		dmypy start -- --use-fine-grained-cache --follow-imports=skip --ignore-missing-imports ;\
		echo "Started mypy server" ;\
	fi

mypy: mypy-server
	time dmypy check ambassador

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

pull-docs:
	{ \
		git fetch https://github.com/datawire/ambassador-docs $(PULL_BRANCH) && \
		docs_head=$$(git rev-parse FETCH_HEAD) && \
		git subtree merge --prefix=docs "$${docs_head}" && \
		git subtree split --prefix=docs --rejoin --onto="$${docs_head}"; \
	}
push-docs:
	{ \
		git fetch https://github.com/datawire/ambassador-docs master && \
		docs_old=$$(git rev-parse FETCH_HEAD) && \
		docs_new=$$(git subtree split --prefix=docs --rejoin --onto="$${docs_old}") && \
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
