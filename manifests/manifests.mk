push-manifests:
	if [ ! -z $(IS_PRIVATE) ]; then \
		echo "Private repo, not pushing chart" ;\
		exit 1 ;\
	fi;
	$(OSS_HOME)/manifests/push_manifests.sh
.PHONY: push-manifests
