tools/ct      ?= ct
tools/k3d     ?= k3d
tools/kubectl ?= kubectl

HELM_TEST_VALUES ?=
CHART_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
CHART_KUBECONFIG := /tmp/kubeconfig/k3dconfig

K3S_VERSION      = 1.22.2+k3s1
K3D_CLUSTER_NAME = helm-chart-test-cluster
K3D_ARGS         = --k3s-server-arg=--no-deploy=traefik

$(CHART_DIR)/ci: %: %.in FORCE
	rm -rf $@
	cp -a $< $@
	$(if $(HELM_TEST_VALUES),for file in $@/*-values.yaml; do cat $(HELM_TEST_VALUES) >> "$$file"; done)

test-chart: lint-chart preflight-chart-test chart-create-cluster $(CHART_DIR)/ci
	KUBECONFIG=$(CHART_KUBECONFIG) $(tools/ct) install --config=$(CHART_DIR)/ct.yaml
	$(MAKE) chart-delete-cluster
.PHONY: test-chart

lint-chart: preflight-kubeconfig
	KUBECONFIG=$(CHART_KUBECONFIG) $(tools/ct) lint --config $(CHART_DIR)/ct.yaml
.PHONY: lint-chart

preflight-chart-test: preflight-kubeconfig
	# check if k3d is installed
	@if ! command -v $(tools/k3d) 2> /dev/null ; then \
	  echo 'k3d not installed, plz do that'; \
	  false; \
	fi
.PHONY: preflight-chart-test

preflight-kubeconfig:
	mkdir -p `dirname $(CHART_KUBECONFIG)`
	touch $(CHART_KUBECONFIG)
	chmod 600 $(CHART_KUBECONFIG)
.PHONY: preflight-kubeconfig

# TODO(lukeshu): De-duplicate chart-create-cluster with
# build-aux/ci.mk's `ci/setup-k3d` target.
chart-create-cluster: preflight-kubeconfig
	$(MAKE) chart-delete-cluster || true
	KUBECONFIG=$(CHART_KUBECONFIG) $(tools/k3d) cluster create $(K3D_CLUSTER_NAME) --image=docker.io/rancher/k3s:v$(subst +,-,$(K3S_VERSION)) $(K3D_ARGS)
	while ! $(tools/kubectl) --kubeconfig=$(CHART_KUBECONFIG) get serviceaccount default >/dev/null; do sleep 1; done
	$(tools/kubectl) --kubeconfig=$(CHART_KUBECONFIG) version
ifneq ($(DEV_USE_IMAGEPULLSECRET),)
	cd ../.. && . venv/bin/activate && KUBECONFIG=$(CHART_KUBECONFIG) python3 -c 'from tests.integration.utils import install_crds; install_crds()'
else
	$(tools/kubectl) --kubeconfig=$(CHART_KUBECONFIG) apply -f ../../manifests/emissary/emissary-crds.yaml
endif
	$(tools/kubectl) --kubeconfig=$(CHART_KUBECONFIG) --namespace=emissary-system wait --timeout=90s --for=condition=available Deployments/emissary-apiext
.PHONY: chart-create-cluster

chart-delete-cluster:
	KUBECONFIG=$(CHART_KUBECONFIG) $(tools/k3d) cluster delete $(K3D_CLUSTER_NAME)
.PHONY: chart-delete-cluster

.SECONDARY:
.PHONY: FORCE
