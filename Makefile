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

update-yaml-locally: sync
	@printf "$(CYN)==> $(GRN)Updating development YAML$(END)\n"
	@printf '  $(CYN)k8s-aes/00-aes-crds.yaml$(END)\n'
	docker exec $(shell $(BUILDER)) python apro/fix-crds.py ambassador/docs/yaml/ambassador/ambassador-crds.yaml apro/k8s-aes-src/00-aes-crds.yaml > k8s-aes/00-aes-crds.yaml
	@printf '  $(CYN)k8s-aes/01-aes.yaml$(END)\n'
	docker exec $(shell $(BUILDER)) python apro/fix-yaml.py apro ambassador/docs/yaml/ambassador/ambassador-rbac.yaml apro/k8s-aes-src/01-aes.yaml > k8s-aes/01-aes.yaml
	@printf "$(CYN)==> $(GRN)Checking whether those changes were no-op$(END)\n"
	git diff k8s-aes
	@if [ -n "$$(git diff k8s-aes)" ]; then \
		printf "$(RED)Please inspect and commit the above changes; then re-run the $(BLU)$(MAKE) $(MAKECMDGOALS)$(RED) command$(END)\n"; \
		exit 1; \
	fi
.PHONY: update-yaml-locally

preflight-docs:
	@if [ -z "$${AMBASSADOR_DOCS}" ]; then printf "$(RED)Please set AMBASSADOR_DOCS to point to your ambassador-docs.git checkout$(END)\n" >&2; exit 1; fi
	@if ! [ -f "$${AMBASSADOR_DOCS}/versions.yml" ]; then printf "$(RED)AMBASSADOR_DOCS=$${AMBASSADOR_DOCS} does not look like an ambassador-docs.git checkout$(END)\n" >&2; exit 1; fi
.PHONY: preflight-docs

update-yaml: update-yaml-locally preflight-docs
	git -C "$${AMBASSADOR_DOCS}" fetch --all --prune --tags
	@printf "$(GRN)In another terminal, verify that your AMBASSADOR_DOCS ($(AMBASSADOR_DOCS)) checkout is up-to-date with the desired branch (probably $(BLU)early-access$(GRN))$(END)\n"
	@read -s -p "$$(printf '$(GRN)Press $(BLU)enter$(GRN) once you have verified this$(END)')"
	@echo
	@printf "$(CYN)==> $(GRN)Updating AMBASSADOR_DOCS YAML$(END)\n"
	@printf '  $(CYN)$${AMBASSADOR_DOCS}/yaml/aes-crds.yaml$(END)\n'
	cp k8s-aes/00-aes-crds.yaml $${AMBASSADOR_DOCS}/yaml/aes-crds.yaml
	@printf '  $(CYN)$${AMBASSADOR_DOCS}/yaml/aes.yaml$(END)\n'
	docker exec $(shell $(BUILDER)) python apro/fix-yaml.py edge_stack ambassador/docs/yaml/ambassador/ambassador-rbac.yaml apro/k8s-aes-src/01-aes.yaml > $${AMBASSADOR_DOCS}/yaml/aes.yaml
	@printf '  $(CYN)$${AMBASSADOR_DOCS}/yaml/oss-migration.yaml$(END)\n'
	sed -e 's/# NOT a generated file/# GENERATED FILE: DO NOT EDIT/' < k8s-aes-src/02-oss-migration.yaml > $${AMBASSADOR_DOCS}/yaml/oss-migration.yaml
	@printf '  $(CYN)$${AMBASSADOR_DOCS}/yaml/resources-migration.yaml$(END)\n'
	sed -e 's/# NOT a generated file/# GENERATED FILE: DO NOT EDIT/' < k8s-aes-src/03-resources-migration.yaml > $${AMBASSADOR_DOCS}/yaml/resources-migration.yaml
	@printf "$(CYN)==> $(GRN)Checking whether those changes were no-op$(END)\n"
	git -C "$${AMBASSADOR_DOCS}" diff .
	@if [ -n "$$(git -C $${AMBASSADOR_DOCS} diff .)" ]; then \
		printf "$(RED)Please inspect and commit the above changes to $(BLU)${AMBASSADOR_DOCS}$(RED); then re-run the $(BLU)$(MAKE) $(MAKECMDGOALS)$(RED) command$(END)\n"; \
		exit 1; \
	fi
.PHONY: update-yaml


final-push: preflight-docs
	@if [ "$$(git push --tags --dry-run 2>&1)" != "Everything up-to-date" ]; then \
		printf "$(RED)Please run: git push --tags$(END)\n"; \
	fi	
	@if [ "$$(git -C $${AMBASSADOR_DOCS} push --dry-run 2>&1)" != "Everything up-to-date" ]; then \
		printf "$(RED)Please run: git -C $${AMBASSADOR_DOCS} push$(END)\n"; \
	fi	

tag-rc:
	@if [ -z "$$(git describe --exact-match HEAD)" ]; then \
		echo Last 10 tags: ; \
		git tag --sort v:refname | egrep '^v[0-9]' | tail -10 ; \
		(read -p "Please enter rc tag: " TAG && echo $${TAG} > /tmp/rc.tag) ; \
		git tag -a $$(cat /tmp/rc.tag) ; \
		git push --tags ; \
	fi

aes-rc: update-yaml
	@$(MAKE) --no-print-directory tag-rc
	@$(MAKE) --no-print-directory final-push
	@printf "Please check your release here: https://circleci.com/gh/datawire/apro\n"

aes-rc-now: update-yaml
	@if [ -n "$$(git status --porcelain)" ]; then \
		printf "$(RED)Your checkout must be clean.$(END)\n" && exit 1; \
	fi
	@$(MAKE) --no-print-directory tag-rc
	@$(MAKE) --no-print-directory rc RELEASE_REGISTRY=quay.io/datawire-dev
	@$(MAKE) --no-print-directory final-push
