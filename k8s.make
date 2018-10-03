# This is the namespace you want to use for development in quay,
# probably your quay.io username:

DEV_REGISTRY ?= $(REGISTRY)
DEV_REGISTRY_NAMESPACE=$(shell source $(PWD)/env.sh && echo $${REGISTRY_NAMESPACE})
DEV_VERSION=$(HASH)
DEV_REPO=$(DEV_REGISTRY_NAMESPACE)/$(NAME)
DEV_IMAGE=$(DEV_REGISTRY)/$(DEV_REPO):$(DEV_VERSION)

PRD_REGISTRY ?= $(REGISTRY)
PRD_REGISTRY_NAMESPACE ?= $(REGISTRY_NAMESPACE)
PRD_VERSION ?= $(VERSION)
PRD_REPO=$(PRD_REGISTRY_NAMESPACE)/$(NAME)
PRD_IMAGE=$(PRD_REGISTRY)/$(PRD_REPO):$(PRD_VERSION)

IMAGE=$($(PROFILE)_IMAGE)

K8S_BUILD ?= k8s_build
K8S_DIR ?= k8s
TEMPLATES:=$(wildcard $(K8S_DIR)/*.yaml)

UNVERSIONED ?= "(Makefile|k8s.make|$(K8S_DIR)/.*)"
VERSIONED=$(K8S_BUILD)/versioned.txt
HASH_FILE=$(K8S_BUILD)/hash.txt
HASH=$(shell cat $(HASH_FILE))

.PHONY: $(VERSIONED)
$(VERSIONED):
	@mkdir -p $(dir $(VERSIONED))
	@git ls-files --exclude-standard | fgrep -v $(K8S_BUILD) | egrep -v $(UNVERSIONED) > $@
	@git ls-files --exclude-standard --others | fgrep -v $(K8S_BUILD) | ( egrep -v $(UNVERSIONED) || true ) >> $@

$(HASH_FILE): $(VERSIONED)
	@sha1sum $(VERSIONED) $(shell cat $(VERSIONED)) | sha1sum | cut -d" " -f1 > $@

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

MANIFESTS_DIR=$(K8S_BUILD)/$(PROFILE)
MANIFESTS=$(TEMPLATES:$(K8S_DIR)/%.yaml=$(MANIFESTS_DIR)/%.yaml)

$(MANIFESTS_DIR)/%.yaml : $(K8S_DIR)/%.yaml 
	@echo "Generating $< -> $@"
	@mkdir -p $(MANIFESTS_DIR) && cat $< | /bin/bash -c "set -a && source $(ENV_FILE) && set +a && IMAGE=$(IMAGE) envsubst" > $@

manifests: $(HASH_FILE) $(MANIFESTS)
apply_cmd = kubectl apply -f $$FILE

define apply
for FILE in $(MANIFESTS); do echo "$(apply_cmd)" && $(apply_cmd); done
endef

.PHONY: deploy
deploy: push $(MANIFESTS)
	$(call guard, deploy, $(apply))

.PHONY: apply
apply: $(HASH_FILE) $(MANIFESTS)
	$(call guard, apply-$(PROFILE), $(apply))

.PHONY: clean-k8s
clean-k8s:
	rm -rf $(K8S_BUILD)

.PHONY: generate
generate: $(HASH_FILE) $(MANIFESTS)