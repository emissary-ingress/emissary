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

export BUILDER_PORTMAPS=-p 8080:8080 -p 8877:8877 -p 8500:8500

OSS_HOME ?= ambassador
include ${OSS_HOME}/Makefile
$(call module,apro,.)

tools/golangci-lint = $(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/golangci-lint
$(tools/golangci-lint): $(CURDIR)/build-aux/bin-go/golangci-lint/go.mod
	mkdir -p $(@D)
	cd $(<D) && go build -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

lint: $(tools/golangci-lint)
	@PS4=; set -ex; { \
	  r=0; \
	  $(tools/golangci-lint) run ./... || r=$$?; \
	  (cd vendor-ratelimit && $(tools/golangci-lint) run ./...) || r=$$?; \
	  exit $$r; \
	}
.PHONY: lint

format: $(tools/golangci-lint)
	$(tools/golangci-lint) run --fix ./... || true
	(cd vendor-ratelimit && $(tools/golangci-lint) run --fix ./...) || true
.PHONY: format

deploy: test-ready
	@docker exec -e AES_IMAGE=$(AMB_IMAGE) -it $(shell $(BUILDER)) kubeapply -f apro/k8s-aes
	@printf "$(GRN)Your ambassador service IP:$(END) $(BLD)$$(docker exec -it $(shell $(BUILDER)) kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')$(END)\n"
.PHONY: deploy

AES_BACKEND_IMAGE=gcr.io/datawireio/aes-backend:$(RELEASE_VERSION)

# XXX: should make base a make variable
deploy-aes-backend: images
	cat Dockerfile.aes-backend | docker build -t aes-backend --build-arg artifacts=$(SNAPSHOT) -
	docker tag aes-backend $(AES_BACKEND_IMAGE)
	docker push $(AES_BACKEND_IMAGE)
	@if [ -z "$(PROD_KUBECONFIG)" ]; then echo please set PROD_KUBECONFIG && exit 1; fi
	cat k8s-aes-backend/*.yaml | AES_BACKEND_IMAGE=$(AES_BACKEND_IMAGE) envsubst | kubectl --kubeconfig="$(PROD_KUBECONFIG)" apply -f -
.PHONY: deploy-aes-backend
