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

# Don't use 'generate' because the OSS Makefile's `make generate` does
# not work when `$(OSS_HOME) != $(CURDIR)`.
pro-generate: cmd/amb-sidecar/firstboot/bindata.go
cmd/amb-sidecar/firstboot/bindata.go: $(GO_BINDATA) $(shell find cmd/amb-sidecar/firstboot/bindata/)
	PATH=$(dir $(GO_BINDATA)):$$PATH; cd $(@D) && go generate
pro-generate-clean:
	rm -f cmd/amb-sidecar/firstboot/bindata.go
.PHONY: pro-generate pro-generate-clean
