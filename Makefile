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

SHELL = bash

# Welcome to the Ambassador Makefile...

.FORCE:
.PHONY: \
    .FORCE clean version setup-develop print-vars \
    docker-login docker-push docker-images publish-website \
    teleproxy-restart teleproxy-stop

# MAIN_BRANCH
# -----------
#
# The name of the main branch (e.g. "stable"). This is set as an variable because it makes it easy to develop and test
# new automation code on a branch that is simulating the purpose of the main branch.
#
MAIN_BRANCH ?= stable

# GIT_BRANCH on TravisCI needs to be set through some external custom logic. Default to a Git native mechanism or
# use what is defined.
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_DIRTY ?= $(shell test -z "$(shell git status --porcelain)" || printf "dirty")

ifndef $(GIT_BRANCH)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
endif

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

# TODO: need to remove the dependency on Travis env var which means this likely needs to be arg passed to make rather
IS_PULL_REQUEST = false
ifdef TRAVIS_PULL_REQUEST
ifneq ($(TRAVIS_PULL_REQUEST),false)
IS_PULL_REQUEST = true
endif
endif

# Note that for everything except RC builds, VERSION will be set to the version
# we'd use for a GA build. This is by design.
ifneq ($(GIT_TAG_SANITIZED),)
VERSION = $(shell printf "$(GIT_TAG_SANITIZED)" | sed -e 's/-.*//g')
else
VERSION = $(GIT_VERSION)
endif

# We need this for tagging in some situations.
LATEST_RC=$(VERSION)-rc-latest

# Is this a random commit, an RC, or a GA?
ifeq ($(shell [[ "$(GIT_BRANCH)" =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]] && echo "GA"), GA)
COMMIT_TYPE=GA
else ifeq ($(shell [[ "$(GIT_BRANCH)" =~ -rc[0-9]+$$ ]] && echo "RC"), RC)
COMMIT_TYPE=RC
else ifeq ($(shell [[ "$(GIT_BRANCH)" =~ -ea[0-9]+$$ ]] && echo "EA"), EA)
COMMIT_TYPE=EA
else ifeq ($(IS_PULL_REQUEST), true)
COMMIT_TYPE=PR
else
COMMIT_TYPE=random
endif

DOC_RELEASE_TYPE?=unstable

ifndef DOCKER_REGISTRY
$(error DOCKER_REGISTRY must be set. Use make DOCKER_REGISTRY=- for a purely local build.)
endif

ifeq ($(DOCKER_REGISTRY), -)
AMBASSADOR_DOCKER_REPO ?= ambassador
else
AMBASSADOR_DOCKER_REPO ?= $(DOCKER_REGISTRY)/ambassador
endif

DOCKER_OPTS =

NETLIFY_SITE=datawire-ambassador

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST UPDATE THE VERSION NUMBERS
# BELOW AND THEN RUN make docker-update-base
ENVOY_BASE_IMAGE ?= quay.io/datawire/ambassador-envoy-alpine-stripped:v1.8.0-15c5befd43fb9ee9b145cc87e507beb801726316-15-ga0d95bafb
AMBASSADOR_DOCKER_TAG ?= $(GIT_VERSION)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

# UPDATE THESE VERSION NUMBERS IF YOU UPDATE ANY OF THE VALUES ABOVE, THEN 
# RUN make docker-update-base.
AMBASSADOR_DOCKER_IMAGE_CACHED ?= quay.io/datawire/ambassador-base:go-6
AMBASSADOR_BASE_IMAGE ?= quay.io/datawire/ambassador-base:ambassador-6

SCOUT_APP_KEY=

# Sets the kat-backend release which contains the kat-client use for E2e testing.
# For details https://github.com/datawire/kat-backend
KAT_BACKEND_RELEASE = 1.1.0

# "make" by itself doesn't make the website. It takes too long and it doesn't
# belong in the inner dev loop.
all: setup-develop docker-push test

clean: clean-test
	rm -rf docs/_book docs/_site docs/package-lock.json
	rm -rf helm/*.tgz
	rm -rf app.json
	rm -rf venv/bin/ambassador
	rm -rf ambassador/ambassador/VERSION.py*
	rm -rf ambassador/build ambassador/dist ambassador/ambassador.egg-info ambassador/__pycache__
	find . \( -name .coverage -o -name .cache -o -name __pycache__ \) -print0 | xargs -0 rm -rf
	find . \( -name *.log \) -print0 | xargs -0 rm -rf
	find ambassador/tests \
		\( -name '*.out' -o -name 'envoy.json' -o -name 'intermediate.json' \) -print0 \
		| xargs -0 rm -f
	rm -rf end-to-end/ambassador-deployment.yaml end-to-end/ambassador-deployment-mounts.yaml
	find end-to-end \( -name 'check-*.json' -o -name 'envoy.json' \) -print0 | xargs -0 rm -f
	rm -f end-to-end/1-parallel/011-websocket/ambassador/service.yaml
	find end-to-end/1-parallel/ -type f -name 'ambassador-deployment*.yaml' -exec rm {} \;

clobber: clean
	-rm -rf docs/node_modules
	-rm -rf kat/kat/client
	-rm -rf venv && echo && echo "Deleted venv, run 'deactivate' command if your virtualenv is activated" || true

print-%:
	@printf "$($*)"

print-vars:
	@echo "MAIN_BRANCH             = $(MAIN_BRANCH)"
	@echo "GIT_BRANCH              = $(GIT_BRANCH)"
	@echo "GIT_BRANCH_SANITIZED    = $(GIT_BRANCH_SANITIZED)"
	@echo "GIT_COMMIT              = $(GIT_COMMIT)"
	@echo "GIT_DIRTY               = $(GIT_DIRTY)"
	@echo "GIT_TAG                 = $(GIT_TAG)"
	@echo "GIT_TAG_SANITIZED       = $(GIT_TAG_SANITIZED)"
	@echo "GIT_VERSION             = $(GIT_VERSION)"
	@echo "GIT_DESCRIPTION         = $(GIT_DESCRIPTION)"
	@echo "IS_PULL_REQUEST         = $(IS_PULL_REQUEST)"
	@echo "COMMIT_TYPE             = $(COMMIT_TYPE)"
	@echo "VERSION                 = $(VERSION)"
	@echo "LATEST_RC               = $(LATEST_RC)"
	@echo "DOCKER_REGISTRY         = $(DOCKER_REGISTRY)"
	@echo "DOCKER_OPTS             = $(DOCKER_OPTS)"
	@echo "AMBASSADOR_DOCKER_REPO  = $(AMBASSADOR_DOCKER_REPO)"
	@echo "AMBASSADOR_DOCKER_TAG   = $(AMBASSADOR_DOCKER_TAG)"
	@echo "AMBASSADOR_DOCKER_IMAGE = $(AMBASSADOR_DOCKER_IMAGE)"
	@echo "KAT_BACKEND_RELEASE = $(KAT_BACKEND_RELEASE)"

export-vars:
	@echo "export MAIN_BRANCH='$(MAIN_BRANCH)'"
	@echo "export GIT_BRANCH='$(GIT_BRANCH)'"
	@echo "export GIT_BRANCH_SANITIZED='$(GIT_BRANCH_SANITIZED)'"
	@echo "export GIT_COMMIT='$(GIT_COMMIT)'"
	@echo "export GIT_DIRTY='$(GIT_DIRTY)'"
	@echo "export GIT_TAG='$(GIT_TAG)'"
	@echo "export GIT_TAG_SANITIZED='$(GIT_TAG_SANITIZED)'"
	@echo "export GIT_VERSION='$(GIT_VERSION)'"
	@echo "export GIT_DESCRIPTION='$(GIT_DESCRIPTION)'"
	@echo "export IS_PULL_REQUEST='$(IS_PULL_REQUEST)'"
	@echo "export COMMIT_TYPE='$(COMMIT_TYPE)'"
	@echo "export VERSION='$(VERSION)'"
	@echo "export LATEST_RC='$(LATEST_RC)'"
	@echo "export DOCKER_REGISTRY='$(DOCKER_REGISTRY)'"
	@echo "export DOCKER_OPTS='$(DOCKER_OPTS)'"
	@echo "export AMBASSADOR_DOCKER_REPO='$(AMBASSADOR_DOCKER_REPO)'"
	@echo "export AMBASSADOR_DOCKER_TAG='$(AMBASSADOR_DOCKER_TAG)'"
	@echo "export AMBASSADOR_DOCKER_IMAGE='$(AMBASSADOR_DOCKER_IMAGE)'"
	@echo "export KAT_BACKEND_RELEASE='$(KAT_BACKEND_RELEASE)'"

docker-base-images:
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	docker build --build-arg ENVOY_BASE_IMAGE=$(ENVOY_BASE_IMAGE) $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE_CACHED) -f Dockerfile.cached .
	docker build --build-arg ENVOY_BASE_IMAGE=$(ENVOY_BASE_IMAGE) $(DOCKER_OPTS) -t $(AMBASSADOR_BASE_IMAGE) -f Dockerfile.ambassador .
	@echo "RESTART ANY DEV SHELLS to make sure they use your new images."

docker-push-base-images:
	@if [ -n "$(AMBASSADOR_DEV)" ]; then echo "Do not run this from a dev shell" >&2; exit 1; fi
	docker push $(AMBASSADOR_DOCKER_IMAGE_CACHED)
	docker push $(AMBASSADOR_BASE_IMAGE)
	@echo "RESTART ANY DEV SHELLS to make sure they use your new images."

docker-update-base: docker-base-images docker-push-base-images

ambassador-docker-image: version
	docker build --build-arg AMBASSADOR_BASE_IMAGE=$(AMBASSADOR_BASE_IMAGE) --build-arg CACHED_CONTAINER_IMAGE=$(AMBASSADOR_DOCKER_IMAGE_CACHED) $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) .

docker-login:
	@if [ -z $(DOCKER_USERNAME) ]; then echo 'DOCKER_USERNAME not defined'; exit 1; fi
	@if [ -z $(DOCKER_PASSWORD) ]; then echo 'DOCKER_PASSWORD not defined'; exit 1; fi

	@printf "$(DOCKER_PASSWORD)" | docker login -u="$(DOCKER_USERNAME)" --password-stdin $(DOCKER_REGISTRY)

docker-images: ambassador-docker-image

docker-push: docker-images
ifneq ($(DOCKER_REGISTRY), -)
	@if [ \( "$(GIT_DIRTY)" != "dirty" \) -o \( "$(GIT_BRANCH)" != "$(MAIN_BRANCH)" \) ]; then \
		echo "PUSH $(AMBASSADOR_DOCKER_IMAGE)"; \
		docker push $(AMBASSADOR_DOCKER_IMAGE) | python end-to-end/linify.py push.log; \
		if [ \( "$(COMMIT_TYPE)" = "RC" \) -o \( "$(COMMIT_TYPE)" = "EA" \) ]; then \
			echo "PUSH $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED)"; \
			docker tag $(AMBASSADOR_DOCKER_IMAGE) $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED); \
			docker push $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED) | python end-to-end/linify.py push.log; \
		fi; \
		if [ "$(COMMIT_TYPE)" = "RC" ]; then \
			echo "PUSH $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC)"; \
			docker tag $(AMBASSADOR_DOCKER_IMAGE) $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC); \
			docker push $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC) | python end-to-end/linify.py push.log; \
		fi; \
	else \
		printf "Git tree on MAIN_BRANCH '$(MAIN_BRANCH)' is dirty and therefore 'docker push' is not allowed!\n"; \
		exit 1; \
	fi
endif

# TODO: validate version is conformant to some set of rules might be a good idea to add here
ambassador/ambassador/VERSION.py:
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
		< VERSION-template.py > ambassador/ambassador/VERSION.py

version: ambassador/ambassador/VERSION.py

e2e-versioned-manifests: venv
	cd end-to-end && PATH="$(shell pwd)/venv/bin:$(PATH)" bash create-manifests.sh $(AMBASSADOR_DOCKER_IMAGE)

e2e: E2E_TEST_NAME=all
e2e: e2e-versioned-manifests
	source venv/bin/activate; \
	if [ "$(E2E_TEST_NAME)" == "all" ]; then \
		bash end-to-end/testall.sh; \
	else \
		E2E_TEST_NAME=$(E2E_TEST_NAME) bash end-to-end/testall.sh; \
	fi

TELEPROXY=venv/bin/teleproxy
TELEPROXY_VERSION=0.3.16

# This should maybe be replaced with a lighterweight dependency if we
# don't currently depend on go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(TELEPROXY):
	curl -o $(TELEPROXY) https://s3.amazonaws.com/datawire-static-files/teleproxy/$(TELEPROXY_VERSION)/$(GOOS)/$(GOARCH)/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod go-w,a+sx $(TELEPROXY)

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

$(KUBERNAUT):
	curl -o $(KUBERNAUT) http://releases.datawire.io/kubernaut/$(KUBERNAUT_VERSION)/$(GOOS)/$(GOARCH)/kubernaut
	chmod +x $(KUBERNAUT)

setup-develop: venv $(PWD)/kat/kat/client $(TELEPROXY) $(KUBERNAUT) version

$(PWD)/kat/kat/client:
	curl -OL https://github.com/datawire/kat-backend/archive/v$(KAT_BACKEND_RELEASE).tar.gz 
	tar xzf v$(KAT_BACKEND_RELEASE).tar.gz
	chmod +x kat-backend-$(KAT_BACKEND_RELEASE)/client/bin/client_$(GOOS)_$(GOARCH)
	mv kat-backend-$(KAT_BACKEND_RELEASE)/client/bin/client_$(GOOS)_$(GOARCH) $(PWD)/kat/kat/client
	rm -rf v$(KAT_BACKEND_RELEASE).tar.gz kat-backend-$(KAT_BACKEND_RELEASE)/

kill_teleproxy = $(shell kill -INT $$(/bin/ps -ef | fgrep venv/bin/teleproxy | fgrep -v grep | awk '{ print $$2 }') 2>/dev/null)

cluster.yaml: $(CLAIM_FILE)
	$(KUBERNAUT_DISCARD)
	$(KUBERNAUT_CLAIM)
	cp ~/.kube/$(CLAIM_NAME).yaml cluster.yaml
	rm -rf /tmp/k8s-*.yaml
	$(call kill_teleproxy)
	$(TELEPROXY) -kubeconfig "$(shell pwd)/cluster.yaml" 2> /tmp/teleproxy.log &
	@echo "Sleeping for Teleproxy cluster"
	sleep 10

setup-test: cluster.yaml

teleproxy-restart:
	$(call kill_teleproxy)
	sleep 0.25 # wait for exit...
	$(TELEPROXY) -kubeconfig "$(shell pwd)/cluster.yaml" 2> /tmp/teleproxy.log &

teleproxy-stop:
	$(call kill_teleproxy)
	sleep 0.25 # wait for exit...
	@if [ $$(ps -ef | grep venv/bin/teleproxy | grep -v grep | wc -l) -gt 0 ]; then \
		echo "teleproxy still running" >&2; \
		ps -ef | grep venv/bin/teleproxy | grep -v grep >&2; \
		false; \
	else \
		echo "teleproxy stopped" >&2; \
	fi

KUBECONFIG=$(shell pwd)/cluster.yaml

shell: setup-develop cluster.yaml
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	AMBASSADOR_DOCKER_IMAGE_CACHED="$(AMBASSADOR_DOCKER_IMAGE_CACHED)" \
	AMBASSADOR_BASE_IMAGE="$(AMBASSADOR_BASE_IMAGE)" \
	KUBECONFIG="$(KUBECONFIG)" \
	AMBASSADOR_DEV=1 \
	bash --init-file releng/init.sh -i

clean-test:
	rm -f cluster.yaml
	test -x $(KUBERNAUT) && $(KUBERNAUT_DISCARD) || true
	rm -f $(CLAIM_FILE)
	$(call kill_teleproxy)

test: setup-develop cluster.yaml
	cd ambassador && \
	AMBASSADOR_DOCKER_IMAGE="$(AMBASSADOR_DOCKER_IMAGE)" \
	AMBASSADOR_DOCKER_IMAGE_CACHED="$(AMBASSADOR_DOCKER_IMAGE_CACHED)" \
	AMBASSADOR_BASE_IMAGE="$(AMBASSADOR_BASE_IMAGE)" \
	KUBECONFIG="$(KUBECONFIG)" \
	PATH="$(shell pwd)/venv/bin:$(PATH)" \
	sh ../releng/run-tests.sh

test-list: setup-develop
	cd ambassador && PATH="$(shell pwd)/venv/bin":$(PATH) pytest --collect-only -q

update-aws:
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

release-prep:
	bash releng/release-prep.sh

release:
	@if [ "$(COMMIT_TYPE)" = "GA" -a "$(VERSION)" != "$(GIT_VERSION)" ]; then \
		set -ex; \
		docker pull $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC); \
		docker tag $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC) $(AMBASSADOR_DOCKER_REPO):$(VERSION); \
		docker push $(AMBASSADOR_DOCKER_REPO):$(VERSION); \
		DOC_RELEASE_TYPE=stable make website; \
		make SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws; \
		set +x; \
	else \
		printf "'make release' can only be run for a GA commit when VERSION is not the same as GIT_COMMIT!\n"; \
		exit 1; \
	fi

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

publish-website:
	bash ./releng/publish-website.sh;

# ------------------------------------------------------------------------------
# CI Targets
# ------------------------------------------------------------------------------

ci-docker: docker-push

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
