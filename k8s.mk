<<<<<<< HEAD
PROFILE?=dev

# NOTE: this is not a typo, this is actually how you spell newline in make
define NL


endef

# NOTE: this is not a typo, this is actually how you spell space in make
define SPACE
 
endef
=======
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
include $(dir $(lastword $(MAKEFILE_LIST)))/kubernaut-ui.mk
include $(dir $(lastword $(MAKEFILE_LIST)))/kubeapply.mk

PROFILE?=dev
>>>>>>> origin

IMAGE_VARS=$(filter %_IMAGE,$(.VARIABLES))
IMAGES=$(foreach var,$(IMAGE_VARS),$($(var)))
IMAGE_DEFS=$(foreach var,$(IMAGE_VARS),$(var)=$($(var))$(NL))
IMAGE_DEFS_SH="$(subst $(SPACE),\n,$(foreach var,$(IMAGE_VARS),$(var)=$($(var))))\n"
<<<<<<< HEAD

env:
	$(eval $(subst @NL,$(NL), $(shell go run build-aux/env.go -profile $(PROFILE) -newline "@NL" -input config.json)))
.PHONY: env

=======
MANIFESTS?=$(wildcard k8s/*.yaml)

env: ## ???
	$(eval $(subst @NL,$(NL), $(shell go run build-aux/env.go -profile $(PROFILE) -newline "@NL" -input config.json)))
.PHONY: env

hash: ## ???
>>>>>>> origin
hash: env
	@echo HASH=$(HASH)
.PHONY: hash

<<<<<<< HEAD
MANIFESTS?=$(wildcard k8s/*.yaml)

=======
push_ok: ## ???
>>>>>>> origin
push_ok: env
	@if [ "$(PROFILE)" == "prod" ]; then echo "CANNOT PUSH TO PROD"; exit 1; fi
.PHONY: push_ok

<<<<<<< HEAD

blah: env
	@echo '$(IMAGES)'
	@echo '$(IMAGE_DEFS)'

=======
blah: ## ???
blah: env
	@echo '$(IMAGES)'
	@echo '$(IMAGE_DEFS)'
.PHONY: blah

push: ## Docker push
>>>>>>> origin
push: push_ok docker
	@for IMAGE in $(IMAGES); do \
		docker push $${IMAGE}; \
	done
	printf $(IMAGE_DEFS_SH) > pushed.txt
.PHONY: push

<<<<<<< HEAD
KUBEAPPLY=$(CURDIR)/kubeapply
KUBEAPPLY_VERSION=0.3.5
# This should maybe be replaced with a lighterweight dependency
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(KUBEAPPLY):
	curl -o $(KUBEAPPLY) https://s3.amazonaws.com/datawire-static-files/kubeapply/$(KUBEAPPLY_VERSION)/$(GOOS)/$(GOARCH)/kubeapply
	chmod go-w,a+x $(KUBEAPPLY)

=======
apply: ## ???
>>>>>>> origin
apply: $(CLUSTER) $(KUBEAPPLY)
	KUBECONFIG=$(CLUSTER) $(sort $(shell cat pushed.txt)) $(KUBEAPPLY) $(MANIFESTS:%=-f %)
.PHONY: apply

<<<<<<< HEAD
deploy: push apply
.PHONY: deploy

k8s.clobber:
	rm -rf $(KUBEAPPLY)
=======
deploy: ## ???
deploy: push apply
.PHONY: deploy

endif
>>>>>>> origin
