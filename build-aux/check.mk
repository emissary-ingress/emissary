include build-aux/tools.mk

#
# Auxiliary Docker images needed for the tests

# Keep this list in-sync with python/tests/src/tests/integration/manifests.py
push-pytest-images: images
	k3d image load -c $(TEST_CLUSTER) ghcr.io/emissary-ingress/test-auth:latest-$(ARCH)
	k3d image load -c $(TEST_CLUSTER) ghcr.io/emissary-ingress/test-shadow:latest-$(ARCH)
	k3d image load -c $(TEST_CLUSTER) ghcr.io/emissary-ingress/test-stats:latest-$(ARCH)
	k3d image load -c $(TEST_CLUSTER) ghcr.io/emissary-ingress/kat-client:latest-$(ARCH)
	k3d image load -c $(TEST_CLUSTER) ghcr.io/emissary-ingress/kat-server:latest-$(ARCH)
.PHONY: push-pytest-images

#
# Helm tests

test-chart-values.yaml: docker/$(LCNAME).docker.push.remote build-aux/check.mk
	{ \
	  echo 'test:'; \
	  echo '  enabled: true'; \
	  echo 'image:'; \
	  sed -E -n '2s/^(.*):.*/  repository: \1/p' < $<; \
	  sed -E -n '2s/.*:/  tag: /p' < $<; \
	} >$@
clean: test-chart-values.yaml.rm
build-output/chart-%/ci: build-output/chart-% test-chart-values.yaml
	rm -rf $@
	cp -a $@.in $@
	for file in $@/*-values.yaml; do cat test-chart-values.yaml >> "$$file"; done

test-chart: $(tools/ct) $(tools/kubectl) $(chart_dir)/ci build-output/yaml-$(patsubst v%,%,$(VERSION)) $(if $(DEV_USE_IMAGEPULLSECRET),push-pytest-images $(OSS_HOME)/.venv)
ifneq ($(DEV_USE_IMAGEPULLSECRET),)
	KUBECONFIG=$(DEV_KUBECONFIG) uv run python3 -c 'from tests.integration.utils import install_crds; install_crds()'
else
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) apply -f build-output/yaml-$(patsubst v%,%,$(VERSION))/emissary-crds.yaml
endif
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) --namespace=emissary-system wait --timeout=90s --for=condition=available Deployments/emissary-apiext
	cd $(chart_dir) && KUBECONFIG=$(DEV_KUBECONFIG) $(abspath $(tools/ct)) install --config=./ct.yaml
.PHONY: test-chart

#
# Other

clean: .pytest_cache.rm-r .coverage.rm

dtest.clean:
	docker container list --filter=label=scope=dtest --quiet | xargs -r docker container kill
clean: dtest.clean
