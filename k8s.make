ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
include $(dir $(lastword $(MAKEFILE_LIST)))/build-aux/kubeapply.mk

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

K8S_DIR ?= k8s

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

endif
