
#DEV_REGISTRY=localhost:5000
#DEV_KUBECONFIG=/tmp/k3s.yaml

RED=\033[1;31m
GRN=\033[1;32m
YEL=\033[1;33m
BLU=\033[1;34m
WHT=\033[1;37m
BLD=\033[1m
END=\033[0m

MODULES :=

module = $(eval MODULES += $(1))$(eval SOURCE_$(1)=$(abspath $(2)))

BUILDER_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

BUILDER = $(abspath $(BUILDER_HOME)/builder.sh)
DBUILD = $(abspath $(BUILDER_HOME)/dbuild.sh)

all: help
.PHONY: all

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

sync: preflight base-envoy.docker
	@$(foreach MODULE,$(MODULES),$(BUILDER) sync $(MODULE) $(SOURCE_$(MODULE)) &&) true
.PHONY: sync

compile: sync
	@$(BUILDER) compile $(SOURCES)
.PHONY: compile

commit:
	@$(BUILDER) commit snapshot
.PHONY: commit

# Docker images that are built from the unified ./builder/Dockerfile
images.builder = $(shell sed -n '/\#external/{N;s/.* as  *//p;}' < $(BUILDER_HOME)/Dockerfile)

images.all += $(images.builder)
images.cluster += $(images.builder)

images: $(addsuffix .docker.tag.dev,$(images.all))
.PHONY: images
snapshot.docker.stamp: compile
	@$(MAKE) --no-print-directory commit
	@docker image inspect snapshot --format='{{.Id}}' > $@
$(addsuffix .docker.stamp,$(images.builder)): %.docker.stamp: snapshot.docker base-envoy.docker
	@printf "$(WHT)==$(GRN)Building $(BLU)$*$(GRN) image$(WHT)==$(END)\n"
	@$(DBUILD) $(BUILDER_HOME) --iidfile $@ --build-arg artifacts=$$(cat snapshot.docker) --build-arg envoy=$$(cat base-envoy.docker) --target $*
%.docker: %.docker.stamp $(COPY_IFCHANGED)
	@$(COPY_IFCHANGED) $< $@
# As a special case, don't enforce the "can't change in CI" rule for
# snapshot.docker, since `docker commit` will bump timestamps.
snapshot.docker: %.docker: %.docker.stamp $(COPY_IFCHANGED)
	@CI= $(COPY_IFCHANGED) $< $@
# Fricking frick, the __pycache__ and .egg files aren't staying the
# same.  Just take off the seat-belt for now, we need to get a release
# out.
ambassador.docker: %.docker: %.docker.stamp $(COPY_IFCHANGED)
	@CI= $(COPY_IFCHANGED) $< $@

define REGISTRY_ERR
$(shell printf '$(RED)ERROR: please set the DEV_REGISTRY make/env variable to the docker registry\n       you would like to use for development$(END)\n' >&2)
$(error error)
endef

push: $(addsuffix .docker.push.dev,$(images.cluster))
.PHONY: push

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
.PHONY: test-ready

PYTEST_ARGS ?=

pytest: test-ready
	@printf "$(WHT)==$(GRN)Running $(BLU)py$(GRN) tests$(WHT)==$(END)\n"
	docker exec \
		-e AMBASSADOR_DOCKER_IMAGE=$$(sed -n 2p ambassador.docker.push.dev) \
		-e KAT_CLIENT_DOCKER_IMAGE=$$(sed -n 2p kat-client.docker.push.dev) \
		-e KAT_SERVER_DOCKER_IMAGE=$$(sed -n 2p kat-server.docker.push.dev) \
		-e TEST_SERVICE_AUTH=$$(sed -n 2p test-auth.docker.push.dev) \
		-e TEST_SERVICE_AUTH_TLS=$$(sed -n 2p test-auth-tls.docker.push.dev) \
		-e TEST_SERVICE_RATELIMIT=$$(sed -n 2p test-ratelimit.docker.push.dev) \
		-e TEST_SERVICE_SHADOW=$$(sed -n 2p test-shadow.docker.push.dev) \
		-e TEST_SERVICE_STATS=$$(sed -n 2p test-stats.docker.push.dev) \
		-e KAT_IMAGE_PULL_POLICY=Always \
		-e KAT_REQ_LIMIT \
		-it $(shell $(BUILDER)) sh -c 'cd ambassador && pytest -ra $(PYTEST_ARGS)'
.PHONY: pytest


GOTEST_PKGS ?= ./...
GOTEST_ARGS ?=

gotest: test-ready
	@printf "$(WHT)==$(GRN)Running $(BLU)go$(GRN) tests$(WHT)==$(END)\n"
	docker exec -w /buildroot/$(MODULE) -e DTEST_REGISTRY=$(DEV_REGISTRY) -e DTEST_KUBECONFIG=/buildroot/kubeconfig.yaml -e GOTEST_PKGS=$(GOTEST_PKGS) -e GOTEST_ARGS=$(GOTEST_ARGS) $(shell $(BUILDER)) /buildroot/builder.sh test-internal
.PHONY: gotest

test: gotest pytest
.PHONY: test

shell:
	@$(BUILDER) shell
.PHONY: shell

clean: $(addsuffix .docker.clean,$(images.all) snapshot)
	@$(BUILDER) clean
.PHONY: clean

clobber: clean
	@$(BUILDER) clobber
.PHONY: clobber

help:
	@printf "$(subst $(NL),\n,$(HELP))\n"
.PHONY: help

define NL


endef

define HELP

This Makefile builds Ambassador using a standard build environment inside
a Docker container. The $(BLD)ambassador$(END), $(BLD)kat-server$(END), and $(BLD)kat-client$(END) images are
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

$(BLD)Targets:$(END)

  $(BLD)make $(BLU)help$(END)      -- displays this message.

  $(BLD)make $(BLU)preflight$(END) -- checks dependencies of this makefile.

  $(BLD)make $(BLU)sync$(END)      -- syncs source code into the build container.

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

  $(BLD)make $(BLU)clean$(END)     -- kills the build container.

  $(BLD)make $(BLU)clobber$(END)   -- kills the build container and the cache volume.

$(BLD)Codebases:$(END)
  $(foreach MODULE,$(MODULES),\n  $(BLD)$(SOURCE_$(MODULE)) ==> $(BLU)$(MODULE)$(END))

endef
