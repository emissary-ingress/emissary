# A quick primer on GNU Make syntax
# =================================
#
# This tries to cover the syntax that is hard to ctrl-f for in
# <https://www.gnu.org/software/make/manual/make.html> (err, hard to
# C-s for in `M-: (info "Make")`).
#
#   At the core is a "rule":
#
#       target: dependency1 dependency2
#       	command to run
#
#   If `target` something that isn't a real file (like 'build', 'lint', or
#   'test'), then it should be marked as "phony":
#
#       target: dependency1 dependency2
#       	command to run
#       .PHONY: target
#
#   You can write reusable "pattern" rules:
#
#       %.o: %.c
#       	command to run
#
#   Of course, if you don't have variables for the inputs and outputs,
#   it's hard to write a "command to run" for a pattern rule.  The
#   variables that you should know are:
#
#       $@ = the target
#       $^ = the list of dependencies (space separated)
#       $< = the first (left-most) dependency
#       $* = the value of the % glob in a pattern rule
#
#       Each of these have $(@D) and $(@F) variants that are the
#       directory-part and file-part of each value, respectively.
#
#       I think those are easy enough to remember mnemonically:
#         - $@ is where you shoul direct the output at.
#         - $^ points up at the dependency list
#         - $< points at the left-most member of the dependency list
#         - $* is the % glob; "*" is well-known as the glob char in other languages
#
#   Make will do its best to guess whether to apply a pattern rule for a
#   given file.  Or, you can explicitly tell it by using a 3-field
#   (2-colon) version:
#
#       foo.o bar.o: %.o: %.c
#       	command to run
#
#   In a non-pattern rule, if there are multiple targets listed, then it
#   is as if rule were duplicated for each target:
#
#       target1 target2: deps
#       	command to run
#
#       # is the same as
#
#       target1: deps
#       	command to run
#       target2: deps
#       	command to run
#
#   Because of this, if you have a command that generates multiple,
#   outputs, it _must_ be a pattern rule:
#
#       %.c %.h: %.y
#       	command to run
#
#   Normally, Make crawls the entire tree of dependencies, updating a file
#   if any of its dependencies have been updated.  There's a really poorly
#   named feature called "order-only" dependencies:
#
#       target: normal-deps | order-only-deps
#
#   Dependencies after the "|" are created if they don't exist, but if
#   they already exist, then don't bother updating them.
#
# Tips:
# -----
#
#  - Use absolute filenames.  It's dumb, but it really does result in
#    fewer headaches.  Use $(OSS_HOME) and $(AES_HOME) to spell the
#    absolute filenames.
#
#  - If you have a multiple-output command where the output files have
#    dissimilar names, have % be just the directory (the above tip makes
#    this easier).
#
#  - It can be useful to use the 2-colon form of a pattern rule when
#    writing a rule for just one file; it lets you use % and $* to avoid
#    repeating yourself, which can be especially useful with long
#    filenames.

BUILDER_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

LCNAME := $(shell echo $(NAME) | tr '[:upper:]' '[:lower:]')
BUILDER_NAME ?= $(LCNAME)

_check.DEV_REGISTRY = $(if $(DEV_REGISTRY),,$(error $(_check.DEV_REGISTRY.err)))
_check.DEV_REGISTRY.err  = $(RED)
_check.DEV_REGISTRY.err += $(NL)ERROR: please set the DEV_REGISTRY make/env variable to the docker registry
_check.DEV_REGISTRY.err += $(NL)       you would like to use for development
_check.DEV_REGISTRY.err += $(END)

.DEFAULT_GOAL = all
include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux/colors.mk

docker.tag.local = $(BUILDER_NAME).local/$(*F)
docker.tag.remote-devloop = $(_check.DEV_REGISTRY)$(DEV_REGISTRY)/$(*F):$(shell docker image inspect --format='{{slice (index (split .Id ":") 1) 0 12}}' $$(cat $<))
docker.tag.remote-cidev   = $(_check.DEV_REGISTRY)$(DEV_REGISTRY)/$(*F):$(subst +,-,$(BUILD_VERSION))
include $(OSS_HOME)/build-aux/docker.mk

MODULES :=

module = $(eval MODULES += $(1))$(eval SOURCE_$(1)=$(abspath $(2)))

BUILDER = BUILDER_NAME=$(BUILDER_NAME) $(abspath $(BUILDER_HOME)/builder.sh)

AWS_S3_BUCKET ?= datawire-static-files

GCR_RELEASE_REGISTRY ?= gcr.io/datawire

all: help
.PHONY: all

.NOTPARALLEL:

noop:
	@true
.PHONY: noop

RSYNC_ERR  = $(RED)ERROR: please update to a version of rsync with the --info option$(END)
GO_ERR     = $(RED)ERROR: please update to go 1.13 or newer$(END)
DOCKER_ERR = $(RED)ERROR: please update to a version of docker built with Go 1.13 or newer$(END)

preflight:
	@printf "$(CYN)==> $(GRN)Preflight checks$(END)\n"

	@echo "Checking that 'rsync' is installed and is new enough to support '--info'"
	@$(if $(shell rsync --help 2>/dev/null | grep -F -- --info),,printf '%s\n' $(call quote.shell,$(RSYNC_ERR)))

	@echo "Checking that 'go' is installed and is 1.13 or later"
	@$(if $(call _prelude.go.VERSION.HAVE,1.13),,printf '%s\n' $(call quote.shell,$(GO_ERR)))

	@echo "Checking that 'docker' is installed and supports the 'slice' function for '--format'"
	@$(if $(and $(shell which docker 2>/dev/null),\
	            $(call _prelude.go.VERSION.ge,$(patsubst go%,%,$(lastword $(shell go version $$(which docker)))),1.13)),\
	      ,\
	      printf '%s\n' $(call quote.shell,$(DOCKER_ERR)))
.PHONY: preflight

preflight-cluster:
	@test -n "$(DEV_KUBECONFIG)" || (printf "$${KUBECONFIG_ERR}\n"; exit 1)
	@if [ "$(DEV_KUBECONFIG)" == '-skip-for-release-' ]; then \
		printf "$(CYN)==> $(RED)Skipping test cluster checks$(END)\n" ;\
	else \
		printf "$(CYN)==> $(GRN)Checking for test cluster$(END)\n" ;\
		success=; \
		for i in {1..5}; do \
			kubectl --kubeconfig $(DEV_KUBECONFIG) -n default get service kubernetes > /dev/null && success=true && break || sleep 15 ; \
		done; \
		if [ ! "$${success}" ] ; then { printf "$$KUBECTL_ERR\n" ; exit 1; } ; fi; \
	fi
.PHONY: preflight-cluster

sync: docker/container.txt
	@printf "${CYN}==> ${GRN}Syncing sources in to builder container${END}\n"
	@$(foreach MODULE,$(MODULES),$(BUILDER) sync $(MODULE) $(SOURCE_$(MODULE)) &&) true
	@if [ -n "$(DEV_KUBECONFIG)" ] && [ "$(DEV_KUBECONFIG)" != '-skip-for-release-' ]; then \
		kubectl --kubeconfig $(DEV_KUBECONFIG) config view --flatten | docker exec -i $$(cat $<) sh -c "cat > /buildroot/kubeconfig.yaml" ;\
	fi
	@if [ -e ~/.docker/config.json ]; then \
		cat ~/.docker/config.json | docker exec -i $$(cat $<) sh -c "mkdir -p /home/dw/.docker && cat > /home/dw/.docker/config.json" ; \
	fi
	@if [ -n "$(GCLOUD_CONFIG)" ]; then \
		printf "Copying gcloud config to builder container\n"; \
		docker cp $(GCLOUD_CONFIG) $$(cat $<):/home/dw/.config/; \
	fi
.PHONY: sync

builder:
	@$(BUILDER) builder
.PHONY: builder

version:
	@$(BUILDER) version
.PHONY: version

raw-version:
	@$(BUILDER) raw-version
.PHONY: raw-version

python/ambassador.version: $(tools/write-ifchanged) FORCE
	set -o pipefail; $(BUILDER) raw-version | $(tools/write-ifchanged) python/ambassador.version

compile: sync
	@$(BUILDER) compile
.PHONY: compile

# For files that should only-maybe update when the rule runs, put ".stamp" on
# the left-side of the ":", and just go ahead and update it within the rule.
#
# ".stamp" should NEVER appear in a dependency list (that is, it
# should never be on the right-side of the ":"), save for in this rule
# itself.
%: %.stamp $(tools/copy-ifchanged)
	@$(tools/copy-ifchanged) $< $@

# Give Make a hint about which pattern rules to apply.  Honestly, I'm
# not sure why Make isn't figuring it out on its own, but it isn't.
_images = builder-base base-envoy $(LCNAME) $(LCNAME)-ea kat-client kat-server
$(foreach i,$(_images), docker/$i.docker.tag.local          ): docker/%.docker.tag.local         : docker/%.docker
$(foreach i,$(_images), docker/$i.docker.tag.remote-devloop ): docker/%.docker.tag.remote-devloop: docker/%.docker
$(foreach i,$(_images), docker/$i.docker.tag.remote-cidev   ): docker/%.docker.tag.remote-cidev  : docker/%.docker

docker/builder-base.docker.stamp: FORCE preflight
	@printf "${CYN}==> ${GRN}Bootstrapping builder base image${END}\n"
	@$(BUILDER) build-builder-base >$@
docker/container.txt.stamp: %/container.txt.stamp: %/builder-base.docker.tag.local %/base-envoy.docker.tag.local FORCE
	@printf "${CYN}==> ${GRN}Bootstrapping builder container${END}\n"
	@($(BOOTSTRAP_EXTRAS) $(BUILDER) bootstrap > $@)

docker/base-envoy.docker.stamp: FORCE
	@set -e; { \
	  if docker image inspect $(ENVOY_DOCKER_TAG) --format='{{ .Id }}' >$@ 2>/dev/null; then \
	    printf "${CYN}==> ${GRN}Base Envoy image is already pulled${END}\n"; \
	  else \
	    printf "${CYN}==> ${GRN}Pulling base Envoy image${END}\n"; \
	    TIMEFORMAT="     (docker pull took %1R seconds)"; \
	    time docker pull $(ENVOY_DOCKER_TAG); \
	    unset TIMEFORMAT; \
	    docker image inspect $(ENVOY_DOCKER_TAG) --format='{{ .Id }}' >$@; \
	  fi; \
	}
docker/$(LCNAME).docker.stamp: %/$(LCNAME).docker.stamp: %/base-envoy.docker.tag.local %/builder-base.docker python/ambassador.version $(BUILDER_HOME)/Dockerfile $(tools/dsum) FORCE
	@printf "${CYN}==> ${GRN}Building image ${BLU}$(LCNAME)${END}\n"
	@printf "    ${BLU}envoy=$$(cat $*/base-envoy.docker)${END}\n"
	@printf "    ${BLU}builderbase=$$(cat $*/builder-base.docker)${END}\n"
	{ $(tools/dsum) '$(LCNAME) build' 3s \
	  docker build -f ${BUILDER_HOME}/Dockerfile . \
	    --build-arg=envoy="$$(cat $*/base-envoy.docker)" \
	    --build-arg=builderbase="$$(cat $*/builder-base.docker)" \
	    --build-arg=version="$(BUILD_VERSION)" \
	    --target=ambassador \
	    --iidfile=$@; }

docker/kat-client.docker.stamp: %/kat-client.docker.stamp: %/base-envoy.docker.tag.local %/builder-base.docker $(BUILDER_HOME)/Dockerfile $(tools/dsum) FORCE
	@printf "${CYN}==> ${GRN}Building image ${BLU}kat-client${END}\n"
	{ $(tools/dsum) 'kat-client build' 3s \
	  docker build -f ${BUILDER_HOME}/Dockerfile . \
	    --build-arg=envoy="$$(cat $*/base-envoy.docker)" \
	    --build-arg=builderbase="$$(cat $*/builder-base.docker)" \
	    --target=kat-client \
	    --iidfile=$@; }
docker/kat-server.docker.stamp: %/kat-server.docker.stamp: %/base-envoy.docker.tag.local %/builder-base.docker $(BUILDER_HOME)/Dockerfile $(tools/dsum) FORCE
	@printf "${CYN}==> ${GRN}Building image ${BLU}kat-server${END}\n"
	{ $(tools/dsum) 'kat-server build' 3s \
	  docker build -f ${BUILDER_HOME}/Dockerfile . \
	    --build-arg=envoy="$$(cat $*/base-envoy.docker)" \
	    --build-arg=builderbase="$$(cat $*/builder-base.docker)" \
	    --target=kat-server \
	    --iidfile=$@; }

REPO=$(BUILDER_NAME)

images: docker/$(LCNAME).docker.tag.local
images: docker/kat-client.docker.tag.local
images: docker/kat-server.docker.tag.local
.PHONY: images


push: docker/$(LCNAME).docker.push.remote-devloop
push: docker/kat-client.docker.push.remote-devloop
push: docker/kat-server.docker.push.remote-devloop
.PHONY: push

push-dev: docker/$(LCNAME).docker.tag.local
	$(if $(IS_DIRTY),$(error push-dev: tree must be clean))
	$(if $(findstring -dev,$(IS_DIRTY)),,$(error push-dev: BUILD_VERSION=$(BUILD_VERSION) is not a dev version))

	$(MAKE) docker/$(LCNAME).docker.push.remote-cidev
	@set -e; { \
		commit=$$(git rev-parse HEAD) ;\
		printf "$(CYN)==> $(GRN)recording $(BLU)$$commit$(GRN) => $(BLU)$$suffix$(GRN) in S3...$(END)\n" ;\
		echo "$$suffix" | aws s3 cp - s3://$(AWS_S3_BUCKET)/dev-builds/$$commit ;\
	}
ifneq ($(IS_PRIVATE),)
	@echo "push-dev: not pushing manifests because this is a private repo"
else
	$(MAKE) \
	  CHART_VERSION_SUFFIX=$$(echo $(BUILD_VERSION) | sed -e 's/^[^+-]*//' -e 's/\+/-/g') \
	  IMAGE_TAG=$${suffix} \
	  IMAGE_REPO="$(DEV_REGISTRY)/$(LCNAME)" \
	  chart-push-ci
	$(MAKE) update-yaml --always-make
	$(MAKE) VERSION_OVERRIDE=$$suffix push-manifests
	$(MAKE) clean-manifests
endif
.PHONY: push-dev

export KUBECONFIG_ERR=$(RED)ERROR: please set the $(BLU)DEV_KUBECONFIG$(RED) make/env variable to the cluster\n       you would like to use for development. Note this cluster must have access\n       to $(BLU)DEV_REGISTRY$(RED) (currently $(BLD)$(DEV_REGISTRY)$(END)$(RED))$(END)
export KUBECTL_ERR=$(RED)ERROR: preflight kubectl check failed$(END)

# Internal target for running a bash shell.
_bash:
	@PS1="\u:\w $$ " /bin/bash
.PHONY: _bash

# Internal runner target that executes an entrypoint after setting up the user's UID/GUID etc.
_runner:
	@printf "$(CYN)==>$(END) * Creating group $(BLU)$$INTERACTIVE_GROUP$(END) with GID $(BLU)$$INTERACTIVE_GID$(END)\n"
	@addgroup -g $$INTERACTIVE_GID $$INTERACTIVE_GROUP
	@printf "$(CYN)==>$(END) * Creating user $(BLU)$$INTERACTIVE_USER$(END) with UID $(BLU)$$INTERACTIVE_UID$(END)\n"
	@adduser -u $$INTERACTIVE_UID -G $$INTERACTIVE_GROUP $$INTERACTIVE_USER -D
	@printf "$(CYN)==>$(END) * Adding user $(BLU)$$INTERACTIVE_USER$(END) to $(BLU)/etc/sudoers$(END)\n"
	@echo "$$INTERACTIVE_USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers
	@printf "$(CYN)==>$(END) * Switching to user $(BLU)$$INTERACTIVE_USER$(END) with shell $(BLU)/bin/bash$(END)\n"
	@su -s /bin/bash $$INTERACTIVE_USER -c "$$ENTRYPOINT"
.PHONY: _runner

# This target is a convenience alias for running the _bash target.
docker/shell: docker/run/_bash
.PHONY: docker/shell

# This target runs any existing target inside of the builder base docker image.
docker/run/%: docker/builder-base.docker
	docker run --net=host \
		-e INTERACTIVE_UID=$$(id -u) \
		-e INTERACTIVE_GID=$$(id -g) \
		-e INTERACTIVE_USER=$$(id -u -n) \
		-e INTERACTIVE_GROUP=$$(id -g -n) \
		-e PYTEST_ARGS="$$PYTEST_ARGS" \
		-e AMBASSADOR_DOCKER_IMAGE="$$AMBASSADOR_DOCKER_IMAGE" \
		-e KAT_CLIENT_DOCKER_IMAGE="$$KAT_CLIENT_DOCKER_IMAGE" \
		-e KAT_SERVER_DOCKER_IMAGE="$$KAT_SERVER_DOCKER_IMAGE" \
		-e DEV_KUBECONFIG="$$DEV_KUBECONFIG" \
		-v /etc/resolv.conf:/etc/resolv.conf \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $${DEV_KUBECONFIG}:$${DEV_KUBECONFIG} \
		-v $${PWD}:$${PWD} \
		-it \
		--init \
		--cap-add=NET_ADMIN \
		--entrypoint /bin/bash \
		$$(cat docker/builder-base.docker) -c "cd $$PWD && ENTRYPOINT=make\ $* make --quiet _runner"

# Don't try running 'make shell' from within docker. That target already tries to run a builder shell.
# Instead, quietly define 'docker/run/shell' to be an alias for 'docker/shell'.
docker/run/shell:
	$(MAKE) --quiet docker/shell

setup-envoy: extract-bin-envoy

extract-bin-envoy: docker/base-envoy.docker.tag.local
	@mkdir -p $(OSS_HOME)/bin/
	@rm -f $(OSS_HOME)/bin/envoy
	@printf "Extracting envoy binary to $(OSS_HOME)/bin/envoy\n"
	@echo "#!/bin/bash" > $(OSS_HOME)/bin/envoy
	@echo "" >> $(OSS_HOME)/bin/envoy
	@echo "docker run -v $(OSS_HOME):$(OSS_HOME) -v /var/:/var/ -v /tmp/:/tmp/ -t --entrypoint /usr/local/bin/envoy-static-stripped $$(cat docker/base-envoy.docker) \"\$$@\"" >> $(OSS_HOME)/bin/envoy
	@chmod +x $(OSS_HOME)/bin/envoy
.PHONY: extract-bin-envoy

mypy-server-stop: sync
	test -t 1 && USE_TTY="-t"; docker exec -i ${USE_TTY} $(shell $(BUILDER)) /buildroot/builder.sh mypy-internal stop
.PHONY: mypy

mypy-server: sync
	 test -t 1 && USE_TTY="-t"; docker exec -i ${USE_TTY} $(shell $(BUILDER)) /buildroot/builder.sh mypy-internal start
.PHONY: mypy

mypy: mypy-server
	test -t 1 && USE_TTY="-t"; docker exec -i ${USE_TTY} $(shell $(BUILDER)) /buildroot/builder.sh mypy-internal check
.PHONY: mypy

create-venv:
	[[ -d $(OSS_HOME)/venv ]] || python3 -m venv $(OSS_HOME)/venv
.PHONY: create-venv

# If we're setting up within Alpine linux, make sure to pin pip and pip-tools
# to something that is still PEP517 compatible. This allows us to set _manylinux.py
# and convince pip to install prebuilt wheels. We do this because there's no good
# rust toolchain to build orjson within Alpine itself.
setup-venv:
	@set -e; { \
		if [ -f /etc/issue ] && grep "Alpine Linux" < /etc/issue ; then \
			pip3 install -U pip==20.2.4 pip-tools==5.3.1; \
			echo 'manylinux1_compatible = True' > venv/lib/python3.8/site-packages/_manylinux.py; \
			pip install orjson==3.3.1; \
			rm -f venv/lib/python3.8/site-packages/_manylinux.py; \
		else \
			pip install orjson==3.6.0; \
		fi; \
		pip install -r $(OSS_HOME)/builder/requirements.txt; \
		pip install -e $(OSS_HOME)/python; \
	}
.PHONY: setup-orjson

setup-diagd: create-venv
	. $(OSS_HOME)/venv/bin/activate && $(MAKE) setup-venv
.PHONY: setup-diagd

shell: docker/container.txt
	@printf "$(CYN)==> $(GRN)Launching interactive shell...$(END)\n"
	@$(BUILDER) shell
.PHONY: shell

AMB_IMAGE_RC=$(RELEASE_REGISTRY)/$(REPO):$(RELEASE_VERSION)
AMB_IMAGE_RELEASE=$(RELEASE_REGISTRY)/$(REPO):$(BUILD_VERSION)

export RELEASE_REGISTRY_ERR=$(RED)ERROR: please set the RELEASE_REGISTRY make/env variable to the docker registry\n       you would like to use for release$(END)

RELEASE_VERSION = $(shell $(BUILDER) release-version)
BUILD_VERSION   = $(shell $(BUILDER) version)
IS_DIRTY        = $(shell $(BUILDER) is-dirty)

# release/promote-oss/.main does the main reusable part of promoting a release:
#  - pull/promote+re-tag/push the Docker image:
#    $(DEV_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION) -> $(RELEASE_REGISTRY)/$(REPO):$(subst +,-,$(RELEASE_VERSION))
#  - update stable.txt
#  - update Metriton's app.json
#
# It is meant to be used by putting `$(MAKE) release/promote-oss/.main
# VARIABLES...` in the recipe of a more specific `release/promote-oss/*`
# rule (currently: `…/dev-to-rc`, `…/to-hotfix`, and `…/to-ga`).
#
# The variables that the calling rule needs to set are:
#  - PROMOTE_FROM_VERSION
#  - PROMOTE_CHANNEL: One of '' (GA), 'wip', 'early', 'test' (RC), or
#    'hotfix'.  Used to determine which stable.txt and app.json to
#    write to.
#
# Additionally, it also makes use of the following global variables:
#  - AWS_S3_BUCKET    (set from CI environment)
#  - DEV_REGISTRY     (set from CI environment)
#  - RELEASE_REGISTRY (set from CI environment)
#  - RELEASE_VERSION  (always set globally in the Makefile)
#  - REPO             (always set globally in the Makefile)
release/promote-oss/.main: $(tools/docker-promote)
# pre-flight
	@[[ "$(RELEASE_VERSION)"      =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]] || (echo "must set RELEASE_VERSION"; exit 1)
	@[[ -n "$(PROMOTE_FROM_VERSION)" ]] || (echo "must set PROMOTE_FROM_VERSION"; exit 1)
	@case "$(PROMOTE_CHANNEL)" in \
	  ""|wip|early|test|hotfix) true;; \
	  *) echo "Unknown PROMOTE_CHANNEL $(PROMOTE_CHANNEL)" >&2 ; exit 1;; \
	esac
	@printf "$(CYN)==> $(GRN)Promoting $(BLU)%s$(GRN) to $(BLU)%s$(GRN) (channel=$(BLU)%s$(GRN))$(END)\n" '$(PROMOTE_FROM_VERSION)' '$(RELEASE_VERSION)' '$(PROMOTE_CHANNEL)'
# pull/re-tag/push the Docker image
	@printf '  $(CYN)$(DEV_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION)$(END)\n'
	$(tools/docker-promote) $(DEV_REGISTRY)/$(REPO):$(PROMOTE_FROM_VERSION) $(RELEASE_REGISTRY)/$(REPO):$(subst +,-,$(RELEASE_VERSION))
	docker push $(RELEASE_REGISTRY)/$(REPO):$(subst +,-,$(RELEASE_VERSION))
# update stable.txt
	@printf '  $(CYN)https://s3.amazonaws.com/$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt$(END)\n'
	printf '%s' "$(RELEASE_VERSION)" | aws s3 cp - s3://$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt
# update app.json
	@printf '  $(CYN)s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json$(END)\n'
	printf '{"application":"emissary","latest_version":"%s","notices":[]}' "$(RELEASE_VERSION)" | aws s3 cp - s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json
.PHONY: release/promote-oss/.main

# release/promote-oss/dev-to-rc promotes a previously blessed
# ("promote-to-passed"ed) dev image from some other CI run to be an
# RC.
release/promote-oss/dev-to-rc:
	@[[ ( "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$$ ) || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an RC tag\n' "$(RELEASE_VERSION)"; exit 1)
	{ $(MAKE) release/promote-oss/.to-rc-or-hf PROMOTE_CHANNEL=test PROMOTE_S3_KEY=dev-builds
.PHONY: release/promote-oss/dev-to-rc

release/promote-oss/.to-rc-or-hf:
# pre-flight
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@$(if $(IS_DIRTY),$(error $@: tree must be clean))
	$(OSS_HOME)/releng/release-wait-for-commit --commit $$(git rev-parse HEAD) --s3-key $(PROMOTE_S3_KEY)
# main (Docker, stable.txt, app.json)
	@PS4=; set -ex; { \
	  commit=$$(git rev-parse HEAD); \
	  dev_version=$$(aws s3 cp s3://$(AWS_S3_BUCKET)/$(PROMOTE_S3_KEY)/$$commit -); \
	  set +x; \
	  if [ -z "$$dev_version" ]; then \
	    printf "$(RED)==> found no passed dev version for $$commit in S3...$(END)\n"; \
	    exit 1; \
	  fi; \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN) for $(BLU)$$commit$(GRN) in S3...$(END)\n"; \
	  set -x; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_CHANNEL=$(PROMOTE_CHANNEL); \
	}
# Helm chart and Kubernetes manifests... (why is this so different than how it is done for GA?)
ifneq ($(IS_PRIVATE),)
	@echo "Not publishing charts or manifests because in a private repo"
else
	{ $(MAKE) chart-push-ci \
	  CHART_VERSION_SUFFIX=$$(echo $(RELEASE_VERSION) | sed 's/^[^-]*-/-/') \
	  IMAGE_TAG=$(RELEASE_VERSION) \
	  IMAGE_REPO=$(RELEASE_REGISTRY)/$(REPO); }
	$(MAKE) --always-make update-yaml
	$(MAKE) push-manifests    VERSION_OVERRIDE=$(RELEASE_VERSION)
	$(MAKE) publish-docs-yaml VERSION_OVERRIDE=$(RELEASE_VERSION)
	$(MAKE) clean-manifests
endif
.PHONY: release/promote-oss/.to-rc-or-hf

release/promote-oss/rc-update-apro:
	$(OSS_HOME)/releng/01-release-rc-update-apro v$(RELEASE_VERSION) v$(VERSIONS_YAML_VERSION)
.PHONY: release/promote-oss/rc-update-apro

release/print-test-artifacts:
	@set -e; { \
		manifest_ver=$(RELEASE_VERSION) ; \
		manifest_ver=$${manifest_ver%"-dirty"} ; \
		echo "export AMBASSADOR_MANIFEST_URL=https://app.getambassador.io/yaml/emissary/$$manifest_ver" ; \
		echo "export HELM_CHART_VERSION=`grep 'version' $(OSS_HOME)/charts/emissary-ingress/Chart.yaml | awk '{ print $$2 }'`" ; \
	}
.PHONY: release/print-test-artifacts

# just push the commit hash to s3
# this should only happen if all tests have passed at a certain commit
release/promote-oss/dev-to-passed-ci:
	@set -e; { \
		commit=$$(git rev-parse HEAD) ;\
		dev_version=$$(aws s3 cp s3://$(AWS_S3_BUCKET)/dev-builds/$$commit -) ;\
		if [ -z "$$dev_version" ]; then \
			printf "$(RED)==> found no dev version for $$commit in S3...$(END)\n" ;\
			exit 1 ;\
		fi ;\
		printf "$(CYN)==> $(GRN)Promoting $(BLU)$$commit$(GRN) => $(BLU)$$dev_version$(GRN) in S3...$(END)\n" ;\
		echo "$$dev_version" | aws s3 cp - s3://$(AWS_S3_BUCKET)/passed-builds/$$commit ;\
	}
.PHONY: release/promote-oss/dev-to-passed-ci

# should run on every PR once the builds have passed
# this is less strong than "release/promote-oss/dev-to-passed-ci"
release/promote-oss/pr-to-passed-ci:
	@set -e; { \
		commit=$$(git rev-parse HEAD) ;\
		dev_version=$$(aws s3 cp s3://$(AWS_S3_BUCKET)/dev-builds/$$commit -) ;\
		if [ -z "$$dev_version" ]; then \
			printf "$(RED)==> found no dev version for $$commit in S3...$(END)\n" ;\
			exit 1 ;\
		fi ;\
		printf "$(CYN)==> $(GRN)Promoting $(BLU)$$commit$(GRN) => $(BLU)$$dev_version$(GRN) in S3...$(END)\n" ;\
		echo "$$dev_version" | aws s3 cp - s3://$(AWS_S3_BUCKET)/passed-pr/$$commit ;\
	}
.PHONY: release/promote-oss/pr-to-passed-ci

# release/promote-oss/to-hotfix promotes a previously blessed
# ("promote-to-passed"ed) dev image from some other CI run to be a
# hotfix release.
#
# Unlike its siblings `release/promote-oss/dev-to-rc` and
# `release/promote-oss/to-ga`, this target is NOT currently run in CI,
# and is meant to be run on a developer's laptop.
#
# Before running `make release/promote-oss/to-hotfix`, you must:
#  - have the commit checked out
#  - have tagged the commit with a Git tag matching "*-hf*"
#  - have already had CI build and bless ("promote-to-passed") a dev image for the commit
#  - have Keybase access
#  - set the following variables:
#    + AWS_S3_BUCKET
#    + RELEASE_REGISTRY
release/promote-oss/to-hotfix:
	@$test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ ( "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+-hf\.[0-9]+\+[0-9]+$$ ) ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like an hotfix tag\n' "$(RELEASE_VERSION)"; exit 1)
	{ docker login \
	  -u $$(keybase fs read /keybase/team/datawireio/secrets/dockerhub.webui.d6eautomaton.username) \
	  -p $$(keybase fs read /keybase/team/datawireio/secrets/dockerhub.webui.d6eautomaton.password); }
	@PS4=; set -e; {
	  export AWS_ACCESS_KEY_ID=$$(    keybase fs read /keybase/team/datawireio/secrets/aws.s3-bot.cli-credentials | sed -n 's/aws_access_key_id=//p'    ); \
	  export AWS_SECRET_ACCESS_KEY=$$(keybase fs read /keybase/team/datawireio/secrets/aws.s3-bot.cli-credentials | sed -n 's/aws_secret_access_key=//p'); \
	  set -x; \
	  $(MAKE) release/promote-oss/.to-rc-or-hf PROMOTE_CHANNEL=hotfix PROMOTE_S3_KEY=passed-pr; \
	}
	docker logout
.PHONY: release/promote-oss/to-hotfix

# To be run from a checkout at the tag you are promoting _from_.
# This is normally run from CI by creating the GA tag.
release/promote-oss/to-ga:
# pre-flight
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(RELEASE_VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(RELEASE_VERSION)"; exit 1)
	$(OSS_HOME)/releng/release-wait-for-commit --commit $$(git rev-parse HEAD) --s3-key passed-builds
	@PS4=; set -ex; { \
	  commit=$$(git rev-parse HEAD); \
	  dev_version=$$(aws s3 cp s3://$(AWS_S3_BUCKET)/passed-builds/$$commit -); \
	  set +x; \
	  if [ -z "$$dev_version" ]; then \
	    printf "$(RED)==> found no passed dev version for $$commit in S3...$(END)\n"; \
	    exit 1; \
	  fi; \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN) for $(BLU)$$commit$(GRN) in S3...$(END)\n"; \
	  set -x; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_CHANNEL=; \
	}
.PHONY: release/promote-oss/to-ga

VERSIONS_YAML_VER := $(shell grep 'version:' $(OSS_HOME)/docs/yaml/versions.yml | awk '{ print $$2 }')
VERSIONS_YAML_VER_STRIPPED := $(subst -ea,,$(VERSIONS_YAML_VER))
RC_NUMBER ?= 0

release/prep-rc:
	@test -n "$(VERSIONS_YAML_VER)" || (printf "version not found in versions.yml\n"; exit 1)
	@test -n "$(RELEASE_REGISTRY)" || (printf "RELEASE_REGISTRY must be set\n"; exit 1)
	@[[ "$(VERSIONS_YAML_VER)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: Version in versions.yml %s does not look like a GA tag\n' "$(VERSIONS_YAML_VER)"; exit 1)
	@[[ -z "$(IS_DIRTY)" ]] || (printf '$(RED)ERROR: tree must be clean\n'; exit 1)
	@AWS_S3_BUCKET=$(AWS_S3_BUCKET) RELEASE_REGISTRY=$(RELEASE_REGISTRY) IMAGE_NAME=$(LCNAME) \
		$(OSS_HOME)/releng/01-release-prep-rc $(VERSIONS_YAML_VER_STRIPPED)-rc.$(RC_NUMBER)
.PHONY: release/prep-rc

release/go:
	@test -n "$(VERSIONS_YAML_VER)" || (printf "version not found in versions.yml\n"; exit 1)
	@test -n "$${RC_NUMBER}" || (printf "RC_NUMBER must be set.\n"; exit 1)
	@[[ "$(VERSIONS_YAML_VER)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(VERSIONS_YAML_VER)"; exit 1)
	@[[ -z "$(IS_DIRTY)" ]] || (printf '$(RED)ERROR: tree must be clean\n'; exit 1)
	@RELEASE_REGISTRY=$(RELEASE_REGISTRY) IMAGE_NAME=$(LCNAME) $(OSS_HOME)/releng/02-release-ga $(VERSIONS_YAML_VER)
.PHONY: release/go

release/manifests:
	@test -n "$(VERSIONS_YAML_VER)" || (printf "version not found in versions.yml\n"; exit 1)
	@[[ "$(VERSIONS_YAML_VER)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(VERSIONS_YAML_VER)"; exit 1)
	@$(OSS_HOME)/releng/release-manifest-image-update --oss-version $(VERSIONS_YAML_VER) --aes-version "$(AES_VERSION)"
.PHONY: release/manifests

release/repatriate:
	@$(OSS_HOME)/releng/release-repatriate $(VERSIONS_YAML_VER)
.PHONY: release/repatriate

release/ga-mirror:
	@test -n "$(VERSIONS_YAML_VER)" || (printf "$(RED)ERROR: version not found in versions.yml\n"; exit 1)
	@[[ "$(VERSIONS_YAML_VER)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(VERSIONS_YAML_VER)"; exit 1)
	@test -n "$(RELEASE_REGISTRY)" || (printf "$(RED)ERROR: RELEASE_REGISTRY not set\n"; exit 1)
	@$(OSS_HOME)/releng/release-mirror-images --ga-version $(VERSIONS_YAML_VER) --source-registry $(RELEASE_REGISTRY) --image-name $(LCNAME) --repo-list $(GCR_RELEASE_REGISTRY)

release/ga-check:
	@$(OSS_HOME)/releng/release-ga-check --ga-version $(VERSIONS_YAML_VER) --source-registry $(RELEASE_REGISTRY) --image-name $(LCNAME)

clean:
	@$(BUILDER) clean
.PHONY: clean

clobber:
	@$(BUILDER) clobber
.PHONY: clobber

AMBASSADOR_DOCKER_IMAGE = $(shell sed -n 2p docker/$(LCNAME).docker.push.remote-devloop 2>/dev/null)
export AMBASSADOR_DOCKER_IMAGE
KAT_CLIENT_DOCKER_IMAGE = $(shell sed -n 2p docker/kat-client.docker.push.remote-devloop 2>/dev/null)
export KAT_CLIENT_DOCKER_IMAGE
KAT_SERVER_DOCKER_IMAGE = $(shell sed -n 2p docker/kat-server.docker.push.remote-devloop 2>/dev/null)
export KAT_SERVER_DOCKER_IMAGE

_user-vars  = BUILDER_NAME
_user-vars += DEV_KUBECONFIG
_user-vars += DEV_REGISTRY
_user-vars += RELEASE_REGISTRY
_user-vars += AMBASSADOR_DOCKER_IMAGE
_user-vars += KAT_CLIENT_DOCKER_IMAGE
_user-vars += KAT_SERVER_DOCKER_IMAGE
env:
	@printf '$(BLD)%s$(END)=$(BLU)%s$(END)\n' $(foreach v,$(_user-vars), $v $(call quote.shell,$(call quote.shell,$($v))) )
.PHONY: env

export:
	@printf 'export %s=%s\n' $(foreach v,$(_user-vars), $v $(call quote.shell,$(call quote.shell,$($v))) )
.PHONY: export
