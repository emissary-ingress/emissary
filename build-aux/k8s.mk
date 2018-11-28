
build-aux/env.$(PROFILE).mk: build-aux/env.go config.json
	PROFILE=$(PROFILE) go run build-aux/env.go -input config.json -output $@
.PHONY: build-aux/env.$(PROFILE).mk

env: build-aux/env.$(PROFILE).mk
	$(eval $(file <build-aux/env.$(PROFILE).mk))
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
	mkdir -p $(MANIFESTS_DIR) && cat $< | IMAGE=$(file <build-aux/pushed.txt) envsubst> $@

push_ok: env
	@if [ "$(PROFILE)" == "prod" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi
.PHONY: push_ok

push: push_ok docker
	docker push $(IMAGE)
	echo $(IMAGE) > build-aux/pushed.txt
.PHONY: push

manifests: $(MANIFESTS)
.PHONY: manifests

KUBEAPPLY=$(GOBIN)/kubeapply

$(KUBEAPPLY):
	$(GO) get github.com/datawire/teleproxy/cmd/kubeapply

apply: $(CLUSTER) $(KUBEAPPLY)
	$(KUBEAPPLY) $(MANIFESTS:%=-f %)
.PHONY: apply

deploy: push manifests apply
.PHONY: deploy

k8s.clean:
	rm -rf $(MANIFESTS_DIR) build-aux/env.*.mk
.PHONY: k8s.clean

k8s.clobber:
	rm -rf $(KUBEAPPLY)
