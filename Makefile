NAME ?= aes

ifndef OSS_HOME
AMBASSADOR_COMMIT = $(shell cat ambassador.commit)

# Git clone ambassador to the specified checkout if OSS_HOME is not set
define SETUP
	./get-amb-repo.sh
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
	@printf "$(CYN)==> $(GRN)Checking whether AMBASSADOR_DOCS is up to date$(END)\n"
	git -C "$${AMBASSADOR_DOCS}" fetch --all --prune --tags
	@printf "$(GRN)In another terminal, verify that your AMBASSADOR_DOCS ($(AMBASSADOR_DOCS)) checkout is up-to-date with the desired branch (probably $(BLU)early-access$(GRN))$(END)\n"
	@read -s -p "$$(printf '$(GRN)Press $(BLU)enter$(GRN) once you have verified this:$(END)')"
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

release/bits: release/bits/aes-plugin-runner
release/bits/aes-plugin-runner: bin_linux_amd64/aes-plugin-runner bin_darwin_amd64/aes-plugin-runner
	aws s3 cp --acl public-read bin_linux_amd64/aes-plugin-runner "s3://datawire-static-files/aes-plugin-runner/$(RELEASE_VERSION)/linux/amd64/aes-plugin-runner"
	aws s3 cp --acl public-read bin_darwin_amd64/aes-plugin-runner "s3://datawire-static-files/aes-plugin-runner/$(RELEASE_VERSION)/darwin/amd64/aes-plugin-runner"
bin_linux_amd64/aes-plugin-runner: FORCE
	mkdir -p $(@D)
	docker cp $$($(BUILDER)):/buildroot/bin/aes-plugin-runner $@
bin_darwin_amd64/aes-plugin-runner: FORCE
	mkdir -p $(@D)
	docker cp $$($(BUILDER)):/buildroot/bin-darwin/aes-plugin-runner $@

release/promote-aes/.main:
	@[[ '$(PROMOTE_FROM_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]
	@[[ '$(PROMOTE_TO_VERSION)'   =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]
	@printf "$(CYN)==> $(GRN)Promoting $(BLU)%s$(GRN) to $(BLU)%s$(GRN)$(END)\n" '$(PROMOTE_FROM_VERSION)' '$(PROMOTE_TO_VERSION)'

	@printf '  $(CYN)$(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION)$(END)\n'
	docker pull $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION)
	docker tag $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION) $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_TO_VERSION)
	docker push $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_TO_VERSION)
.PHONY: release/promote-aes/.main

# To be run from a checkout at the tag you are promoting _from_.
# At present, this is to be run by-hand.
release/promote-aes/to-ea-latest:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-ea[0-9]+$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an EA tag\n' "$(RELEASE_VERSION)"; exit 1)
	@{ $(MAKE) release/promote-aes/.main \
	  PROMOTE_FROM_VERSION="$(RELEASE_VERSION)" \
	  PROMOTE_TO_VERSION="$$(echo "$(RELEASE_VERSION)" | sed 's/-ea.*/-ea-latest/')" \
	; }
.PHONY: release/promote-aes/to-ea-latest

# To be run from a checkout at the tag you are promoting _from_.
# At present, this is to be run by-hand.
release/promote-aes/to-rc-latest:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]+$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an RC tag\n' "$(RELEASE_VERSION)"; exit 1)
	@{ $(MAKE) release/promote-aes/.main \
	  PROMOTE_FROM_VERSION="$(RELEASE_VERSION)" \
	  PROMOTE_TO_VERSION="$$(echo "$(RELEASE_VERSION)" | sed 's/-rc.*/-rc-latest/')" \
	; }
.PHONY: release/promote-aes/to-rc-latest

# To be run from a checkout at the tag you are promoting _to_.
# This is normally run from CI by creating the GA tag.
# This assumes that the version you are promoting from is the same as the OSS -rc-latest version.
release/promote-aes/to-ga:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(RELEASE_VERSION)"; exit 1)
	@set -e; {
	  rc_latest=$(curl -sL --fail https://s3.amazonaws.com/datawire-static-files/ambassador/teststable.txt); \
	  if ! [[ "$$rc_latest" == "$(RELEASE_VERSION)"-rc* ]]; then \
	    printf '$(RED)ERROR: https://s3.amazonaws.com/datawire-static-files/ambassador/teststable.txt => %s does not look like a RC of %s' "$$rc_latest" "$(RELEASE_VERSION)"; \
	    exit 1; \
	  fi; \
	  $(MAKE) release/promote-aes/.main \
	    PROMOTE_FROM_VERSION="$$rc_latest" \
	    PROMOTE_TO_VERSION="$(RELEASE_VERSION)" \
	    ; \
	  printf '  $(CYN)aes-plugin-runner$(END)\n'; \
	  aws s3 cp --acl public-read "s3://datawire-static-files/aes-plugin-runner/$$rc_latest/linux/amd64/aes-plugin-runner" \
	                              "s3://datawire-static-files/aes-plugin-runner/$(RELEASE_VERSION)/linux/amd64/aes-plugin-runner"; \
	  aws s3 cp --acl public-read "s3://datawire-static-files/aes-plugin-runner/$$rc_latest/darwin/amd64/aes-plugin-runner" \
	                              "s3://datawire-static-files/aes-plugin-runner/$(RELEASE_VERSION)/darwin/amd64/aes-plugin-runner"; \
	  printf '  $(CYN)edgectl$(END)\n'; \
	  aws s3 cp --acl public-read "s3://datawire-static-files/edgectl/$$rc_latest/linux/amd64/edgectl" \
	                              "s3://datawire-static-files/edgectl/$(RELEASE_VERSION)/linux/amd64/edgectl"; \
	  aws s3 cp --acl public-read "s3://datawire-static-files/edgectl/$$rc_latest/darwin/amd64/edgectl" \
	                              "s3://datawire-static-files/edgectl/$(RELEASE_VERSION)/darwin/amd64/edgectl"; \
	  aws s3 cp --acl public-read "s3://datawire-static-files/edgectl/$$rc_latest/windows/amd64/edgectl.exe" \
	                              "s3://datawire-static-files/edgectl/$(RELEASE_VERSION)/windows/amd64/edgectl.exe"; \
	}
	@printf '  $(CYN)edgectl (metadata)$(END)\n'
	echo "$RELEASE_VERSION" | aws s3 cp --acl public-read - s3://datawire-static-files/edgectl/stable.txt
.PHONY: release/promote-aes/to-ga

define _help.aes-targets
  $(BLD)make $(BLU)lint$(END)                -- runs golangci-lint.

  $(BLD)make $(BLU)format$(END)              -- runs golangci-lint with --fix.

  $(BLD)make $(BLU)deploy$(END)              -- deploys AES to $(BLD)\$$DEV_REGISTRY$(END) and $(BLD)\$$DEV_KUBECONFIG$(END). ($(DEV_REGISTRY) and $(DEV_KUBECONFIG))

  $(BLD)make $(BLU)deploy-aes-backend$(END)  -- ???

  $(BLD)make $(BLU)update-yaml-locally$(END) -- updates the YAML in $(BLD)k8s-aes/$(END).

    The YAML in $(BLD)k8s-aes/$(END) is generated from $(BLD)k8s-aes-src/$(END) and
    from $(BLD)\$$OSS_HOME/docs/yaml/ambassador/ambassador-rbac.yaml$(END).

  $(BLD)make $(BLU)update-yaml$(END)         -- updates the YAML in $(BLD)k8s-aes/$(END) and in $(BLD)\$$AMBASSADOR_DOCS/yaml/$(END). ($(AMBASSADOR_DOCS)/yaml/)

  $(BLD)make $(BLU)release/promote-aes/to-rc-latest$(END) -- promote an early-access '-eaN' release to '-ea-latest'

  $(BLD)make $(BLU)release/promote-aes/to-ea-latest$(END) -- promote a release candidate '-rcN' release to '-rc-latest'

  $(BLD)make $(BLU)release/promote-aes/to-ga$(END) -- promote a release candidate to general availability
endef
_help.targets += $(NL)$(NL)$(_help.aes-targets)
