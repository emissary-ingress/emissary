
#DEV_REGISTRY=localhost:5000
#DEV_KUBECONFIG=/tmp/k3s.yaml

# Choose colors carefully. If they don't work on both a black 
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
RED=\033[1;31m
GRN=\033[1;32m
BLU=\033[1;34m
CYN=\033[1;36m
BLD=\033[1m
END=\033[0m

MODULES :=

module = $(eval MODULES += $(1))$(eval SOURCE_$(1)=$(abspath $(2)))

BUILDER_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

BUILDER = BUILDER_NAME=$(NAME) $(abspath $(BUILDER_HOME)/builder.sh)
DBUILD = $(abspath $(BUILDER_HOME)/dbuild.sh)

all: help
.PHONY: all

.NOTPARALLEL:

export RSYNC_ERR=$(RED)ERROR: please update to a version of rsync with the --info option$(END)
export DOCKER_ERR=$(RED)ERROR: cannot find docker, please make sure docker is installed$(END)

preflight:
ifeq ($(strip $(shell $(BUILDER))),)
	@printf "$(CYN)==> $(GRN)Preflight checks$(END)\n"
# Checking for rsync --info
	test -n "$$(rsync --help | fgrep -- --info)" || (printf "$${RSYNC_ERR}\n"; exit 1)
# Checking for docker
	which docker > /dev/null || (printf "$${DOCKER_ERR}\n"; exit 1)
endif
.PHONY: preflight

sync: preflight
	@$(foreach MODULE,$(MODULES),$(BUILDER) sync $(MODULE) $(SOURCE_$(MODULE)) &&) true
	@test -n "$(DEV_KUBECONFIG)" || (printf "$${KUBECONFIG_ERR}\n"; exit 1)
	@if [ "$(DEV_KUBECONFIG)" != '-skip-for-release-' ]; then \
		printf "$(CYN)==> $(GRN)Checking for test cluster$(END)\n" ;\
		kubectl --kubeconfig $(DEV_KUBECONFIG) -n default get service kubernetes > /dev/null || (printf "$${KUBECTL_ERR}\n"; exit 1) ;\
		cat $(DEV_KUBECONFIG) | docker exec -i $$($(BUILDER)) sh -c "cat > /buildroot/kubeconfig.yaml" ;\
	else \
		printf "$(CYN)==> $(RED)Skipping test cluster checks$(END)\n" ;\
	fi
	@if [ -e ~/.docker/config.json ]; then \
		cat ~/.docker/config.json | docker exec -i $$($(BUILDER)) sh -c "mkdir -p /home/dw/.docker && cat > /home/dw/.docker/config.json" ; \
	fi

	@if [ -n "$(GCLOUD_CONFIG)" ]; then \
		printf "Copying gcloud config to builder container\n"; \
		docker cp $(GCLOUD_CONFIG) $$($(BUILDER)):/home/dw/.config/; \
	fi

.PHONY: sync

builder:
	@$(BUILDER) builder
.PHONY: builder

version:
	@$(BUILDER) version
.PHONY: version

compile:
	@$(MAKE) --no-print-directory sync
	@$(BUILDER) compile
.PHONY: compile

SNAPSHOT=snapshot-$(NAME)

commit:
	@$(BUILDER) commit $(SNAPSHOT)
.PHONY: commit

REPO=$(NAME)

images:
	@$(MAKE) --no-print-directory compile
	@$(MAKE) --no-print-directory commit
.PHONY: images

AMB_IMAGE=$(DEV_REGISTRY)/$(REPO):$(shell docker images -q $(REPO):latest)
KAT_CLI_IMAGE=$(DEV_REGISTRY)/kat-client:$(shell docker images -q kat-client:latest)
KAT_SRV_IMAGE=$(DEV_REGISTRY)/kat-server:$(shell docker images -q kat-server:latest)

export REGISTRY_ERR=$(RED)ERROR: please set the DEV_REGISTRY make/env variable to the docker registry\n       you would like to use for development$(END)

push: images
	@test -n "$(DEV_REGISTRY)" || (printf "$${REGISTRY_ERR}\n"; exit 1)
	@$(BUILDER) push $(AMB_IMAGE) $(KAT_CLI_IMAGE) $(KAT_SRV_IMAGE)
.PHONY: push

export KUBECONFIG_ERR=$(RED)ERROR: please set the $(BLU)DEV_KUBECONFIG$(RED) make/env variable to the cluster\n       you would like to use for development. Note this cluster must have access\n       to $(BLU)DEV_REGISTRY$(RED) (currently $(BLD)$(DEV_REGISTRY)$(END)$(RED))$(END)
export KUBECTL_ERR=$(RED)ERROR: preflight kubectl check failed$(END)

test-ready: push
# XXX noop target for teleproxy tests
	@docker exec -w /buildroot/ambassador -i $(shell $(BUILDER)) sh -c "echo bin_linux_amd64/edgectl: > Makefile"
	@docker exec -w /buildroot/ambassador -i $(shell $(BUILDER)) sh -c "mkdir -p bin_linux_amd64"
	@docker exec -w /buildroot/ambassador -d $(shell $(BUILDER)) ln -s /buildroot/bin/edgectl /buildroot/ambassador/bin_linux_amd64/edgectl
.PHONY: test-ready

PYTEST_ARGS ?=
export PYTEST_ARGS

pytest: test-ready
	$(MAKE) pytest-only
.PHONY: pytest

pytest-only: sync
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) tests$(END)\n"
	docker exec \
		-e AMBASSADOR_DOCKER_IMAGE=$(AMB_IMAGE) \
		-e KAT_CLIENT_DOCKER_IMAGE=$(KAT_CLI_IMAGE) \
		-e KAT_SERVER_DOCKER_IMAGE=$(KAT_SRV_IMAGE) \
		-e KAT_IMAGE_PULL_POLICY=Always \
		-e DOCKER_NETWORK=$(NAME) \
		-e KAT_REQ_LIMIT \
		-e KAT_RUN_MODE \
		-e KAT_VERBOSE \
		-e PYTEST_ARGS \
		-it $(shell $(BUILDER)) /buildroot/builder.sh pytest-internal
.PHONY: pytest-only


GOTEST_PKGS ?= ./...
export GOTEST_PKGS
GOTEST_ARGS ?=
export GOTEST_ARGS

gotest: test-ready
	@printf "$(CYN)==> $(GRN)Running $(BLU)go$(GRN) tests$(END)\n"
	docker exec \
		-e DTEST_REGISTRY=$(DEV_REGISTRY) \
		-e DTEST_KUBECONFIG=/buildroot/kubeconfig.yaml \
		-e GOTEST_PKGS \
		-e GOTEST_ARGS \
		-it $(shell $(BUILDER)) /buildroot/builder.sh gotest-internal
.PHONY: gotest

test: gotest pytest
.PHONY: test

shell:
	@$(BUILDER) shell
.PHONY: shell

AMB_IMAGE_RC=$(RELEASE_REGISTRY)/$(REPO):$(RELEASE_VERSION)
AMB_IMAGE_RC_LATEST=$(RELEASE_REGISTRY)/$(REPO):$(BUILD_VERSION)-rc-latest
AMB_IMAGE_RELEASE=$(RELEASE_REGISTRY)/$(REPO):$(BUILD_VERSION)

export RELEASE_REGISTRY_ERR=$(RED)ERROR: please set the RELEASE_REGISTRY make/env variable to the docker registry\n       you would like to use for release$(END)

RELEASE_TYPE=$$($(BUILDER) release-type)
RELEASE_VERSION=$$($(BUILDER) release-version)
BUILD_VERSION=$$($(BUILDER) version)

rc: images
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@if [ "$(RELEASE_TYPE)" = release ]; then \
		(printf "$(RED)ERROR: 'make rc' can only be used for non-release tags$(END)\n" && exit 1); \
	fi
	@printf "$(CYN)==> $(GRN)Pushing release candidate $(BLU)$(REPO)$(GRN) image$(END)\n"
	docker tag $(REPO) $(AMB_IMAGE_RC)
	docker push $(AMB_IMAGE_RC)
	@if [ "$(RELEASE_TYPE)" = rc ]; then \
		docker tag $(REPO) $(AMB_IMAGE_RC_LATEST) && \
		docker push $(AMB_IMAGE_RC_LATEST) && \
		printf "$(GRN)Tagged $(RELEASE_VERSION) as latest RC$(END)\n" ; \
	fi
.PHONY: rc

release-prep:
	bash $(OSS_HOME)/releng/release-prep.sh
.PHONY: release-prep

release:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@$(MAKE) --no-print-directory sync
	@if [ "$(RELEASE_TYPE)" != release ]; then \
		(printf "$(RED)ERROR: 'make release' can only be used for release tags ('vX.Y.Z')$(END)\n" && exit 1); \
	fi
	@printf "$(CYN)==> $(GRN)Promoting release $(BLU)$(REPO)$(GRN) image$(END)\n"
	docker pull $(AMB_IMAGE_RC_LATEST)
	docker tag $(AMB_IMAGE_RC_LATEST) $(AMB_IMAGE_RELEASE)
	docker push $(AMB_IMAGE_RELEASE)
.PHONY: release

clean:
	@$(BUILDER) clean
.PHONY: clean

clobber:
	@$(BUILDER) clobber
.PHONY: clobber

CURRENT_CONTEXT=$(shell kubectl --kubeconfig=$(DEV_KUBECONFIG) config current-context)
CURRENT_NAMESPACE=$(shell kubectl config view -o=jsonpath="{.contexts[?(@.name==\"$(CURRENT_CONTEXT)\")].context.namespace}")

env:
	@printf "$(BLD)DEV_KUBECONFIG$(END)=$(BLU)\"$(DEV_KUBECONFIG)\"$(END)"
	@printf " # Context: $(BLU)$(CURRENT_CONTEXT)$(END), Namespace: $(BLU)$(CURRENT_NAMESPACE)$(END)\n"
	@printf "$(BLD)DEV_REGISTRY$(END)=$(BLU)\"$(DEV_REGISTRY)\"$(END)\n"
	@printf "$(BLD)RELEASE_REGISTRY$(END)=$(BLU)\"$(RELEASE_REGISTRY)\"$(END)\n"
.PHONY: env

help:
	@printf "$(subst $(NL),\n,$(HELP))\n"
.PHONY: help

# NOTE: this is not a typo, this is actually how you spell newline in Make
define NL


endef

# NOTE: this is not a typo, this is actually how you spell space in Make
define SPACE
 
endef

COMMA = ,

define HELP
$(_help.intro)

$(BLD)Targets:$(END)

$(_help.targets)

$(BLD)Codebases:$(END)
  $(foreach MODULE,$(MODULES),\n  $(BLD)$(SOURCE_$(MODULE)) ==> $(BLU)$(MODULE)$(END))

endef

define _help.intro
This Makefile builds Ambassador using a standard build environment inside
a Docker container. The $(BLD)$(REPO)$(END), $(BLD)kat-server$(END), and $(BLD)kat-client$(END) images are
created from this container after the build stage is finished.

The build works by maintaining a running build container in the background.
It gets source code into that container via $(BLD)rsync$(END). The $(BLD)/root$(END) directory in
this container is a Docker volume, which allows files (e.g. the Go build
cache and $(BLD)pip$(END) downloads) to be cached across builds.

This arrangement also permits building multiple codebases. This is useful
for producing builds with extended functionality. Each external codebase
is synced into the container at the $(BLD)/buildroot/<name>$(END) path.

The build system doesn't try to magically handle all dependencies. In
general, if you change something that is not pure source code, you will
likely need to do a $(BLD)make clean$(END) in order to see the effect. For example,
Python code only gets set up once, so if you change $(BLD)requirements.txt$(END) or
$(BLD)setup.py$(END), then you will need to do a clean build to see the effects.
Assuming you didn't $(BLD)make clobber$(END), this shouldn't take long due to the
cache in the Docker volume.
endef

define _help.targets
  $(BLD)make $(BLU)help$(END)      -- displays this message.

  $(BLD)make $(BLU)env$(END)       -- display the value of important env vars.

  $(BLD)make $(BLU)preflight$(END) -- checks dependencies of this makefile.

  $(BLD)make $(BLU)sync$(END)      -- syncs source code into the build container.

  $(BLD)make $(BLU)version$(END)   -- display source code version.

  $(BLD)make $(BLU)compile$(END)   -- syncs and compiles the source code in the build container.

  $(BLD)make $(BLU)images$(END)    -- creates images from the build container.

  $(BLD)make $(BLU)push$(END)      -- pushes images to $(BLD)\$$DEV_REGISTRY$(END). ($(DEV_REGISTRY))

  $(BLD)make $(BLU)test$(END)      -- runs Go and Python tests inside the build container.

    The tests require a Kubernetes cluster and a Docker registry in order to
    function. These must be supplied via the $(BLD)make$(END)/$(BLD)env$(END) variables $(BLD)\$$DEV_KUBECONFIG$(END)
    and $(BLD)\$$DEV_REGISTRY$(END).

  $(BLD)make $(BLU)gotest$(END)    -- runs just the Go tests inside the build container.

    Use $(BLD)\$$GOTEST_PKGS$(END) to control which packages are passed to $(BLD)gotest$(END). ($(GOTEST_PKGS))
    Use $(BLD)\$$GOTEST_ARGS$(END) to supply additional non-package arguments. ($(GOTEST_ARGS))

  $(BLD)make $(BLU)pytest$(END)    -- runs just the Python tests inside the build container.

    Use $(BLD)\$$PYTEST_ARGS$(END) to pass args to pytest. ($(PYTEST_ARGS))

  $(BLD)make $(BLU)shell$(END)     -- starts a shell in the build container.

  $(BLD)make $(BLU)rc$(END)        -- push a release candidate image to $(BLD)\$$RELEASE_REGISTRY$(END). ($(RELEASE_REGISTRY))

    The current commit must be tagged for this to work, and your tree must be clean.
    If the tag is of the form 'vX.Y.Z-rc[0-9]*', this will also push a tag of the
    form 'vX.Y.Z-rc-latest'.

  $(BLD)make $(BLU)release$(END)   -- promote a release candidate to a release.

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z'. You must also have previously
    build an RC for the same tag using the current $(BLD)\$$RELEASE_REGISTRY$(END).

  $(BLD)make $(BLU)clean$(END)     -- kills the build container.

  $(BLD)make $(BLU)clobber$(END)   -- kills the build container and the cache volume.
endef
