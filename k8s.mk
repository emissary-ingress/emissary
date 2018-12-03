PROFILE?=dev

# NOTE: this is not a typo, this is actually how you spell newline in make
define NL


endef

env:
	$(eval $(subst @NL,$(NL), $(shell go run build-aux/env.go -profile $(PROFILE) -newline "@NL" -input config.json)))
.PHONY: env

hash: env
	@echo HASH=$(HASH)
.PHONY: hash

K8S_DIR ?= k8s
TEMPLATES:=$(wildcard $(K8S_DIR)/*.yaml)

MANIFESTS_DIR=$(K8S_DIR).$(PROFILE)
MANIFESTS=$(TEMPLATES:$(K8S_DIR)/%.yaml=$(MANIFESTS_DIR)/%.yaml)

export IMAGE

$(MANIFESTS_DIR)/%.yaml : $(K8S_DIR)/%.yaml env
	@echo "Generating $< -> $@"
	mkdir -p $(MANIFESTS_DIR) && cat $< | IMAGE=$(file <pushed.txt) envsubst> $@

push_ok: env
	@if [ "$(PROFILE)" == "prod" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi
.PHONY: push_ok

push: push_ok docker
	docker push $(IMAGE)
	echo $(IMAGE) > pushed.txt
.PHONY: push

manifests: $(MANIFESTS)
.PHONY: manifests

KUBEAPPLY=$(GOBIN)/kubeapply

$(KUBEAPPLY):
	$(GO) get github.com/datawire/teleproxy/cmd/kubeapply

apply: manifests $(CLUSTER) $(KUBEAPPLY)
	$(KUBEAPPLY) $(MANIFESTS:%=-f %)
.PHONY: apply

deploy: push apply
.PHONY: deploy

k8s.clean:
	rm -rf $(MANIFESTS_DIR)
.PHONY: k8s.clean

k8s.clobber:
	rm -rf $(KUBEAPPLY)
