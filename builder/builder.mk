
#DEV_REGISTRY=localhost:5000
#DEV_KUBECONFIG=/tmp/k3s.yaml

RED=\033[1;31m
GRN=\033[1;32m
YEL=\033[1;33m
BLU=\033[1;34m
WHT=\033[1;37m
END=\033[0m

MODULES :=

module = $(eval MODULES += $(1))$(eval SOURCE_$(1)=$(abspath $(2)))

BUILDER_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

BUILDER = $(abspath $(BUILDER_HOME)/builder.sh)
DBUILD = $(abspath $(BUILDER_HOME)/dbuild.sh)

all: help

.NOTPARALLEL:

export RSYNC_ERR=$(RED)ERROR: please update to a version of rsync with the --info option$(END)
export DOCKER_ERR=$(RED)ERROR: cannot find docker, please make sure docker is installed$(END)

preflight:
ifeq ($(strip $(shell $(BUILDER))),)
	@printf "$(WHT)==$(GRN)Preflight checks$(WHT)==$(END)\n"
	# Checking for rsync --info
	test -n "$$(rsync --help | fgrep -- --info)" || (printf "$${RSYNC_ERR}\n"; exit 1)
	# Checking for docker
	which docker > /dev/null || (printf "$${DOCKER_ERR}\n"; exit 1)
endif
.PHONY: preflight

sync: preflight
	@$(foreach MODULE,$(MODULES),$(BUILDER) sync $(MODULE) $(SOURCE_$(MODULE)) &&) true

compile:
	@$(MAKE) --no-print-directory sync
	@$(BUILDER) compile $(SOURCES)

commit:
	@$(BUILDER) commit snapshot

images:
	@$(MAKE) --no-print-directory compile
	@$(MAKE) --no-print-directory commit
	@printf "$(WHT)==$(GRN)Building $(BLU)ambassador$(GRN) image$(WHT)==$(END)\n"
	@$(DBUILD) $(BUILDER_HOME) --build-arg artifacts=snapshot --target ambassador -t ambassador
	@printf "$(WHT)==$(GRN)Building $(BLU)kat-client$(GRN) image$(WHT)==$(END)\n"
	@$(DBUILD) $(BUILDER_HOME) --build-arg artifacts=snapshot --target kat-client -t kat-client
	@printf "$(WHT)==$(GRN)Building $(BLU)kat-server$(GRN) image$(WHT)==$(END)\n"
	@$(DBUILD) $(BUILDER_HOME) --build-arg artifacts=snapshot --target kat-server -t kat-server

AMB_IMAGE=$(DEV_REGISTRY)/ambassador:$(shell docker images -q ambassador:latest)
KAT_CLI_IMAGE=$(DEV_REGISTRY)/kat-client:$(shell docker images -q kat-client:latest)
KAT_SRV_IMAGE=$(DEV_REGISTRY)/kat-server:$(shell docker images -q kat-server:latest)

export REGISTRY_ERR=$(RED)ERROR: please set the DEV_REGISTRY make/env variable to the docker registry\n       you would like to use for development$(END)

push: images
	@test -n "$(DEV_REGISTRY)" || (printf "$${REGISTRY_ERR}\n"; exit 1)
	@printf "$(WHT)==$(GRN)Pushing $(BLU)ambassador$(GRN) image$(WHT)==$(END)\n"
	docker tag ambassador $(AMB_IMAGE)
	docker push $(AMB_IMAGE)
	@printf "$(WHT)==$(GRN)Pushing $(BLU)kat-client$(GRN) image$(WHT)==$(END)\n"
	docker tag kat-client $(KAT_CLI_IMAGE)
	docker push $(KAT_CLI_IMAGE)
	@printf "$(WHT)==$(GRN)Pushing $(BLU)kat-server$(GRN) image$(WHT)==$(END)\n"
	docker tag kat-server $(KAT_SRV_IMAGE)
	docker push $(KAT_SRV_IMAGE)

export KUBECONFIG_ERR=$(RED)ERROR: please set the $(YEL)DEV_KUBECONFIG$(RED) make/env variable to the docker registry\n       you would like to use for development. Note this cluster must have access\n       to $(YEL)DEV_REGISTRY$(RED) ($(WHT)$(DEV_REGISTRY)$(RED))$(END)
export KUBECTL_ERR=$(RED)ERROR: preflight kubectl check failed$(END)

test-ready: push
	@test -n "$(DEV_KUBECONFIG)" || (printf "$${KUBECONFIG_ERR}\n"; exit 1)
	@kubectl --kubeconfig $(DEV_KUBECONFIG) -n default get service kubernetes > /dev/null || (printf "$${KUBECTL_ERR}\n"; exit 1)
	@cat $(DEV_KUBECONFIG) | docker exec -i $(shell $(BUILDER)) sh -c "cat > /buildroot/kubeconfig.yaml"
	@if [ -e ~/.docker/config.json ]; then \
		cat ~/.docker/config.json | docker exec -i $(shell $(BUILDER)) sh -c "mkdir -p /home/dw/.docker && cat > /home/dw/.docker/config.json" ; \
	fi
# XXX noop target for teleproxy tests
	@docker exec -w /buildroot/ambassador -i $(shell $(BUILDER)) sh -c "echo bin_linux_amd64/edgectl: > Makefile"
	@docker exec -w /buildroot/ambassador -i $(shell $(BUILDER)) sh -c "mkdir -p bin_linux_amd64"
	@docker exec -w /buildroot/ambassador -d $(shell $(BUILDER)) ln -s /buildroot/bin/edgectl /buildroot/ambassador/bin_linux_amd64/edgectl

PYTEST_ARGS ?=

pytest: test-ready
	@printf "$(WHT)==$(GRN)Running $(BLU)py$(GRN) tests$(WHT)==$(END)\n"
	docker exec \
		-e AMBASSADOR_DOCKER_IMAGE=$(AMB_IMAGE) \
		-e KAT_CLIENT_DOCKER_IMAGE=$(KAT_CLI_IMAGE) \
		-e KAT_SERVER_DOCKER_IMAGE=$(KAT_SRV_IMAGE) \
		-e KAT_IMAGE_PULL_POLICY=Always \
		-e KAT_REQ_LIMIT \
		-it $(shell $(BUILDER)) pytest $(PYTEST_ARGS)


GOTEST_PKGS ?= ./...
GOTEST_ARGS ?=

gotest: test-ready
	@printf "$(WHT)==$(GRN)Running $(BLU)go$(GRN) tests$(WHT)==$(END)\n"
	docker exec -w /buildroot/$(MODULE) -e DTEST_REGISTRY=$(DEV_REGISTRY) -e DTEST_KUBECONFIG=/buildroot/kubeconfig.yaml -e GOTEST_PKGS=$(GOTEST_PKGS) -e GOTEST_ARGS=$(GOTEST_ARGS) $(shell $(BUILDER)) /buildroot/builder.sh test-internal

test: gotest pytest

shell:
	@$(BUILDER) shell

clean:
	@$(BUILDER) clean

clobber:
	@$(BUILDER) clobber

help:
	@printf "$(subst $(NL),\n,$(HELP))\n"

define NL


endef

define HELP

This Makefile builds ambassador source code in a standard build environment
inside a docker container. It then creates the ambassdaor, kat-server, and
kat-client images from this container.

The build works by maintaining a running build container in the background.
It gets source code into that container via rsync. The $(WHT)/root$(END) directory in
this container is a docker volume. This allows files to be cached across
builds, e.g. go build caching and pip downloads.

This arrangement also permits building multiple codebases. This is
useful for producing builds with extended functionality. Each external
codebases is synced into the container at the $(WHT)/buildroot/<name>$(END) path.

The build system doesn't try to magically handle all dependencies. In general
if you change something that is not pure source code, you will likely need to
do a $(WHT)make clean$(END) in order to see the effect. For example, python code only
gets setup once, so if you change requirements.txt or setup.py, then do a clean
build. This shouldn't take $(WHT)that$(END) long because (assuming you didn't $(WHT)make clobber$(END))
the docker volume will cache all the downloaded golang and/or python packages.

$(WHT)Targets:$(END)

  $(WHT)make $(BLU)help$(END)      -- displays this message.

  $(WHT)make $(BLU)preflight$(END) -- checks dependencies of this makefile.

  $(WHT)make $(BLU)sync$(END)      -- syncs source code into the build container.

  $(WHT)make $(BLU)compile$(END)   -- syncs and compiles the source code in the build container.

  $(WHT)make $(BLU)images$(END)    -- creates images from the build container.

  $(WHT)make $(BLU)push$(END)      -- pushes images to DEV_REGISTRY. ($(DEV_REGISTRY))

  $(WHT)make $(BLU)test$(END)      -- runs go and python tests inside the build container.

    The tests require a kubernetes cluster and a docker registry in order to function. These
    must be supplied via the make/env variables DEV_KUBECONFIG and DEV_REGISTRY.

  $(WHT)make $(BLU)gotest$(END)    -- runs go tests inside the build container.

    Use $(YEL)GOTEST_PKGS$(END) to control which packages are passed to gotest. ($(GOTEST_PKGS))
    Use $(YEL)GOTEST_ARGS$(END) to supply additional non-package arguments. ($(GOTEST_ARGS))

  $(WHT)make $(BLU)pytest$(END)    -- runs python tests inside the build container.

    Use $(YEL)PYTEST_ARGS$(END) to pass args to pytest. ($(PYTEST_ARGS))

  $(WHT)make $(BLU)shell$(END)     -- starts a shell in the build container.

  $(WHT)make $(BLU)clean$(END)     -- kills the build container.

  $(WHT)make $(BLU)clobber$(END)   -- kills the build container and the cache volume.

$(WHT)Codebases:$(END)
  $(foreach MODULE,$(MODULES),\n  $(WHT)$(SOURCE_$(MODULE)) ==> $(BLU)$(MODULE)$(END))

endef
