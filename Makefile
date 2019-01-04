include build-aux/kubeapply.mk
include build-aux/help.mk
.DEFAULT_GOAL = help

NAME=ambassador-pro

REGISTRY=quay.io
REGISTRY_NAMESPACE=datawire
VERSION=0.0.2
K8S_DIR=scripts

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

define help.body
  PROFILE      = $(PROFILE)
  DEV_IMAGE    = $(DEV_IMAGE)
  PRD_IMAGE    = $(PRD_IMAGE)
  IMAGE        = $(IMAGE) # $(value IMAGE)
  CI_IMAGE_SHA = $(CI_IMAGE_SHA) # $(value CI_IMAGE_SHA)
  CI_IMAGE_TAG = $(CI_IMAGE_TAG) # $(value CI_IMAGE_TAG)
  GOBIN        = $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
endef

.PHONY: build
build: ## docker build -t $(IMAGE)
	docker build . -t $(IMAGE)

.PHONY: push_ok
push_ok: ## Check whether it is OK to Docker push
	@if [ "$(IMAGE)" == "$(PRD_IMAGE)" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi

.PHONY: push
push: ## docker push $(IMAGE)
push: push_ok build
	docker push $(IMAGE)

.PHONY: deploy
deploy: ## Deploy $(IMAGE) to a k8s cluster
deploy: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && IMAGE=$(IMAGE) && . ./env.sh && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: apply
apply: ## Like 'deploy', but sets IMAGE=test
apply: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && . ./env.sh && IMAGE=test && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: push-commit-image
push-commit-image: ## docker push $(CI_IMAGE_SHA)
	docker tag $(IMAGE) $(CI_IMAGE_SHA)
	docker push $(CI_IMAGE_SHA)

.PHONY: push-tagged-image
push-tagged-image: ## docker push $(CI_IMAGE_TAG)
	docker pull $(CI_IMAGE_SHA)
	docker tag $(CI_IMAGE_SHA) $(CI_IMAGE_TAG)
	docker push $(CI_IMAGE_TAG)

.PHONY: run
run: ## Run ambassador-oauth locally
run: install
	@echo " >>> running oauth server"
	ambassador-oauth 

.PHONY: install
install: ## Compile ambassador-oauth (to $GOBIN)
install: vendor
	@echo " >>> building"
	go install ./cmd/...

.PHONY: clean
clean: ## Clean
	@echo " >>> cleaning compiled objects and binaries"
	go clean -i ./...

.PHONY: test
test: ## Check: unit tests
	@echo " >>> testing code.."
	go test ./...

vendor: ## Update the ./vendor/ directory based on Gopkg.toml
	@echo " >>> installing dependencies"
	dep ensure -vendor-only

format: ## Adjust the source code per `go fmt`
	@echo " >>> running format"
	go fmt ./...

check_format: ## Check: go fmt
	@echo " >>> checking format"
	if [ $$(go fmt $$(go list ./... | grep -v vendor/)) ]; then exit 1; fi

e2e_build: ## Build a oauth-client Docker image, for e2e testing
	@echo " >>> building docker for e2e testing"
	docker build -t e2e/test:latest e2e

e2e_test: ## Check: e2e tests
	@echo " >>> running e2e tests"
	docker run --rm e2e/test:latest
