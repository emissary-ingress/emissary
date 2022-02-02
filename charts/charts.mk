EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress

push-preflight: $(OSS_HOME)/venv $(tools/yq)
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
chart-push-ci: charts/emissary-ingress/Chart.yaml
chart-push-ci: charts/emissary-ingress/values.yaml
chart-push-ci: charts/emissary-ingress/README.md
	[[ -z "$(IS_PRIVATE)" ]] || (echo "Private repo, not pushing chart" >&2; exit 1)
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/push_chart.sh
.PHONY: chart-push-ci

release/changelog:
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/update_chart_changelog.sh
.PHONY: release/changelog

release/chart/update-images: charts/emissary-ingress/Chart.yaml
release/chart/update-images: charts/emissary-ingress/values.yaml
release/chart/update-images: charts/emissary-ingress/README.md
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	IMAGE_TAG="${IMAGE_TAG}" CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/image_tag_changelog_update.sh
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/update_chart_changelog.sh
