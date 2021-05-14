AMBASSADOR_CHART = $(OSS_HOME)/charts/ambassador
EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress
YQ := $(OSS_HOME)/.circleci/yq

define _push_chart
	$(1)/ci/push_chart.sh
endef

define _set_tag
	$(YQ) write -i $(1)/values.yaml 'image.tag' $(2)
endef

define _set_repo
	$(YQ) write -i $(1)/values.yaml 'image.repository' $(2)
endef

chart-push-ci:
	@echo ">>> This will dirty your local tree and should only be run in CI"
	@echo ">>> If running locally, you'll probably want to reset charts/ directory after running this"
	@[ -n "${CHART_VERSION_SUFFIX}" ] || (echo "CHART_VERSION_SUFFIX must be set for non-GA pushes" && exit 1)
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	@[ -n "${IMAGE_REPO}" ] || (echo "IMAGE_REPO must be set" && exit 1)
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+).*/version: \1${CHART_VERSION_SUFFIX}/g" $$chart/Chart.yaml && rm $$chart/Chart.yaml.bak ; \
		$(call _set_tag,$$chart,${IMAGE_TAG}) ; \
		$(call _set_repo,$$chart,${IMAGE_REPO}) ; \
		$(call _push_chart,$$chart) ; \
	done ;

# This is pretty Draconian. Use with care.
chart-clean:
	git restore charts/*/Chart.yaml charts/*/values.yaml
	rm -f charts/*/*.tgz charts/*/index.yaml charts/*/tmp.yaml
.PHONY: chart-clean
