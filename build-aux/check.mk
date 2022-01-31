include build-aux/tools.mk

# Keep this list in-sync with python/tests/integration/manifests.py
push-pytest-images: docker/emissary.docker.push.remote
push-pytest-images: docker/test-auth.docker.push.remote
push-pytest-images: docker/test-shadow.docker.push.remote
push-pytest-images: docker/test-stats.docker.push.remote
push-pytest-images: docker/kat-client.docker.push.remote
push-pytest-images: docker/kat-server.docker.push.remote
.PHONY: push-pytest-images

# These are triggered by `python/tests/integration/manifests.py` (or by `push-pytest-images`)
test_svcs = auth shadow stats
$(foreach svc,$(test_svcs),docker/.test-$(svc).docker.stamp): docker/.%.docker.stamp: docker/%/Dockerfile FORCE
	docker build --iidfile=$@ $(<D)
$(foreach svc,$(test_svcs),docker/test-$(svc).docker): docker/%.docker: docker/.%.docker.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
