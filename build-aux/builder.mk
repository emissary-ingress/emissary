BUILDER_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

LCNAME := $(shell echo $(NAME) | tr '[:upper:]' '[:lower:]')
BUILDER_NAME ?= $(LCNAME)

include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux/colors.mk

docker.tag.local = $(BUILDER_NAME).local/$(*F)
docker.tag.remote = $(if $(DEV_REGISTRY),,$(error $(REGISTRY_ERR)))$(DEV_REGISTRY)/$(*F):$(patsubst v%,%,$(VERSION))
include $(OSS_HOME)/build-aux/docker.mk

include $(OSS_HOME)/build-aux/teleproxy.mk

MODULES :=

module = $(eval MODULES += $(1))$(eval SOURCE_$(1)=$(abspath $(2)))

BUILDER = BUILDER_NAME=$(BUILDER_NAME) $(abspath $(BUILDER_HOME)/builder.sh)

AWS_S3_BUCKET ?= datawire-static-files

# the image used for running the Ingress v1 tests with KIND.
# the current, official image does not support Ingress v1, so we must build our own image with k8s 1.18.
# build this image with:
# 1. checkout the Kuberentes sources in a directory like "~/sources/kubernetes"
# 2. kind build node-image --kube-root ~/sources/kubernetes
# 3. docker tag kindest/node:latest docker.io/datawire/kindest-node:latest
# 4. docker push docker.io/datawire/kindest-node:latest
# This will not be necessary once the KIND images are built for a Kubernetes 1.18 and support Ingress v1beta1 improvements.
KIND_IMAGE ?= kindest/node:v1.18.0
#KIND_IMAGE ?= docker.io/datawire/kindest-node:latest
KIND_KUBECONFIG = /tmp/kind-kubeconfig

# The ingress conformance tests directory
# build this image with:
# 1. checkout https://github.com/kubernetes-sigs/ingress-controller-conformance
# 2. cd ingress-controller-conformance && make image
# 3. docker tag ingress-controller-conformance:latest docker.io/datawire/ingress-controller-conformance:latest
# 4. docker push docker.io/datawire/ingress-controller-conformance:latest
INGRESS_TEST_IMAGE ?= docker.io/datawire/ingress-controller-conformance:latest

# local ports for the Ingress conformance tests
INGRESS_TEST_LOCAL_PLAIN_PORT = 8000
INGRESS_TEST_LOCAL_TLS_PORT = 8443
INGRESS_TEST_LOCAL_ADMIN_PORT = 8877

# directory with the manifests for loading Ambassador for running the Ingress Conformance tests
# NOTE: these manifests can be slightly different to the regular ones asd they include
INGRESS_TEST_MANIF_DIR = $(BUILDER_HOME)/../manifests/emissary/
INGRESS_TEST_MANIFS = emissary-crds.yaml emissary-emissaryns.yaml

# DOCKER_BUILDKIT is _required_ by our Dockerfile, since we use Dockerfile extensions for the
# Go build cache. See https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md.
export DOCKER_BUILDKIT := 1

all: help
.PHONY: all

.NOTPARALLEL:

# the name of the Docker network
# note: use your local k3d/microk8s/kind network for running tests
DOCKER_NETWORK ?= $(BUILDER_NAME)

# local host IP address (and not 127.0.0.1)
ifneq ($(shell which ipconfig 2>/dev/null),)
  # macOS
  HOST_IP := $(shell ipconfig getifaddr $$(route get 1.1.1.1 | awk '/interface:/ {print $$2}'))
else ifneq ($(shell which ip 2>/dev/null),)
  # modern (iproute2) GNU/Linux
  #HOST_IP := $(shell ip --json route get to 1.1.1.1 | jq -r '.[0].prefsrc')
  HOST_IP := $(shell ip route get to 1.1.1.1 | sed -n '1s/.*src \([0-9.]\+\).*/\1/p')
else
  $(error I do not know how to get the host IP on this system; it has neither 'ipconfig' (macOS) nor 'ip' (modern GNU/Linux))
  # ...and I (lukeshu) couldn't figure out a good way to do it on old (net-tools) GNU/Linux.
endif

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

preflight-cluster: $(tools/kubectl)
	@test -n "$(DEV_KUBECONFIG)" || (printf "$${KUBECONFIG_ERR}\n"; exit 1)
	@if [ "$(DEV_KUBECONFIG)" == '-skip-for-release-' ]; then \
		printf "$(CYN)==> $(RED)Skipping test cluster checks$(END)\n" ;\
	else \
		printf "$(CYN)==> $(GRN)Checking for test cluster$(END)\n" ;\
		success=; \
		for i in {1..5}; do \
			$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) -n default get service kubernetes > /dev/null && success=true && break || sleep 15 ; \
		done; \
		if [ ! "$${success}" ] ; then { printf "$$KUBECTL_ERR\n" ; exit 1; } ; fi; \
	fi
.PHONY: preflight-cluster

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

docker/.$(LCNAME).docker.stamp: %/.$(LCNAME).docker.stamp: %/base.docker.tag.local %/base-envoy.docker.tag.local %/base-pip.docker.tag.local python/ambassador.version $(BUILDER_HOME)/Dockerfile $(OSS_HOME)/build-aux/py-version.txt $(tools/dsum) vendor FORCE
	@printf "${CYN}==> ${GRN}Building image ${BLU}$(LCNAME)${END}\n"
	@printf "    ${BLU}base=$$(sed -n 2p $*/base.docker.tag.local)${END}\n"
	@printf "    ${BLU}envoy=$$(cat $*/base-envoy.docker)${END}\n"
	@printf "    ${BLU}builderbase=$$(sed -n 2p $*/base-pip.docker.tag.local)${END}\n"
	{ $(tools/dsum) '$(LCNAME) build' 3s \
	  docker build -f ${BUILDER_HOME}/Dockerfile . \
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

REGISTRY_ERR  = $(RED)
REGISTRY_ERR += $(NL)ERROR: please set the DEV_REGISTRY make/env variable to the docker registry
REGISTRY_ERR += $(NL)       you would like to use for development
REGISTRY_ERR += $(END)

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

export KUBECONFIG_ERR=$(RED)ERROR: please set the $(BLU)DEV_KUBECONFIG$(RED) make/env variable to the cluster\n       you would like to use for development. Note this cluster must have access\n       to $(BLU)DEV_REGISTRY$(RED) (currently $(BLD)$(DEV_REGISTRY)$(END)$(RED))$(END)
export KUBECTL_ERR=$(RED)ERROR: preflight kubectl check failed$(END)

test-ready: push preflight-cluster
.PHONY: test-ready

PYTEST_ARGS ?=
export PYTEST_ARGS

pytest: push-pytest-images
pytest: $(tools/kubestatus)
pytest: $(tools/kubectl)
pytest: $(OSS_HOME)/venv
pytest: docker/base-envoy.docker.tag.local
pytest: proxy
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) tests$(END)\n"
	@echo "AMBASSADOR_DOCKER_IMAGE=$$AMBASSADOR_DOCKER_IMAGE"
	@echo "DEV_KUBECONFIG=$$DEV_KUBECONFIG"
	@echo "PYTEST_ARGS=$$PYTEST_ARGS"
	set -e; { \
	  . $(OSS_HOME)/venv/bin/activate; \
	  export SOURCE_ROOT=$(CURDIR); \
	  export ENVOY_DOCKER_TAG=$$(cat docker/base-envoy.docker); \
	  export KUBESTATUS_PATH=$(CURDIR)/tools/bin/kubestatus; \
	  pytest --tb=short -rP $(PYTEST_ARGS); \
	}
.PHONY: pytest

pytest-unit: $(OSS_HOME)/venv
pytest-unit: docker/base-envoy.docker.tag.local
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) unit tests$(END)\n"
	set -e; { \
	  . $(OSS_HOME)/venv/bin/activate; \
	  export SOURCE_ROOT=$(CURDIR); \
	  export ENVOY_DOCKER_TAG=$$(cat docker/base-envoy.docker); \
	  pytest --tb=short -rP $(PYTEST_ARGS) python/tests/unit; \
	}
.PHONY: pytest-unit

pytest-integration: push-pytest-images
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) integration tests$(END)\n"
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS python/tests/integration"
.PHONY: pytest-integration

pytest-kat: push-pytest-images # doing this all at once is too much for CI...
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS python/tests/kat"
# ... so we have a separate rule to run things split up
build-aux/.pytest-kat.txt.stamp: $(OSS_HOME)/venv push-pytest-images $(tools/kubectl) FORCE
	. venv/bin/activate && set -o pipefail && pytest --collect-only python/tests/kat 2>&1 | sed -En 's/.*<Function (.*)>/\1/p' | cut -d. -f1 | sort -u > $@
build-aux/pytest-kat.txt: build-aux/%: build-aux/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
clean: build-aux/.pytest-kat.txt.stamp.rm build-aux/pytest-kat.txt.rm
pytest-kat-%: build-aux/pytest-kat.txt $(tools/py-split-tests)
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS -k '$$($(tools/py-split-tests) $(subst -of-, ,$*) <build-aux/pytest-kat.txt)' python/tests/kat"

$(OSS_HOME)/venv: python/requirements.txt python/requirements-dev.txt
	rm -rf $@
	python3 -m venv $@
	$@/bin/pip3 install -r python/requirements.txt
	$@/bin/pip3 install -r python/requirements-dev.txt
	$@/bin/pip3 install -e $(OSS_HOME)/python
clobber: venv.rm-r

GOTEST_ARGS ?= -race -count=1 -timeout 30m
GOTEST_ARGS += -parallel=150 # The ./pkg/envoy-control-plane/cache/v{2,3}/ tests require high parallelism to reliably work
GOTEST_PKGS ?= ./...
gotest: $(OSS_HOME)/venv $(tools/kubectl)
	@printf "$(CYN)==> $(GRN)Running $(BLU)go$(GRN) tests$(END)\n"
	{ . $(OSS_HOME)/venv/bin/activate && \
	  export PATH=$(tools.bindir):$${PATH} && \
	  export EDGE_STACK=$(GOTEST_AES_ENABLED) && \
	  go test $(GOTEST_ARGS) $(GOTEST_PKGS); }
.PHONY: gotest

# Ingress v1 conformance tests, using KIND and the Ingress Conformance Tests suite.
ingresstest: $(tools/kubectl) | docker/$(LCNAME).docker.push.remote
	@printf "$(CYN)==> $(GRN)Running $(BLU)Ingress v1$(GRN) tests$(END)\n"
	@[ -n "$(INGRESS_TEST_IMAGE)" ] || { printf "$(RED)ERROR: no INGRESS_TEST_IMAGE defined$(END)\n"; exit 1; }
	@[ -n "$(INGRESS_TEST_MANIF_DIR)" ] || { printf "$(RED)ERROR: no INGRESS_TEST_MANIF_DIR defined$(END)\n"; exit 1; }
	@[ -d "$(INGRESS_TEST_MANIF_DIR)" ] || { printf "$(RED)ERROR: $(INGRESS_TEST_MANIF_DIR) does not seem a valid directory$(END)\n"; exit 1; }
	@[ -n "$(HOST_IP)" ] || { printf "$(RED)ERROR: no IP obtained for host$(END)\n"; ip addr ; exit 1; }

	@printf "$(CYN)==> $(GRN)Creating/recreating KIND cluster with image $(KIND_IMAGE)$(END)\n"
	@for i in {1..5} ; do \
		kind delete cluster 2>/dev/null || true ; \
		kind create cluster --image $(KIND_IMAGE) && break || sleep 10 ; \
	done

	@printf "$(CYN)==> $(GRN)Saving KUBECONFIG at $(KIND_KUBECONFIG)$(END)\n"
	@kind get kubeconfig > $(KIND_KUBECONFIG)
	@sleep 10

	@APISERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane` ; \
		[ -n "$$APISERVER_IP" ] || { printf "$(RED)ERROR: no IP obtained for API server$(END)\n"; docker ps ; docker inspect kind-control-plane ; exit 1; } ; \
		printf "$(CYN)==> $(GRN)API server at $$APISERVER_IP. Fixing server in $(KIND_KUBECONFIG).$(END)\n" ; \
		sed -i -e "s|server: .*|server: https://$$APISERVER_IP:6443|g" $(KIND_KUBECONFIG)

	@printf "$(CYN)==> $(GRN)Showing some cluster info:$(END)\n"
	@$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) cluster-info || { printf "$(RED)ERROR: kubernetes cluster not ready $(END)\n"; exit 1 ; }
	@$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) version || { printf "$(RED)ERROR: kubernetes cluster not ready $(END)\n"; exit 1 ; }

	@printf "$(CYN)==> $(GRN)Loading Ambassador (from the Ingress conformance tests) with image=$$(sed -n 2p docker/$(LCNAME).docker.push.remote)$(END)\n"
	@for f in $(INGRESS_TEST_MANIFS) ; do \
		printf "$(CYN)==> $(GRN)... $$f $(END)\n" ; \
		cat $(INGRESS_TEST_MANIF_DIR)/$$f | sed -e "s|image:.*ambassador\:.*|image: $$(sed -n 2p docker/$(LCNAME).docker.push.remote)|g" | tee /dev/tty | $(tools/kubectl) apply -f - ; \
	done

	@printf "$(CYN)==> $(GRN)Waiting for Ambassador to be ready$(END)\n"
	@$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) wait --for=condition=available --timeout=180s deployment/ambassador || { \
		printf "$(RED)ERROR: Ambassador was not ready after 3 mins $(END)\n"; \
		$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) get services --all-namespaces ; \
		exit 1 ; }

	@printf "$(CYN)==> $(GRN)Exposing Ambassador service$(END)\n"
	@$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) expose deployment ambassador --type=LoadBalancer --name=ambassador

	@printf "$(CYN)==> $(GRN)Starting the tests container (in the background)$(END)\n"
	@docker stop -t 3 ingress-tests 2>/dev/null || true && docker rm ingress-tests 2>/dev/null || true
	@docker run -d --rm --name ingress-tests -e KUBECONFIG=/opt/.kube/config --mount type=bind,source=$(KIND_KUBECONFIG),target=/opt/.kube/config \
		--entrypoint "/bin/sleep" $(INGRESS_TEST_IMAGE) 600

	@printf "$(CYN)==> $(GRN)Loading the Ingress conformance tests manifests$(END)\n"
	@docker exec -ti ingress-tests \
		/opt/ingress-controller-conformance apply --api-version=networking.k8s.io/v1beta1 --ingress-controller=getambassador.io/ingress-controller --ingress-class=ambassador
	@sleep 10

	@printf "$(CYN)==> $(GRN)Forwarding traffic to Ambassador service$(END)\n"
	@$(tools/kubectl) --kubeconfig=$(KIND_KUBECONFIG) port-forward --address=$(HOST_IP) svc/ambassador \
		$(INGRESS_TEST_LOCAL_PLAIN_PORT):8080 $(INGRESS_TEST_LOCAL_TLS_PORT):8443 $(INGRESS_TEST_LOCAL_ADMIN_PORT):8877 &
	@sleep 10

	@for url in "http://$(HOST_IP):$(INGRESS_TEST_LOCAL_PLAIN_PORT)" "https://$(HOST_IP):$(INGRESS_TEST_LOCAL_TLS_PORT)" "http://$(HOST_IP):$(INGRESS_TEST_LOCAL_ADMIN_PORT)/ambassador/v0/check_ready" ; do \
		printf "$(CYN)==> $(GRN)Waiting until $$url is ready...$(END)\n" ; \
		until curl --silent -k "$$url" ; do printf "$(CYN)==> $(GRN)... still waiting.$(END)\n" ; sleep 2 ; done ; \
		printf "$(CYN)==> $(GRN)... $$url seems to be ready.$(END)\n" ; \
	done
	@sleep 30

	@printf "$(CYN)==> $(GRN)Running the Ingress conformance tests against $(HOST_IP)$(END)\n"
	@docker exec -ti ingress-tests \
		/opt/ingress-controller-conformance verify \
			--api-version=networking.k8s.io/v1beta1 \
			--use-insecure-host=$(HOST_IP):$(INGRESS_TEST_LOCAL_PLAIN_PORT) \
			--use-secure-host=$(HOST_IP):$(INGRESS_TEST_LOCAL_TLS_PORT)

	@printf "$(CYN)==> $(GRN)Cleaning up...$(END)\n"
	-@pkill $(tools/kubectl) -9
	@docker stop -t 3 ingress-tests 2>/dev/null || true && docker rm ingress-tests 2>/dev/null || true

	@if [ -n "$(CLEANUP)" ] ; then \
		printf "$(CYN)==> $(GRN)We are done. Destroying the cluster now.$(END)\n"; kind delete cluster || true; \
	else \
		printf "$(CYN)==> $(GRN)We are done. You should destroy the cluster with 'kind delete cluster'.$(END)\n"; \
	fi

test: ingresstest gotest pytest
.PHONY: test

AMB_IMAGE_RC=$(RELEASE_REGISTRY)/$(REPO):$(patsubst v%,%,$(VERSION))
AMB_IMAGE_RELEASE=$(RELEASE_REGISTRY)/$(REPO):$(patsubst v%,%,$(VERSION))

export RELEASE_REGISTRY_ERR=$(RED)ERROR: please set the RELEASE_REGISTRY make/env variable to the docker registry\n       you would like to use for release$(END)

release/promote-oss/.main: $(tools/docker-promote)
	@[[ '$(PROMOTE_FROM_VERSION)' =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]] || (echo >&2 'Must set PROMOTE_FROM_VERSION to a vSEMVER value'; exit 1)
	@[[ '$(PROMOTE_TO_VERSION)'   =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]] || (echo >&2 'Must set PROMOTE_TO_VERSION to a vSEMVER value' ; exit 1)
	@[[ -n '$(PROMOTE_FROM_REPO)'                                     ]] || (echo >&2 'Must set PROMOTE_FROM_REPO' ; exit 1)
	@[[ -n '$(PROMOTE_TO_REPO)'                                       ]] || (echo >&2 'Must set PROMOTE_TO_REPO' ; exit 1)
	@case '$(PROMOTE_CHANNEL)' in \
	  ''|wip|early|test|hotfix) true;; \
	  *) echo >&2 'Unknown PROMOTE_CHANNEL $(PROMOTE_CHANNEL)'; exit 1;; \
	esac

	@printf "$(CYN)==> $(GRN)Promoting $(BLU)%s$(GRN) to $(BLU)%s$(GRN) (channel='$(BLU)%s$(GRN)')$(END)\n" '$(PROMOTE_FROM_VERSION)' '$(PROMOTE_TO_VERSION)' '$(PROMOTE_CHANNEL)'

	@printf '  pushing $(CYN)$(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_FROM_VERSION))$(END)...\n'
	$(tools/docker-promote) $(PROMOTE_FROM_REPO):$(patsubst v%,%,$(PROMOTE_FROM_VERSION)) $(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_TO_VERSION))
	docker push $(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_TO_VERSION))

ifneq ($(IS_PRIVATE),)
	@echo '$@: not pushing to S3 because this is a private repo'
else
	@printf '  pushing $(CYN)https://s3.amazonaws.com/$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt$(END)...\n'
	printf '%s' "$(patsubst v%,%,$(PROMOTE_TO_VERSION))" | aws s3 cp - s3://$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt

	@printf '  pushing $(CYN)s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json$(END)...\n'
	printf '{"application":"emissary","latest_version":"%s","notices":[]}' "$(patsubst v%,%,$(PROMOTE_TO_VERSION))" | aws s3 cp - s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json

	{ $(MAKE) \
	  push-manifests \
	  publish-docs-yaml; }
endif
.PHONY: release/promote-oss/.main

release/promote-oss/to-rc: $(tools/devversion)
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+|-dev)$$ ]] || (printf '$(RED)ERROR: VERSION=%s does not look like an RC or dev tag\n' "$(VERSION)"; exit 1)
	@set -e; { \
	  dev_version=$$($(tools/devversion)); \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN).$(END)\n"; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_TO_VERSION='$(VERSION)' \
	    PROMOTE_FROM_REPO='$(DEV_REGISTRY)/$(REPO)' \
	    PROMOTE_TO_REPO='$(RELEASE_REGISTRY)/$(REPO)' \
	    PROMOTE_CHANNEL='test'; \
	}
.PHONY: release/promote-oss/to-rc

# To be run from a checkout at the tag you are promoting _from_.
# This is normally run from CI by creating the GA tag.
release/promote-oss/to-ga: $(tools/devversion)
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: VERSION=%s does not look like a GA tag\n' "$(VERSION)"; exit 1)
	@set -e; { \
	  dev_version=$$($(tools/devversion)); \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN).$(END)\n"; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_TO_VERSION='$(VERSION)' \
	    PROMOTE_FROM_REPO='$(DEV_REGISTRY)/$(REPO)' \
	    PROMOTE_TO_REPO='$(RELEASE_REGISTRY)/$(REPO)' \
	    PROMOTE_CHANNEL=''; \
	}
.PHONY: release/promote-oss/to-ga

# `make release/ga-check` is meant to be run by a human maintainer to
# check that CI did all the right things.
release/ga-check:
	{ $(OSS_HOME)/releng/release-ga-check \
	  --ga-version=$(patsubst v%,%,$(VERSION)) \
	  --chart-version=$(patsubst v%,%,$(CHART_VERSION)) \
	  --source-registry=$(RELEASE_REGISTRY) \
	  --image-name=$(LCNAME); }

AMBASSADOR_DOCKER_IMAGE = $(shell sed -n 2p docker/$(LCNAME).docker.push.remote 2>/dev/null)
export AMBASSADOR_DOCKER_IMAGE

_user-vars  = BUILDER_NAME
_user-vars += DEV_KUBECONFIG
_user-vars += DEV_REGISTRY
_user-vars += RELEASE_REGISTRY
_user-vars += AMBASSADOR_DOCKER_IMAGE
env:
	@printf '$(BLD)%s$(END)=$(BLU)%s$(END)\n' $(foreach v,$(_user-vars), $v $(call quote.shell,$(call quote.shell,$($v))) )
.PHONY: env

export:
	@printf 'export %s=%s\n' $(foreach v,$(_user-vars), $v $(call quote.shell,$(call quote.shell,$($v))) )
.PHONY: export

help:
	@printf '%s\n' $(call quote.shell,$(_help.intro))
.PHONY: help

targets:
	@printf '%s\n' $(call quote.shell,$(HELP_TARGETS))
.PHONY: help

define HELP_TARGETS
$(BLD)Targets:$(END)

$(_help.targets)

$(BLD)Codebases:$(END)
  $(foreach MODULE,$(MODULES),$(NL)  $(BLD)$(SOURCE_$(MODULE)) ==> $(BLU)$(MODULE)$(END))

endef

# Style note: _help.intro
# - is wrapped to 72 columns (after stripping the ANSI color codes)
# - has sentences separated with 2 spaces
# - uses bold blue ("$(BLU)") when introducing a new variable
# - uses bold ("$(BLD)") for variables that have already been introduced
# - uses bold ("$(BLD)") when you would use `backticks` in markdown
define _help.intro
This Makefile builds Ambassador using a standard build environment
inside a Docker container.  The $(BLD)$(REPO)$(END), $(BLD)kat-server$(END), and $(BLD)kat-client$(END)
images are created from this container after the build stage is
finished.

The build works by maintaining a running build container in the
background.  It gets source code into that container via $(BLD)rsync$(END).  The
$(BLD)/home/dw$(END) directory in this container is a Docker volume, which allows
files (e.g. the Go build cache and $(BLD)pip$(END) downloads) to be cached across
builds.

This arrangement also permits building multiple codebases.  This is
useful for producing builds with extended functionality.  Each external
codebase is synced into the container at the $(BLD)/buildroot/<name>$(END) path.

You can control the name of the container and the images it builds by
setting $(BLU)$$BUILDER_NAME$(END), which defaults to $(BLD)$(LCNAME)$(END).  Note well that if
you want to make multiple clones of this repo and build in more than one
of them at the same time, you $(BLD)must$(END) set $(BLD)$$BUILDER_NAME$(END) so that each clone
has its own builder!  If you do not do this, your builds will collide
with confusing results.

The build system doesn't try to magically handle all dependencies.  In
general, if you change something that is not pure source code, you will
likely need to do a $(BLD)$(MAKE) clean$(END) in order to see the effect.  For example,
Python code only gets set up once, so if you change $(BLD)setup.py$(END), then you
will need to do a clean build to see the effects.  Assuming you didn't
$(BLD)$(MAKE) clobber$(END), this shouldn't take long due to the cache in the Docker
volume.

All targets that deploy to a cluster by way of $(BLU)$$DEV_REGISTRY$(END) can be made
to have the cluster use an imagePullSecret to pull from $(BLD)$$DEV_REGISTRY$(END),
by setting $(BLU)$$DEV_USE_IMAGEPULLSECRET$(END) to a non-empty value.  The
imagePullSecret will be constructed from $(BLD)$$DEV_REGISTRY$(END),
$(BLU)$$DOCKER_BUILD_USERNAME$(END), and $(BLU)$$DOCKER_BUILD_PASSWORD$(END).

Use $(BLD)$(MAKE) $(BLU)targets$(END) for help about available $(BLD)make$(END) targets.
endef

define _help.targets
  $(BLD)$(MAKE) $(BLU)help$(END)         -- displays the main help message.

  $(BLD)$(MAKE) $(BLU)targets$(END)      -- displays this message.

  $(BLD)$(MAKE) $(BLU)env$(END)          -- display the value of important env vars.

  $(BLD)$(MAKE) $(BLU)export$(END)       -- display important env vars in shell syntax, for use with $(BLD)eval$(END).

  $(BLD)$(MAKE) $(BLU)preflight$(END)    -- checks dependencies of this makefile.

  $(BLD)$(MAKE) $(BLU)version$(END)      -- display source code version.

  $(BLD)$(MAKE) $(BLU)images$(END)       -- creates images from the build container.

  $(BLD)$(MAKE) $(BLU)push$(END)         -- pushes images to $(BLD)$$DEV_REGISTRY$(END). ($(DEV_REGISTRY))

  $(BLD)$(MAKE) $(BLU)test$(END)         -- runs Go and Python tests inside the build container.

    The tests require a Kubernetes cluster and a Docker registry in order to
    function. These must be supplied via the $(BLD)$(MAKE)$(END)/$(BLD)env$(END) variables $(BLD)$$DEV_KUBECONFIG$(END)
    and $(BLD)$$DEV_REGISTRY$(END).

  $(BLD)$(MAKE) $(BLU)gotest$(END)       -- runs just the Go tests inside the build container.

    Use $(BLD)$$GOTEST_PKGS$(END) to control which packages are passed to $(BLD)gotest$(END). ($(GOTEST_PKGS))
    Use $(BLD)$$GOTEST_ARGS$(END) to supply additional non-package arguments. ($(GOTEST_ARGS))
    Example: $(BLD)$(MAKE) gotest GOTEST_PKGS=./cmd/entrypoint GOTEST_ARGS=-v$(END)  # run entrypoint tests verbosely

  $(BLD)$(MAKE) $(BLU)pytest$(END)       -- runs just the Python tests inside the build container.

    Use $(BLD)$$PYTEST_ARGS$(END) to pass args to $(BLD)pytest$(END). ($(PYTEST_ARGS))

    Example: $(BLD)$(MAKE) pytest PYTEST_ARGS="-k Lua"$(END)  # run only the Lua test, with a real Envoy

  $(BLD)$(MAKE) $(BLU)release/promote-oss/to-ga$(END) -- promote a release candidate to general availability

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z'. You must also have previously
    built and promoted the RC that will become GA, using $(BLD)release/bits$(END).

  $(BLD)$(MAKE) $(BLU)clean$(END)     -- kills the build container.

  $(BLD)$(MAKE) $(BLU)clobber$(END)   -- kills the build container and the cache volume.

  $(BLD)$(MAKE) $(BLU)generate$(END)  -- update generated files that get checked in to Git.

    1. Use $(BLD)$$ENVOY_COMMIT$(END) to update the vendored gRPC protobuf files ('api/envoy').
    2. Run 'protoc' to generate things from the protobuf files (both those from
       Envoy, and those from 'api/kat').
    3. Use $(BLD)$$ENVOY_GO_CONTROL_PLANE_COMMIT$(END) to update the vendored+patched copy of
       envoyproxy/go-control-plane ('pkg/envoy-control-plane/').
    4. Use the Go CRD definitions in 'pkg/api/getambassador.io/' to generate YAML
       (and a few 'zz_generated.*.go' files).

  $(BLD)$(MAKE) $(BLU)generate-fast$(END) -- like $(BLD)make generate$(END), but skips the slow Envoy stuff.

  $(BLD)$(MAKE) $(BLU)go-mod-tidy$(END) -- 'go mod tidy', but plays nice with 'make generate'

  $(BLD)$(MAKE) $(BLU)guess-envoy-go-control-plane-commit$(END) -- Make a suggestion for setting ENVOY_GO_CONTROL_PLANE_COMMIT= in generate.mk

  $(BLD)$(MAKE) $(BLU)lint$(END)        -- runs golangci-lint.

  $(BLD)$(MAKE) $(BLU)format$(END)      -- runs golangci-lint with --fix.

endef
