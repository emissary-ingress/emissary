ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
include $(dir $(lastword $(MAKEFILE_LIST)))/build-aux/kubeapply.mk

CI_IMAGE_SHA=$(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)
CI_IMAGE_TAG=$(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_TAG)

DEV_REGISTRY ?= $(REGISTRY)
DEV_REGISTRY_NAMESPACE=$(REGISTRY_NAMESPACE)
DEV_VERSION=$(HASH)
DEV_REPO=$(DEV_REGISTRY_NAMESPACE)/$(NAME)
DEV_IMAGE=$(DEV_REGISTRY)/$(DEV_REPO):$(DEV_VERSION)

PRD_REGISTRY ?= $(REGISTRY)
PRD_REGISTRY_NAMESPACE ?= $(REGISTRY_NAMESPACE)
PRD_VERSION ?= $(VERSION)
PRD_REPO=$(PRD_REGISTRY_NAMESPACE)/$(NAME)
PRD_IMAGE=$(PRD_REGISTRY)/$(PRD_REPO):$(PRD_VERSION)

PROFILE=DEV

IMAGE=$($(PROFILE)_IMAGE)

K8S_BUILD ?= k8s_build
K8S_DIR ?= k8s

UNVERSIONED ?= "(Makefile|k8s.make|$(K8S_DIR)/.*)"
VERSIONED=$(K8S_BUILD)/versioned.txt
HASH_FILE=$(K8S_BUILD)/hash.txt
HASH=$(shell cat $(HASH_FILE))

.PHONY: $(VERSIONED)
$(VERSIONED):
	@mkdir -p $(dir $(VERSIONED))
	git ls-files --exclude-standard | fgrep -v $(K8S_BUILD) | egrep -v $(UNVERSIONED) > $@
	git ls-files --exclude-standard --others | fgrep -v $(K8S_BUILD) | ( egrep -v $(UNVERSIONED) || true ) >> $@

$(HASH_FILE): $(VERSIONED)
	sha1sum $(VERSIONED) $(shell cat $(VERSIONED)) | sha1sum | cut -d" " -f1 > $@

ACTIONS=$(K8S_BUILD)/actions

guard_line=$(HASH) $(strip $(1))

define guard
@if ! fgrep -s -q "$(guard_line)" $(ACTIONS); then \
	$(2) && echo "$(guard_line)" >> $(ACTIONS); \
else \
	echo "Hash $(guard_line) already done."; \
fi
endef

.PHONY: build
build: $(HASH_FILE)
	$(call guard, build, docker build . -t $(IMAGE))

.PHONY: push_ok
push_ok: $(HASH_FILE)
	@if [ "$(IMAGE)" == "$(PRD_IMAGE)" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi

.PHONY: push
push: push_ok build
	$(call guard, push, docker push $(IMAGE))

.PHONY: deploy
deploy: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && IMAGE=$(IMAGE) && source ./env.sh && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: apply
apply: push $(KUBEAPPLY) env.sh $(wildcard $(K8S_DIR)/*.yaml)
	set -a && source ./env.sh && IMAGE=test && $(KUBEAPPLY) $(foreach y,$(filter %.yaml,$^), -f $y)

.PHONY: clean-k8s
clean-k8s:
	rm -rf $(K8S_BUILD)

.PHONY: push-commit-image
push-commit-image: $(HASH_FILE)
	docker tag $(IMAGE) $(CI_IMAGE_SHA)
	docker push $(CI_IMAGE_SHA)

.PHONY: push-tagged-image
push-tagged-image:
	docker pull $(CI_IMAGE_SHA)
	docker tag $(CI_IMAGE_SHA) $(CI_IMAGE_TAG)
	docker push $(CI_IMAGE_TAG)

endif
