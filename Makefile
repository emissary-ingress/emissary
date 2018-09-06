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
.PHONY: .FORCE clean version setup-develop print-vars docker-login docker-push docker-images publish-website helm

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

AMBASSADOR_DOCKER_TAG ?= $(GIT_VERSION)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

SCOUT_APP_KEY=

all: test docker-push website

clean: clean-test
	rm -rf docs/yaml docs/_book docs/_site docs/package-lock.json
	rm -rf helm/*.tgz
	rm -rf app.json
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
	-rm -r venv 2> /dev/null && echo && echo "Deleted venv, run 'deactivate' command if your virtualenv is activated" || true

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
	@echo "IS_PULL_REQUEST         = $(IS_PULL_REQUEST)"
	@echo "COMMIT_TYPE             = $(COMMIT_TYPE)"
	@echo "VERSION                 = $(VERSION)"
	@echo "LATEST_RC               = $(LATEST_RC)"
	@echo "DOCKER_REGISTRY         = $(DOCKER_REGISTRY)"
	@echo "DOCKER_OPTS             = $(DOCKER_OPTS)"
	@echo "AMBASSADOR_DOCKER_REPO  = $(AMBASSADOR_DOCKER_REPO)"
	@echo "AMBASSADOR_DOCKER_TAG   = $(AMBASSADOR_DOCKER_TAG)"
	@echo "AMBASSADOR_DOCKER_IMAGE = $(AMBASSADOR_DOCKER_IMAGE)"

export-vars:
	@echo "export MAIN_BRANCH='$(MAIN_BRANCH)'"
	@echo "export GIT_BRANCH='$(GIT_BRANCH)'"
	@echo "export GIT_BRANCH_SANITIZED='$(GIT_BRANCH_SANITIZED)'"
	@echo "export GIT_COMMIT='$(GIT_COMMIT)'"
	@echo "export GIT_DIRTY='$(GIT_DIRTY)'"
	@echo "export GIT_TAG='$(GIT_TAG)'"
	@echo "export GIT_TAG_SANITIZED='$(GIT_TAG_SANITIZED)'"
	@echo "export GIT_VERSION='$(GIT_VERSION)'"
	@echo "export IS_PULL_REQUEST='$(IS_PULL_REQUEST)'"
	@echo "export COMMIT_TYPE='$(COMMIT_TYPE)'"
	@echo "export VERSION='$(VERSION)'"
	@echo "export LATEST_RC='$(LATEST_RC)'"
	@echo "export DOCKER_REGISTRY='$(DOCKER_REGISTRY)'"
	@echo "export DOCKER_OPTS='$(DOCKER_OPTS)'"
	@echo "export AMBASSADOR_DOCKER_REPO='$(AMBASSADOR_DOCKER_REPO)'"
	@echo "export AMBASSADOR_DOCKER_TAG='$(AMBASSADOR_DOCKER_TAG)'"
	@echo "export AMBASSADOR_DOCKER_IMAGE='$(AMBASSADOR_DOCKER_IMAGE)'"

ambassador-docker-image:
	docker build -q $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) ./ambassador

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
		if [ "$(COMMIT_TYPE)" = "RC" ]; then \
			echo "PUSH $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED)"; \
			docker tag $(AMBASSADOR_DOCKER_IMAGE) $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED); \
			docker push $(AMBASSADOR_DOCKER_REPO):$(GIT_TAG_SANITIZED) | python end-to-end/linify.py push.log; \
			echo "PUSH $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC)"; \
			docker tag $(AMBASSADOR_DOCKER_IMAGE) $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC); \
			docker push $(AMBASSADOR_DOCKER_REPO):$(LATEST_RC) | python end-to-end/linify.py push.log; \
		fi; \
	else \
		printf "Git tree on MAIN_BRANCH '$(MAIN_BRANCH)' is dirty and therefore 'docker push' is not allowed!\n"; \
		exit 1; \
	fi
endif

ambassador/ambassador/VERSION.py:
	# TODO: validate version is conformant to some set of rules might be a good idea to add here
	$(call check_defined, VERSION, VERSION is not set)
	@echo "Generating and templating version information -> $(VERSION)"
	sed -e "s/{{VERSION}}/$(VERSION)/g" < VERSION-template.py > ambassador/ambassador/VERSION.py

version: ambassador/ambassador/VERSION.py

e2e-versioned-manifests: venv website-yaml
	cd end-to-end && PATH=$(shell pwd)/venv/bin:$(PATH) bash create-manifests.sh $(AMBASSADOR_DOCKER_IMAGE)

website-yaml:
	mkdir -p docs/yaml
	cp -R templates/* docs/yaml
	find ./docs/yaml \
		-type f \
		-exec sed \
			-i''\
			-e "s|{{AMBASSADOR_DOCKER_IMAGE}}|$(AMBASSADOR_DOCKER_REPO):$(VERSION)|g" \
			{} \;

website: website-yaml
	VERSION=$(VERSION) bash docs/build-website.sh

helm:
	echo "Helm version $(VERSION)"
	cd helm && helm package --app-version "$(VERSION)" --version "$(VERSION)" ambassador/
	curl -o tmp.yaml -k -L https://getambassador.io/helm/index.yaml
	helm repo index helm --url https://www.getambassador.io/helm --merge tmp.yaml

helm-update: helm
	aws s3api put-object --bucket datawire-static-files \
		--key ambassador/ambassador-$(VERSION).tgz \
		--body helm/ambassador-$(VERSION).tgz
	aws s3api put-object --bucket datawire-static-files \
		--key ambassador/index.yaml \
		--body helm/index.yaml
	rm tmp.yaml helm/index.yaml helm/ambassador-$(VERSION).tgz

e2e: E2E_TEST_NAME=all
e2e: e2e-versioned-manifests
	source venv/bin/activate; \
	if [ "$(E2E_TEST_NAME)" == "all" ]; then \
		bash end-to-end/testall.sh; \
	else \
		E2E_TEST_NAME=$(E2E_TEST_NAME) bash end-to-end/testall.sh; \
	fi

TELEPROXY=venv/bin/teleproxy
TELEPROXY_VERSION=0.1.0
# This should maybe be replaced with a lighterweight dependency if we
# don't currently depend on go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(TELEPROXY):
	curl -o $(TELEPROXY) https://s3.amazonaws.com/datawire-static-files/teleproxy/$(TELEPROXY_VERSION)/$(GOOS)/$(GOARCH)/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod go-w $(TELEPROXY)
	sudo chmod a+sx $(TELEPROXY)

KUBERNAUT=venv/bin/kubernaut

$(KUBERNAUT):
	curl -o $(KUBERNAUT) https://s3.amazonaws.com/datawire-static-files/kubernaut/$(shell curl -f -s https://s3.amazonaws.com/datawire-static-files/kubernaut/stable.txt)/kubernaut
	chmod +x $(KUBERNAUT)

venv/bin/ambassador:
	venv/bin/pip -v install -q -e ambassador/.

setup-develop: venv $(TELEPROXY) venv/bin/ambassador $(KUBERNAUT)

cluster.yaml:
	$(KUBERNAUT) discard
	$(KUBERNAUT) claim
	cp ~/.kube/kubernaut cluster.yaml
	rm -f /tmp/k8s-*.yaml
	skill -INT teleproxy
	$(TELEPROXY) -kubeconfig $(shell pwd)/cluster.yaml 2> /tmp/teleproxy.log &

KUBECONFIG=$(shell pwd)/cluster.yaml

shell: setup-develop cluster.yaml
	KUBECONFIG=$(KUBECONFIG) bash --init-file releng/init.sh -i

clean-test:
	rm -f cluster.yaml
	$(KUBERNAUT) discard
	skill -INT teleproxy

test: version setup-develop cluster.yaml
	cd ambassador && KUBECONFIG=$(KUBECONFIG) PATH=$(shell pwd)/venv/bin:$(PATH) pytest --tb=short --cov=ambassador --cov=ambassador_diag --cov-report term-missing

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
		DOC_RELEASE_TYPE=stable make website publish-website; \
		make SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws; \
		make helm-update; \
		set +x; \
	else \
		printf "'make release' can only be run for a GA commit when VERSION is not the same as GIT_COMMIT!\n"; \
		exit 1; \
	fi

# ------------------------------------------------------------------------------
# Virtualenv
# ------------------------------------------------------------------------------

venv: version venv/bin/activate

venv/bin/activate: dev-requirements.txt ambassador/.
	test -d venv || virtualenv venv --python python3
	venv/bin/pip -v install -q -Ur dev-requirements.txt
	venv/bin/pip -v install -q -e ambassador/.
	touch venv/bin/activate

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

publish-website:
	RELEASE_TYPE=$(DOC_RELEASE_TYPE) \
    NETLIFY_SITE=$(NETLIFY_SITE) \
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
