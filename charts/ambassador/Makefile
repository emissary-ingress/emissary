HELM_TEST_IMAGE = quay.io/helmpack/chart-testing:v3.0.0-rc.1
K3D_CLUSTER_NAME = helm-chart-test-cluster
CHART_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
KUBECONFIG := /tmp/kubeconfig/k3dconfig
CT_EXEC = docker run --rm -v $(KUBECONFIG):/root/.kube/config -v $(CHART_DIR):/charts --network host $(HELM_TEST_IMAGE) ct
K3D_EXEC := KUBECONFIG=$(KUBECONFIG) k3d

test-chart: lint-chart preflight-chart-test chart-create-cluster
	$(CT_EXEC) install --config /charts/ct.yaml && \
		$(MAKE) chart-delete-cluster
.PHONY: test-chart

lint-chart: preflight-kubeconfig
	$(CT_EXEC) lint --config /charts/ct.yaml
.PHONY: lint-chart

preflight-chart-test: preflight-kubeconfig
	# check if k3d is installed
	@if ! command -v k3d 2> /dev/null ; then \
		printf 'k3d not installed, plz do that'; \
	    false; \
	fi
.PHONY: preflight-chart-test

preflight-kubeconfig:
	mkdir -p `dirname $(KUBECONFIG)`
	touch $(KUBECONFIG)
.PHONY: preflight-kubeconfig

chart-create-cluster: preflight-kubeconfig
	$(MAKE) chart-delete-cluster || true
	$(K3D_EXEC) cluster create $(K3D_CLUSTER_NAME) --k3s-server-arg "--no-deploy=traefik"
.PHONY: chart-create-cluster

chart-delete-cluster:
	$(K3D_EXEC) cluster delete $(K3D_CLUSTER_NAME)
.PHONY: chart-delete-cluster
