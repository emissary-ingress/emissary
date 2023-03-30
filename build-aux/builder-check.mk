# TODO(lukeshu): Migrate from this to the other `build-aux/check.mk`.

include build-aux/colors.mk

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
INGRESS_TEST_MANIF_DIR = manifests/emissary/
INGRESS_TEST_MANIFS = emissary-crds.yaml emissary-emissaryns.yaml

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

export KUBECONFIG_ERR=$(RED)ERROR: please set the $(BLU)DEV_KUBECONFIG$(RED) make/env variable to the cluster\n       you would like to use for development. Note this cluster must have access\n       to $(BLU)DEV_REGISTRY$(RED) (currently $(BLD)$(DEV_REGISTRY)$(END)$(RED))$(END)
export KUBECTL_ERR=$(RED)ERROR: preflight kubectl check failed$(END)

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
	@echo "KAT_RUN_MODE=$$KAT_RUN_MODE"
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

pytest-kat-envoy3: push-pytest-images # doing this all at once is too much for CI...
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS python/tests/kat"
# ... so we have a separate rule to run things split up
build-aux/.pytest-kat.txt.stamp: $(OSS_HOME)/venv push-pytest-images $(tools/kubectl) FORCE
	. venv/bin/activate && set -o pipefail && pytest --collect-only python/tests/kat 2>&1 | sed -En 's/.*<Function (.*)>/\1/p' | cut -d. -f1 | sort -u > $@
build-aux/pytest-kat.txt: build-aux/%: build-aux/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
clean: build-aux/.pytest-kat.txt.stamp.rm build-aux/pytest-kat.txt.rm
pytest-kat-envoy3-%: build-aux/pytest-kat.txt $(tools/py-split-tests)
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS -k '$$($(tools/py-split-tests) $(subst -of-, ,$*) <build-aux/pytest-kat.txt)' python/tests/kat"

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

AMBASSADOR_DOCKER_IMAGE = $(shell sed -n 2p docker/$(LCNAME).docker.push.remote 2>/dev/null)
export AMBASSADOR_DOCKER_IMAGE
