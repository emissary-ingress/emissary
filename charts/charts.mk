# No change to these, we'll just publish it with the normie stuffs
AMBASSADOR_CHART := $(OSS_HOME)/charts/ambassador
EMISSARY_CHART := $(OSS_HOME)/charts/emissary-ingress
YQ := $(OSS_HOME)/.circleci/yq

chart-push-ci: chart-set-tag chart-set-repo
	@[ -n "${CHART_VERSION_SUFFIX}" ] || (echo "CHART_VERSION_SUFFIX must be set for non-GA pushes" && exit 1)
	@sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+)/version: \1${CHART_VERSION_SUFFIX}/g" $(AMBASSADOR_CHART)/Chart.yaml && rm $(AMBASSADOR_CHART)/Chart.yaml.bak
	@sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+)/version: \1${CHART_VERSION_SUFFIX}/g" $(EMISSARY_CHART)/Chart.yaml && rm $(EMISSARY_CHART)/Chart.yaml.bak
	@$(MAKE) _push-charts

_push-charts:
	@echo ">>> Pushing charts..."
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$$chart/ci/push_chart.sh ; \
	done ;

chart-set-tag:
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	@echo ">>> Setting tag to ${IMAGE_TAG}"
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$(YQ) write -i $$chart/values.yaml 'image.tag' ${IMAGE_TAG} ;\
	done ;

chart-set-repo:
	@[ -n "${IMAGE_REPO}" ] || (echo "IMAGE_REPO must be set" && exit 1)
	@echo ">> Setting repo to ${IMAGE_REPO}"
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$(YQ) write -i $$chart/values.yaml 'image.tag' ${IMAGE_TAG} ;\
	done ;
