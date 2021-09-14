include $(OSS_HOME)/build-aux/teleproxy.mk

COPY_GOLD = $(abspath $(BUILDER_HOME)/copy-gold.sh)

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
INGRESS_TEST_MANIFS = ambassador-crds.yaml ambassador.yaml

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

test-ready: push preflight-cluster
.PHONY: test-ready

PYTEST_ARGS ?=
export PYTEST_ARGS

PYTEST_GOLD_DIR ?= $(abspath python/tests/gold)

pytest: docker/$(LCNAME).docker.push.remote-devloop
pytest: docker/kat-client.docker.push.remote-devloop
pytest: docker/kat-server.docker.push.remote-devloop
pytest: $(tools/kubestatus)
	@$(MAKE) setup-diagd
	@$(MAKE) setup-envoy
	@$(MAKE) proxy
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) tests$(END)\n"
	@echo "AMBASSADOR_DOCKER_IMAGE=$$AMBASSADOR_DOCKER_IMAGE"
	@echo "KAT_CLIENT_DOCKER_IMAGE=$$KAT_CLIENT_DOCKER_IMAGE"
	@echo "KAT_SERVER_DOCKER_IMAGE=$$KAT_SERVER_DOCKER_IMAGE"
	@echo "DEV_KUBECONFIG=$$DEV_KUBECONFIG"
	@echo "KAT_RUN_MODE=$$KAT_RUN_MODE"
	@echo "PYTEST_ARGS=$$PYTEST_ARGS"
	. $(OSS_HOME)/venv/bin/activate; \
		$(OSS_HOME)/builder/builder.sh pytest-local
.PHONY: pytest

pytest-unit:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) unit tests$(END)\n"
	@$(MAKE) setup-envoy
	@$(MAKE) setup-diagd
	. $(OSS_HOME)/venv/bin/activate; \
		PYTEST_ARGS="$$PYTEST_ARGS python/tests/unit" $(OSS_HOME)/builder/builder.sh pytest-local-unit
.PHONY: pytest-unit

pytest-integration:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) integration tests$(END)\n"
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS python/tests/integration"
.PHONY: pytest-integration

pytest-kat:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) kat tests$(END)\n"
	$(MAKE) pytest PYTEST_ARGS="$$PYTEST_ARGS python/tests/kat"
.PHONY: pytest-kat

pytest-builder: test-ready
	$(MAKE) pytest-builder-only
.PHONY: pytest-builder

pytest-envoy-ah:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy$(GRN) kat tests (ah) $(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy PYTEST_ARGS="$$PYTEST_ARGS --letter-range ah python/tests/kat"
.PHONY: pytest-envoy-ah

pytest-envoy-ip:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy$(GRN) kat tests (ip)$(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy PYTEST_ARGS="$$PYTEST_ARGS --letter-range ip python/tests/kat"
.PHONY: pytest-envoy-ip

pytest-envoy-qz:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy$(GRN) kat tests (qz)$(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy PYTEST_ARGS="$$PYTEST_ARGS --letter-range qz python/tests/kat"
.PHONY: pytest-envoy-qz


pytest-envoy:
	$(MAKE) pytest KAT_RUN_MODE=envoy PYTEST_ARGS="$$PYTEST_ARGS python/tests/kat"
.PHONY: pytest-envoy

pytest-envoy-builder:
	$(MAKE) pytest-builder KAT_RUN_MODE=envoy
.PHONY: pytest-envoy-builder

pytest-envoy-v2-ah:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy v2$(GRN) kat tests (ah)$(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy AMBASSADOR_ENVOY_API_VERSION=V2 PYTEST_ARGS="$$PYTEST_ARGS --letter-range ah python/tests/kat"
.PHONY: pytest-envoy-ah

pytest-envoy-v2-ip:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy v2$(GRN) kat tests (ip)$(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy AMBASSADOR_ENVOY_API_VERSION=V2 PYTEST_ARGS="$$PYTEST_ARGS --letter-range ip python/tests/kat"
.PHONY: pytest-envoy-v2-ip

pytest-envoy-v2-qz:
	@printf "$(CYN)==> $(GRN)Running $(BLU)py envoy v2$(GRN) kat tests (qz)$(END)\n"
	$(MAKE) pytest KAT_RUN_MODE=envoy AMBASSADOR_ENVOY_API_VERSION=V2 PYTEST_ARGS="$$PYTEST_ARGS --letter-range qz python/tests/kat"
.PHONY: pytest-envoy-v2-qz

pytest-envoy-v2:
	$(MAKE) pytest KAT_RUN_MODE=envoy AMBASSADOR_ENVOY_API_VERSION=V2 PYTEST_ARGS="$$PYTEST_ARGS python/tests/kat"
.PHONY: pytest-envoy-v2

pytest-envoy-v2-builder:
	$(MAKE) pytest-builder KAT_RUN_MODE=envoy AMBASSADOR_ENVOY_API_VERSION=V2
.PHONY: pytest-envoy-v2-builder

pytest-builder-only: sync preflight-cluster
pytest-builder-only: | docker/$(LCNAME).docker.push.remote-devloop
pytest-builder-only: | docker/kat-client.docker.push.remote-devloop
pytest-builder-only: | docker/kat-server.docker.push.remote-devloop
	@printf "$(CYN)==> $(GRN)Running $(BLU)py$(GRN) tests in builder shell$(END)\n"
	docker exec \
		-e AMBASSADOR_DOCKER_IMAGE=$$(sed -n 2p docker/$(LCNAME).docker.push.remote-devloop) \
		-e KAT_CLIENT_DOCKER_IMAGE=$$(sed -n 2p docker/kat-client.docker.push.remote-devloop) \
		-e KAT_SERVER_DOCKER_IMAGE=$$(sed -n 2p docker/kat-server.docker.push.remote-devloop) \
		-e KAT_IMAGE_PULL_POLICY=Always \
		-e DOCKER_NETWORK=$(DOCKER_NETWORK) \
		-e KAT_REQ_LIMIT \
		-e KAT_RUN_MODE \
		-e KAT_VERBOSE \
		-e PYTEST_ARGS \
		-e TEST_SERVICE_REGISTRY \
		-e TEST_SERVICE_VERSION \
		-e DEV_USE_IMAGEPULLSECRET \
		-e DEV_REGISTRY \
		-e DOCKER_BUILD_USERNAME \
		-e DOCKER_BUILD_PASSWORD \
		-e AMBASSADOR_ENVOY_API_VERSION \
		-e AMBASSADOR_LEGACY_MODE \
		-e AMBASSADOR_FAST_RECONFIGURE \
		-e AWS_SECRET_ACCESS_KEY \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SESSION_TOKEN \
		-it $(shell $(BUILDER)) /buildroot/builder.sh pytest-internal ; test_exit=$$? ; \
		[ -n "$(TEST_XML_DIR)" ] && docker cp $(shell $(BUILDER)):/tmp/test-data/pytest.xml $(TEST_XML_DIR) ; exit $$test_exit
.PHONY: pytest-builder-only

pytest-gold:
	sh $(COPY_GOLD) $(PYTEST_GOLD_DIR)

GOTEST_PKGS = github.com/datawire/ambassador/v2/...
GOTEST_MODDIRS = $(OSS_HOME)
export GOTEST_PKGS
export GOTEST_MODDIRS

GOTEST_ARGS ?= -race -count=1
export GOTEST_ARGS

gotest: setup-diagd
	@printf "$(CYN)==> $(GRN)Running $(BLU)go$(GRN) tests$(END)\n"
	. $(OSS_HOME)/venv/bin/activate; \
		EDGE_STACK=$(GOTEST_AES_ENABLED) \
		$(OSS_HOME)/builder/builder.sh gotest-local
.PHONY: gotest

# Ingress v1 conformance tests, using KIND and the Ingress Conformance Tests suite.
ingresstest: | docker/$(LCNAME).docker.push.remote-devloop
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
	@kubectl --kubeconfig=$(KIND_KUBECONFIG) cluster-info || { printf "$(RED)ERROR: kubernetes cluster not ready $(END)\n"; exit 1 ; }
	@kubectl --kubeconfig=$(KIND_KUBECONFIG) version || { printf "$(RED)ERROR: kubernetes cluster not ready $(END)\n"; exit 1 ; }

	@printf "$(CYN)==> $(GRN)Loading Ambassador (from the Ingress conformance tests) with image=$$(sed -n 2p docker/$(LCNAME).docker.push.remote-devloop)$(END)\n"
	@for f in $(INGRESS_TEST_MANIFS) ; do \
		printf "$(CYN)==> $(GRN)... $$f $(END)\n" ; \
		cat $(INGRESS_TEST_MANIF_DIR)/$$f | sed -e "s|image:.*ambassador\:.*|image: $$(sed -n 2p docker/$(LCNAME).docker.push.remote-devloop)|g" | tee /dev/tty | kubectl apply -f - ; \
	done

	@printf "$(CYN)==> $(GRN)Waiting for Ambassador to be ready$(END)\n"
	@kubectl --kubeconfig=$(KIND_KUBECONFIG) wait --for=condition=available --timeout=180s deployment/ambassador || { \
		printf "$(RED)ERROR: Ambassador was not ready after 3 mins $(END)\n"; \
		kubectl --kubeconfig=$(KIND_KUBECONFIG) get services --all-namespaces ; \
		exit 1 ; }

	@printf "$(CYN)==> $(GRN)Exposing Ambassador service$(END)\n"
	@kubectl --kubeconfig=$(KIND_KUBECONFIG) expose deployment ambassador --type=LoadBalancer --name=ambassador

	@printf "$(CYN)==> $(GRN)Starting the tests container (in the background)$(END)\n"
	@docker stop -t 3 ingress-tests 2>/dev/null || true && docker rm ingress-tests 2>/dev/null || true
	@docker run -d --rm --name ingress-tests -e KUBECONFIG=/opt/.kube/config --mount type=bind,source=$(KIND_KUBECONFIG),target=/opt/.kube/config \
		--entrypoint "/bin/sleep" $(INGRESS_TEST_IMAGE) 600

	@printf "$(CYN)==> $(GRN)Loading the Ingress conformance tests manifests$(END)\n"
	@docker exec -ti ingress-tests \
		/opt/ingress-controller-conformance apply --api-version=networking.k8s.io/v1beta1 --ingress-controller=getambassador.io/ingress-controller --ingress-class=ambassador
	@sleep 10

	@printf "$(CYN)==> $(GRN)Forwarding traffic to Ambassador service$(END)\n"
	@kubectl --kubeconfig=$(KIND_KUBECONFIG) port-forward --address=$(HOST_IP) svc/ambassador \
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
	-@pkill kubectl -9
	@docker stop -t 3 ingress-tests 2>/dev/null || true && docker rm ingress-tests 2>/dev/null || true

	@if [ -n "$(CLEANUP)" ] ; then \
		printf "$(CYN)==> $(GRN)We are done. Destroying the cluster now.$(END)\n"; kind delete cluster || true; \
	else \
		printf "$(CYN)==> $(GRN)We are done. You should destroy the cluster with 'kind delete cluster'.$(END)\n"; \
	fi

test: ingresstest gotest pytest
.PHONY: test
