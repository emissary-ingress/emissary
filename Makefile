NAME ?= aes

ifndef OSS_HOME
AMBASSADOR_COMMIT = $(shell cat ambassador.commit)

# Git clone ambassador to the specified checkout if OSS_HOME is not set
# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
define SETUP
	PS4=; set +x; { \
	    unset GIT_DIR GIT_WORK_TREE; \
	    if ! [ -e ambassador ]; then \
	        git init ambassador; \
		INIT=yes; \
	    fi ; \
	    if ! git -C ambassador remote get-url origin &>/dev/null; then \
	        set -x; \
	        git -C ambassador remote add origin https://github.com/datawire/ambassador; \
	        git -C ambassador remote set-url --push origin no_push; \
	    fi; \
	    { set +x 1; } 2>/dev/null; \
	    if [ -n "$${INIT}" ] || [ "$$(cd ambassador && git rev-parse HEAD)" != "$(AMBASSADOR_COMMIT)" ]; then \
	        set -x; \
	        git -C ambassador fetch; \
		git -C ambassador checkout -q $(AMBASSADOR_COMMIT); \
	    fi; \
	}
endef
endif

OUTPUT:=$(shell $(SETUP))
ifneq ($(strip $(OUTPUT)),)
$(info $(OUTPUT))
endif

OSS_HOME ?= ambassador
include ${OSS_HOME}/Makefile
$(call module,apro,.)

deploy: test-ready
	@docker exec -e AES_IMAGE=$(AMB_IMAGE) -it $(shell $(BUILDER)) kubeapply -f apro/k8s-aes
	@printf "$(GRN)Your ambassador service IP:$(END) $(BLD)$$(docker exec -it $(shell $(BUILDER)) kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')$(END)\n"
.PHONY: deploy

include build-aux/go-bindata.mk

sync: cmd/amb-sidecar/webui/bindata.go
cmd/amb-sidecar/webui/bindata.go: $(GO_BINDATA) $(shell find cmd/amb-sidecar/webui/bindata/)
	PATH=$(dir $(GO_BINDATA)):$$PATH; cd $(@D) && go generate

AES_BACKEND_IMAGE=gcr.io/datawireio/aes-backend:$(RELEASE_VERSION)

# XXX: should make base a make variable
deploy-aes-backend: images
	cat Dockerfile.aes-backend | docker build -t aes-backend --build-arg artifacts=$(SNAPSHOT) -
	docker tag aes-backend $(AES_BACKEND_IMAGE)
	docker push $(AES_BACKEND_IMAGE)
	@if [ -z "$(PROD_KUBECONFIG)" ]; then echo please set PROD_KUBECONFIG && exit 1; fi
	cat k8s-aes-backend/*.yaml | AES_BACKEND_IMAGE=$(AES_BACKEND_IMAGE) envsubst | kubectl --kubeconfig="$(PROD_KUBECONFIG)" apply -f -
.PHONY: deploy-aes-backend
