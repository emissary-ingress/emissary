include build-aux/tools.mk

# These are triggered by `python/tests/integration/manifests.py`.
test_svcs = auth ratelimit shadow stats
$(foreach svc,$(test_svcs),docker/.test-$(svc).docker.stamp): docker/.%.docker.stamp: docker/%/Dockerfile FORCE
	docker build --iidfile=$@ $(<D)
$(foreach svc,$(test_svcs),docker/test-$(svc).docker): docker/%.docker: docker/.%.docker.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
