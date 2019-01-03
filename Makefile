NAME=ambassador-pro

REGISTRY=quay.io
REGISTRY_NAMESPACE=datawire
VERSION=0.0.2
K8S_DIR=scripts

include build-aux/kubeapply.mk

CI_IMAGE_SHA=$(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)
CI_IMAGE_TAG=$(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_TAG)

DEV_REGISTRY ?= $(REGISTRY)
DEV_REGISTRY_NAMESPACE=$(REGISTRY_NAMESPACE)
DEV_VERSION=$(shell git describe --no-match --always --abbrev=40 --dirty)
DEV_REPO=$(DEV_REGISTRY_NAMESPACE)/$(NAME)
DEV_IMAGE=$(DEV_REGISTRY)/$(DEV_REPO):$(DEV_VERSION)

PRD_REGISTRY ?= $(REGISTRY)
PRD_REGISTRY_NAMESPACE ?= $(REGISTRY_NAMESPACE)
PRD_VERSION ?= $(VERSION)
PRD_REPO=$(PRD_REGISTRY_NAMESPACE)/$(NAME)
PRD_IMAGE=$(PRD_REGISTRY)/$(PRD_REPO):$(PRD_VERSION)

PROFILE=DEV

IMAGE=$($(PROFILE)_IMAGE)

.PHONY: build
build:
	docker build . -t $(IMAGE)

.PHONY: push_ok
push_ok:
	@if [ "$(IMAGE)" == "$(PRD_IMAGE)" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi

.PHONY: push
push: push_ok build
	docker push $(IMAGE)

.PHONY: deploy
deploy: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && IMAGE=$(IMAGE) && source ./env.sh && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: apply
apply: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && source ./env.sh && IMAGE=test && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: push-commit-image
push-commit-image:
	docker tag $(IMAGE) $(CI_IMAGE_SHA)
	docker push $(CI_IMAGE_SHA)

.PHONY: push-tagged-image
push-tagged-image:
	docker pull $(CI_IMAGE_SHA)
	docker tag $(CI_IMAGE_SHA) $(CI_IMAGE_TAG)
	docker push $(CI_IMAGE_TAG)

.PHONY: run
run: install
	@echo " >>> running oauth server"
	ambassador-oauth 

.PHONY: install
install: tools vendor
	@echo " >>> building"
	go install ./cmd/...

.PHONY: clean
clean:
	@echo " >>> cleaning compiled objects and binaries"
	go clean -i ./...

.PHONY: test
test:
	@echo " >>> testing code.."
	go test ./...

vendor:
	@echo " >>> installing dependencies"
	dep ensure -vendor-only

format:
	@echo " >>> running format"
	go fmt ./...

check_format:
	@echo " >>> checking format"
	if [ $$(go fmt $$(go list ./... | grep -v vendor/)) ]; then exit 1; fi

tools:
	command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo " >>> installing go dep"; \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh; \
	fi

e2e_build:
	@echo " >>> building docker for e2e testing"
	docker build -t e2e/test:latest e2e

e2e_test:
	@echo " >>> running e2e tests"
	docker run --rm e2e/test:latest
