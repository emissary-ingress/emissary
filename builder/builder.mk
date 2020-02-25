
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

BUILDER_NAME ?= $(NAME)

BUILDER = BUILDER_NAME=$(BUILDER_NAME) $(abspath $(BUILDER_HOME)/builder.sh)
DBUILD = $(abspath $(BUILDER_HOME)/dbuild.sh)

all: help
.PHONY: all

.NOTPARALLEL:

export RSYNC_ERR=$(RED)ERROR: please update to a version of rsync with the --info option$(END)
export DOCKER_ERR=$(RED)ERROR: cannot find docker, please make sure docker is installed$(END)

# the name of the Docker network
# note: use your local k3d/microk8s/kind network for running tests
DOCKER_NETWORK ?= $(BUILDER_NAME)

preflight:
ifeq ($(strip $(shell $(BUILDER))),)
	@printf "$(CYN)==> $(GRN)Preflight checks$(END)\n"
# Checking for rsync --info
	test -n "$$(rsync --help | fgrep -- --info)" || (printf "$${RSYNC_ERR}\n"; exit 1)
# Checking for docker
	which docker > /dev/null || (printf "$${DOCKER_ERR}\n"; exit 1)
endif
.PHONY: preflight

preflight-cluster:
	@test -n "$(DEV_KUBECONFIG)" || (printf "$${KUBECONFIG_ERR}\n"; exit 1)
	@if [ "$(DEV_KUBECONFIG)" != '-skip-for-release-' ]; then \
		printf "$(CYN)==> $(GRN)Checking for test cluster$(END)\n" ;\
		kubectl --kubeconfig $(DEV_KUBECONFIG) -n default get service kubernetes > /dev/null || { printf "$${KUBECTL_ERR}\n"; exit 1; } ;\
	else \
		printf "$(CYN)==> $(RED)Skipping test cluster checks$(END)\n" ;\
	fi
.PHONY: preflight-cluster

sync: preflight
	@$(foreach MODULE,$(MODULES),$(BUILDER) sync $(MODULE) $(SOURCE_$(MODULE)) &&) true
	@if [ -n "$(DEV_KUBECONFIG)" ] && [ "$(DEV_KUBECONFIG)" != '-skip-for-release-' ]; then \
		kubectl --kubeconfig $(DEV_KUBECONFIG) config view --flatten | docker exec -i $$($(BUILDER)) sh -c "cat > /buildroot/kubeconfig.yaml" ;\
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

SNAPSHOT=snapshot-$(BUILDER_NAME)

commit:
	@$(BUILDER) commit $(SNAPSHOT)
.PHONY: commit

REPO=$(BUILDER_NAME)

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

test-ready: push preflight-cluster
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

pytest-only: sync preflight-cluster
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) tests$(END)\n"
	docker exec \
		-e AMBASSADOR_DOCKER_IMAGE=$(AMB_IMAGE) \
		-e KAT_CLIENT_DOCKER_IMAGE=$(KAT_CLI_IMAGE) \
		-e KAT_SERVER_DOCKER_IMAGE=$(KAT_SRV_IMAGE) \
		-e KAT_IMAGE_PULL_POLICY=Always \
		-e DOCKER_NETWORK=$(DOCKER_NETWORK) \
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
	docker exec \
		-w /buildroot/ambassador \
		-e GOOS=windows \
		-it $(shell $(BUILDER)) go build -o /dev/null ./cmd/edgectl
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

# 'rc' is a deprecated alias for 'release/bits', kept around for the
# moment to avoid pain with needing to update apro.git in lockstep.
rc: release/bits
.PHONY: rc

release/bits: images
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@printf "$(CYN)==> $(GRN)Pushing $(BLU)$(REPO)$(GRN) Docker image$(END)\n"
	docker tag $(REPO) $(AMB_IMAGE_RC)
	docker push $(AMB_IMAGE_RC)
.PHONY: release/bits

release/promote-oss/.main:
	@[[ "$(RELEASE_VERSION)"      =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]
	@[[ '$(PROMOTE_FROM_VERSION)' =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]
	@[[ '$(PROMOTE_TO_VERSION)'   =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]
	@case "$(PROMOTE_CHANNEL)" in \
		""|early|test) true ;; \
		*) echo "Unknown PROMOTE_CHANNEL $(PROMOTE_CHANNEL)" >&2 ; exit 1;; \
	esac
	@printf "$(CYN)==> $(GRN)Promoting $(BLU)%s$(GRN) to $(BLU)%s$(GRN) (channel=$(BLU)%s$(GRN))$(END)\n" '$(PROMOTE_FROM_VERSION)' '$(PROMOTE_TO_VERSION)' '$(PROMOTE_CHANNEL)'

	@printf '  $(CYN)$(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION)$(END)\n'
	docker pull $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION)
	docker tag $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION) $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_TO_VERSION)
	docker push $(RELEASE_REGISTRY)/$(REPO):$(PROMOTE_TO_VERSION)

	@printf '  $(CYN)https://s3.amazonaws.com/datawire-static-files/ambassador/$(PROMOTE_CHANNEL)stable.txt$(END)\n'
	printf '%s' "$(RELEASE_VERSION)" | aws s3 cp - s3://datawire-static-files/ambassador/$(PROMOTE_CHANNEL)stable.txt

	@printf '  $(CYN)s3://scout-datawire-io/ambassador/$(PROMOTE_CHANNEL)app.json$(END)\n'
	printf '{"application":"ambassador","latest_version":"%s","notices":[]}' "$(RELEASE_VERSION)" | aws s3 cp - s3://scout-datawire-io/ambassador/$(PROMOTE_CHANNEL)app.json
.PHONY: release/promote-oss/.main

# To be run from a checkout at the tag you are promoting _from_.
# At present, this is to be run by-hand.
release/promote-oss/to-ea-latest:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-ea\.[0-9]+$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an EA tag\n' "$(RELEASE_VERSION)"; exit 1)
	@{ $(MAKE) release/promote-oss/.main \
	  PROMOTE_FROM_VERSION="$(RELEASE_VERSION)" \
	  PROMOTE_TO_VERSION="$$(echo "$(RELEASE_VERSION)" | sed 's/-ea.*/-ea-latest/')" \
	  PROMOTE_CHANNEL=early \
	; }
.PHONY: release/promote-oss/to-ea-latest

# To be run from a checkout at the tag you are promoting _from_.
# At present, this is to be run by-hand.
release/promote-oss/to-rc-latest:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an RC tag\n' "$(RELEASE_VERSION)"; exit 1)
	@{ $(MAKE) release/promote-oss/.main \
	  PROMOTE_FROM_VERSION="$(RELEASE_VERSION)" \
	  PROMOTE_TO_VERSION="$$(echo "$(RELEASE_VERSION)" | sed 's/-rc.*/-rc-latest/')" \
	  PROMOTE_CHANNEL=test \
	; }
.PHONY: release/promote-oss/to-rc-latest

# To be run from a checkout at the tag you are promoting _to_.
# This is normally run from CI by creating the GA tag.
release/promote-oss/to-ga:
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(RELEASE_VERSION)"; exit 1)
	@set -e; { \
	  rc_latest=$$(curl -sL --fail https://s3.amazonaws.com/datawire-static-files/ambassador/teststable.txt); \
	  if ! [[ "$$rc_latest" == "$(RELEASE_VERSION)"-rc.* ]]; then \
	    printf '$(RED)ERROR: https://s3.amazonaws.com/datawire-static-files/ambassador/teststable.txt => %s does not look like a RC of %s\n' "$$rc_latest" "$(RELEASE_VERSION)"; \
	    exit 1; \
	  fi; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$rc_latest" \
	    PROMOTE_TO_VERSION="$(RELEASE_VERSION)" \
	    PROMOTE_CHANNEL='' \
	    ; \
	}
.PHONY: release/promote-oss/to-ga

release-prep:
	bash $(OSS_HOME)/releng/release-prep.sh
.PHONY: release-prep

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
	@printf "$(BLD)AMBASSADOR_DOCKER_IMAGE$(END)=$(BLU)\"$(AMB_IMAGE)\"$(END)\n"
	@printf "$(BLD)KAT_CLIENT_DOCKER_IMAGE$(END)=$(BLU)\"$(KAT_CLI_IMAGE)\"$(END)\n"
	@printf "$(BLD)KAT_SERVER_DOCKER_IMAGE$(END)=$(BLU)\"$(KAT_SRV_IMAGE)\"$(END)\n"
.PHONY: env

export:
	@printf "export DEV_KUBECONFIG=\"$(DEV_KUBECONFIG)\"\n"
	@printf "export DEV_REGISTRY=\"$(DEV_REGISTRY)\"\n"
	@printf "export RELEASE_REGISTRY=\"$(RELEASE_REGISTRY)\"\n"
	@printf "export AMBASSADOR_DOCKER_IMAGE=\"$(AMB_IMAGE)\"\n"
	@printf "export KAT_CLIENT_DOCKER_IMAGE=\"$(KAT_CLI_IMAGE)\"\n"
	@printf "export KAT_SERVER_DOCKER_IMAGE=\"$(KAT_SRV_IMAGE)\"\n"
.PHONY: export

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
    Example: $(BLD)make gotest GOTEST_PKGS=./cmd/edgectl GOTEST_ARGS=-v$(END)  # run edgectl tests verbosely

  $(BLD)make $(BLU)pytest$(END)    -- runs just the Python tests inside the build container.

    Use $(BLD)\$$PYTEST_ARGS$(END) to pass args to $(BLD)pytest$(END). ($(PYTEST_ARGS))
    Example: $(BLD)make pytest PYTEST_ARGS=\"-k schemas\"$(END)  # run only tests with \"schemas\" in the name

  $(BLD)make $(BLU)shell$(END)     -- starts a shell in the build container.

  $(BLD)make $(BLU)release/bits$(END) -- do the 'push some bits' part of a release

    The current commit must be tagged for this to work, and your tree must be clean.
    If the tag is of the form 'vX.Y.Z-(ea|rc).[0-9]*'.

  $(BLD)make $(BLU)release/promote-oss/to-ea-latest$(END) -- promote an early-access '-ea.N' release to '-ea-latest'

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z-ea.N'. You must also have previously
    built an EA for the same tag using $(BLD)release/bits$(END).

  $(BLD)make $(BLU)release/promote-oss/to-rc-latest$(END) -- promote a release candidate '-rc.N' release to '-rc-latest'

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z-rc.N'. You must also have previously
    built an RC for the same tag using $(BLD)release/bits$(END).

  $(BLD)make $(BLU)release/promote-oss/to-ga$(END) -- promote a release candidate to general availability

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z'. You must also have previously
    built and promoted the RC that will become GA, using $(BLD)release/bits$(END) and
    $(BLD)release/promote-oss/to-rc-latest$(END).

  $(BLD)make $(BLU)clean$(END)     -- kills the build container.

  $(BLD)make $(BLU)clobber$(END)   -- kills the build container and the cache volume.
endef
