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

MANIFESTS?=$(wildcard k8s/*.yaml)

push_ok: env
	@if [ "$(PROFILE)" == "prod" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi
.PHONY: push_ok

push: push_ok docker
	docker push $(IMAGE)
	echo $(IMAGE) > pushed.txt
.PHONY: push

KUBEAPPLY=$(CURDIR)/kubeapply
KUBEAPPLY_VERSION=0.3.2
# This should maybe be replaced with a lighterweight dependency
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(KUBEAPPLY):
	curl -o $(KUBEAPPLY) https://s3.amazonaws.com/datawire-static-files/kubeapply/$(KUBEAPPLY_VERSION)/$(GOOS)/$(GOARCH)/kubeapply
	chmod go-w,a+x $(KUBEAPPLY)

apply: $(CLUSTER) $(KUBEAPPLY)
	KUBECONFIG=$(CLUSTER) IMAGE=$(file <pushed.txt) $(KUBEAPPLY) $(MANIFESTS:%=-f %)
.PHONY: apply

deploy: push apply
.PHONY: deploy

k8s.clobber:
	rm -rf $(KUBEAPPLY)
