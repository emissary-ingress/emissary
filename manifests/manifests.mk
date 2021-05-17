push-manifests:
	$(OSS_HOME)/manifests/push_manifests.sh
.PHONY: push-manifests

# This should always be safe to run because the manifest yaml should all be generated
clean-manifests:
	@git restore $(OSS_HOME)/manifests/*/*.yaml
	git restore $(OSS_HOME)/docs/yaml
.PHONY: clean-manifests
