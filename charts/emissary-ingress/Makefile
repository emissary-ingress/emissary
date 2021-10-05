HELM_TEST_IMAGE = quay.io/helmpack/chart-testing:v3.0.0-rc.1
HELM_TEST_VALUES ?=
CHART_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
CHART_KUBECONFIG := /tmp/kubeconfig/k3dconfig
K3D_EXEC := KUBECONFIG=$(CHART_KUBECONFIG) k3d

K3S_VERSION      = 1.21.5+k3s1
K3D_CLUSTER_NAME = helm-chart-test-cluster
K3D_ARGS         = --k3s-server-arg=--no-deploy=traefik

test-chart: lint-chart preflight-chart-test chart-create-cluster
	rm -rf $(CHART_DIR)/ci.tmp
	cp -a $(CHART_DIR)/ci $(CHART_DIR)/ci.tmp
	$(if $(HELM_TEST_VALUES),for file in $(CHART_DIR)/ci.tmp/*-values.yaml; do cat $(HELM_TEST_VALUES) >> "$$file"; done)
	{ docker run --rm \
	  --network=host \
	  --volume=$(CHART_KUBECONFIG):/root/.kube/config \
	  --volume=$(CHART_DIR):/charts \
	  --volume=$(CHART_DIR)/ci.tmp:/charts/ci \
	  $(HELM_TEST_IMAGE) \
	  ct install --config=/charts/ct.yaml; }
	$(MAKE) chart-delete-cluster
.PHONY: test-chart

lint-chart: preflight-kubeconfig
# This does NOT have the extra ci.tmp volume that the chart-test rule has.
	{ docker run --rm \
	  --network=host \
	  --volume=$(CHART_KUBECONFIG):/root/.kube/config \
	  --volume=$(CHART_DIR):/charts \
	  $(HELM_TEST_IMAGE) \
	  ct lint --config /charts/ct.yaml; }
.PHONY: lint-chart

preflight-chart-test: preflight-kubeconfig
	# check if k3d is installed
	@if ! command -v k3d 2> /dev/null ; then \
	  echo 'k3d not installed, plz do that'; \
	  false; \
	fi
.PHONY: preflight-chart-test

preflight-kubeconfig:
	mkdir -p `dirname $(CHART_KUBECONFIG)`
	touch $(CHART_KUBECONFIG)
.PHONY: preflight-kubeconfig

chart-create-cluster: preflight-kubeconfig
	$(MAKE) chart-delete-cluster || true
	$(K3D_EXEC) cluster create $(K3D_CLUSTER_NAME) --image=docker.io/rancher/k3s:v$(subst +,-,$(K3S_VERSION)) $(K3D_ARGS)
.PHONY: chart-create-cluster

chart-delete-cluster:
	$(K3D_EXEC) cluster delete $(K3D_CLUSTER_NAME)
.PHONY: chart-delete-cluster
