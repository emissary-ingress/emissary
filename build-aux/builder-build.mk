# TODO(lukeshu): Clean up and incorporate in to `build-aux/main.mk`.

include build-aux/prelude.mk
include build-aux/colors.mk

BUILDER_NAME ?= $(LCNAME)

# DOCKER_BUILDKIT is _required_ by our Dockerfile, since we use Dockerfile extensions for the
# Go build cache. See https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md.
export DOCKER_BUILDKIT := 1

all: help
.PHONY: all

.NOTPARALLEL:

GO_ERR     = $(RED)ERROR: please update to go 1.13 or newer$(END)
DOCKER_ERR = $(RED)ERROR: please update to a version of docker built with Go 1.13 or newer$(END)

preflight:
	@printf "$(CYN)==> $(GRN)Preflight checks$(END)\n"

	@echo "Checking that 'go' is installed and is 1.13 or later"
	@$(if $(call _prelude.go.VERSION.HAVE,1.13),,printf '%s\n' $(call quote.shell,$(GO_ERR)))

	@echo "Checking that 'docker' is installed and supports the 'slice' function for '--format'"
	@$(if $(and $(shell which docker 2>/dev/null),\
	            $(call _prelude.go.VERSION.ge,$(patsubst go%,%,$(lastword $(shell go version $$(which docker)))),1.13)),\
	      ,\
	      printf '%s\n' $(call quote.shell,$(DOCKER_ERR)))
.PHONY: preflight

python/ambassador.version: $(tools/write-ifchanged) FORCE
	set -e -o pipefail; { \
	  echo $(patsubst v%,%,$(VERSION)); \
	  git rev-parse HEAD; \
	} | $(tools/write-ifchanged) $@
clean: python/ambassador.version.rm

# Give Make a hint about which pattern rules to apply.  Honestly, I'm
# not sure why Make isn't figuring it out on its own, but it isn't.
_images = base-envoy $(LCNAME) kat-client kat-server
$(foreach i,$(_images), docker/$i.docker.tag.local  ): docker/%.docker.tag.local : docker/%.docker
$(foreach i,$(_images), docker/$i.docker.tag.remote ): docker/%.docker.tag.remote: docker/%.docker

docker/.base-envoy.docker.stamp: FORCE
	@set -e; { \
	  if docker image inspect $(ENVOY_DOCKER_TAG) --format='{{ .Id }}' >$@ 2>/dev/null; then \
	    printf "${CYN}==> ${GRN}Base Envoy image is already pulled${END}\n"; \
	  else \
	    printf "${CYN}==> ${GRN}Pulling base Envoy image${END}\n"; \
	    TIMEFORMAT="     (docker pull took %1R seconds)"; \
	    time docker pull $(ENVOY_DOCKER_TAG); \
	    unset TIMEFORMAT; \
	  fi; \
	  echo $(ENVOY_DOCKER_TAG) >$@; \
	}
clobber: docker/base-envoy.docker.clean

docker/.$(LCNAME).docker.stamp: %/.$(LCNAME).docker.stamp: \
  %/base.docker.tag.local \
  %/base-envoy.docker.tag.local \
  %/base-pip.docker.tag.local \
  python/ambassador.version \
  build-aux/Dockerfile \
  $(OSS_HOME)/build-aux/py-version.txt \
  $(tools/dsum) \
  vendor \
  FORCE
	@printf "${CYN}==> ${GRN}Building image ${BLU}$(LCNAME)${END}\n"
	@printf "    ${BLU}base=$$(sed -n 2p $*/base.docker.tag.local)${END}\n"
	@printf "    ${BLU}envoy=$$(cat $*/base-envoy.docker)${END}\n"
	@printf "    ${BLU}builderbase=$$(sed -n 2p $*/base-pip.docker.tag.local)${END}\n"
	{ $(tools/dsum) '$(LCNAME) build' 3s \
	  docker build -f build-aux/Dockerfile . \
			--platform="$(BUILD_ARCH)" \
	    --build-arg=base="$$(sed -n 2p $*/base.docker.tag.local)" \
	    --build-arg=envoy="$$(cat $*/base-envoy.docker)" \
	    --build-arg=builderbase="$$(sed -n 2p $*/base-pip.docker.tag.local)" \
	    --build-arg=py_version="$$(cat build-aux/py-version.txt)" \
	    --iidfile=$@; }
clean: docker/$(LCNAME).docker.clean

REPO=$(BUILDER_NAME)

images: docker/$(LCNAME).docker.tag.local
images: docker/kat-client.docker.tag.local
images: docker/kat-server.docker.tag.local
.PHONY: images

push: docker/$(LCNAME).docker.push.remote
push: docker/kat-client.docker.push.remote
push: docker/kat-server.docker.push.remote
.PHONY: push

# `make push-dev` is meant to be run by CI.
push-dev: docker/$(LCNAME).docker.tag.local
	@[[ '$(VERSION)' == *-* ]] || (echo "$(RED)$@: VERSION=$(VERSION) is not a pre-release version$(END)" >&2; exit 1)

	@printf '$(CYN)==> $(GRN)pushing $(BLU)%s$(GRN) as $(BLU)$(GRN)...$(END)\n' '$(LCNAME)' '$(DEV_REGISTRY)/$(LCNAME):$(patsubst v%,%,$(VERSION))'
	docker tag $$(cat docker/$(LCNAME).docker) '$(DEV_REGISTRY)/$(LCNAME):$(patsubst v%,%,$(VERSION))'
	docker push '$(DEV_REGISTRY)/$(LCNAME):$(patsubst v%,%,$(VERSION))'
.PHONY: push-dev

$(OSS_HOME)/venv: python/requirements.txt python/requirements-dev.txt
	rm -rf $@
	python3 -m venv $@
	$@/bin/pip3 install -r python/requirements.txt
	$@/bin/pip3 install -r python/requirements-dev.txt
	$@/bin/pip3 install -e $(OSS_HOME)/python
clobber: venv.rm-r
