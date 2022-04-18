push-manifests:
ifneq ($(IS_PRIVATE),)
	@echo "Private repo, not pushing chart" >&2
	@exit 1
else
	manifests/push_manifests.sh manifests/emissary
endif
.PHONY: push-manifests
