EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress

push-preflight: create-venv $(tools/yq)
	@$(OSS_HOME)/venv/bin/python -m pip install ruamel.yaml
.PHONY: push-preflight

release/ga/chart-push:
	$(OSS_HOME)/releng/release-wait-for-commit --commit $$(git rev-parse HEAD) --s3-key chart-builds
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/push_chart.sh
.PHONY: release/ga/chart-push

release/promote-chart-passed:
	@set -ex; { \
	  commit=$$(git rev-parse HEAD); \
	  printf "$(CYN)==> $(GRN)Promoting $(BLU)$$commit$(GRN) in S3...$(END)\n"; \
	  echo "PASSED" | aws s3 cp - s3://$(AWS_S3_BUCKET)/chart-builds/$$commit; \
	}
.PHONY: release/promote-chart-passed

chart-push-ci: push-preflight
	[[ -z "$(IS_PRIVATE)" ]] || (echo "Private repo, not pushing chart" >&2; exit 1)
	@[ -n "${CHART_VERSION_SUFFIX}" ] || (echo "CHART_VERSION_SUFFIX must be set for non-GA pushes" && exit 1)
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	@[ -n "${IMAGE_REPO}" ] || (echo "IMAGE_REPO must be set" && exit 1)
	sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+).*/version: \1${CHART_VERSION_SUFFIX}/g" $(EMISSARY_CHART)/Chart.yaml && rm $(EMISSARY_CHART)/Chart.yaml.bak
	{ $(OSS_HOME)/venv/bin/python $(OSS_HOME)/charts/scripts/update_chart_image_values.py \
	  --values-file=$(EMISSARY_CHART)/values.yaml \
	  --tag=$(IMAGE_TAG) \
	  --repo=$(IMAGE_REPO); }
	$(tools/yq) w -i $(EMISSARY_CHART)/Chart.yaml 'appVersion' ${IMAGE_TAG}
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/push_chart.sh
.PHONY: chart-push-ci

release/changelog:
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/update_chart_changelog.sh
.PHONY: release/changelog

release/chart/update-images: $(tools/yq) $(tools/chart-doc-gen)
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	([[ "${IMAGE_TAG}" =~ .*\.0$$ ]] && $(MAKE) release/chart-bump/minor) || $(MAKE) release/chart-bump/revision
	[[ "${IMAGE_TAG}" =~ .*\-ea$$ ]] && sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+).*/version: \1-ea/g" $(EMISSARY_CHART)/Chart.yaml && rm $(EMISSARY_CHART)/Chart.yaml.bak
	{ $(OSS_HOME)/venv/bin/python $(OSS_HOME)/charts/scripts/update_chart_image_values.py \
	  --values-file=$(EMISSARY_CHART)/values.yaml \
	  --tag=$(IMAGE_TAG); }
	$(tools/yq) w -i $(EMISSARY_CHART)/Chart.yaml 'appVersion' ${IMAGE_TAG}
	IMAGE_TAG="${IMAGE_TAG}" CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/image_tag_changelog_update.sh
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/update_chart_changelog.sh
	$(MAKE) $(EMISSARY_CHART)/README.md

# Both charts should have same versions for now. Just makes things a bit easier if we publish them together for now
release/chart-bump/revision:
	$(OSS_HOME)/charts/scripts/bump_chart_version.sh patch $(EMISSARY_CHART)/Chart.yaml
.PHONY: release/chart-bump/revision

release/chart-bump/minor:
	$(OSS_HOME)/charts/scripts/bump_chart_version.sh minor $(EMISSARY_CHART)/Chart.yaml
.PHONY: release/chart-bump/minor
