NAME ?= aes

ifndef OSS_HOME
AMBASSADOR_COMMIT = shared/edgy

# Git clone
# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
define SETUP
	PS4=; set +x; { \
	    unset GIT_DIR GIT_WORK_TREE; \
	    if [ -e ambassador ]; then exit; fi ; \
	    set -x; \
	    git init ambassador; \
	    cd ambassador; \
	    if ! git remote get-url origin &>/dev/null; then \
	        git remote add origin https://github.com/datawire/ambassador; \
	        git remote set-url --push origin git@github.com:datawire/ambassador.git; \
	    fi; \
	    git fetch || true; \
	    if [ $(AMBASSADOR_COMMIT) != '-' ]; then \
	        git checkout $(AMBASSADOR_COMMIT); \
	    elif ! git rev-parse HEAD >/dev/null 2>&1; then \
	        git checkout origin/master; \
	    fi; \
	}
endef
endif

DUMMY:=$(shell $(SETUP))

OSS_HOME ?= ambassador
include ${OSS_HOME}/Makefile
$(call module,apro,.)

deploy: test-ready
	@docker exec -e AES_IMAGE=$(AMB_IMAGE) -it $(shell $(BUILDER)) kubeapply -f apro/k8s-aes
	@printf "$(GRN)Your ambassador service IP:$(END) $(BLD)$$(docker exec -it $(shell $(BUILDER)) kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')$(END)\n"


include build-aux/go-bindata.mk

# Don't use 'generate' because the OSS Makefile's `make generate` does
# not work when `$(OSS_HOME) != $(CURDIR)`.
pro-generate: cmd/amb-sidecar/firstboot/bindata.go
cmd/amb-sidecar/firstboot/bindata.go: $(GO_BINDATA) $(shell find cmd/amb-sidecar/firstboot/bindata/)
	PATH=$(dir $(GO_BINDATA)):$$PATH; cd $(@D) && go generate
pro-generate-clean:
	rm -f cmd/amb-sidecar/firstboot/bindata.go
.PHONY: pro-generate pro-generate-clean
