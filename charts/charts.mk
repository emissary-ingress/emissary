AMBASSADOR_CHART = $(OSS_HOME)/charts/ambassador
EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress
YQ := $(OSS_HOME)/.circleci/yq

define _push_chart
	CHART_NAME=$(1) $(OSS_HOME)/charts/scripts/push_chart.sh
endef

define _set_tag_and_repo
	$(OSS_HOME)/venv/bin/python $(OSS_HOME)/charts/scripts/update_chart_image_values.py $(1) $(2) $(3)
endef

define _set_tag
	$(OSS_HOME)/venv/bin/python $(OSS_HOME)/charts/scripts/update_chart_image_values.py $(1) $(2)
endef

define _docgen
	if [[ -f $(1)/doc.yaml ]] ; then \
		chart-doc-gen -d $(1)/doc.yaml -t $(1)/readme.tpl -v $(1)/values.yaml > $(1)/README.md ; \
	fi
endef

push-preflight: create-venv
	@$(OSS_HOME)/venv/bin/python -m pip install ruamel.yaml
.PHONY: push-preflight

chart-push-ci: push-preflight
	@echo ">>> This will dirty your local tree and should only be run in CI"
	@echo ">>> If running locally, you'll probably want to run make chart-clean after running this"
	@[ -n "${CHART_VERSION_SUFFIX}" ] || (echo "CHART_VERSION_SUFFIX must be set for non-GA pushes" && exit 1)
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	@[ -n "${IMAGE_REPO}" ] || (echo "IMAGE_REPO must be set" && exit 1)
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+).*/version: \1${CHART_VERSION_SUFFIX}/g" $$chart/Chart.yaml && rm $$chart/Chart.yaml.bak ; \
		$(call _set_tag_and_repo,$$chart/values.yaml,${IMAGE_TAG},${IMAGE_REPO}) ; \
		$(call _push_chart,`basename $$chart`) ; \
	done ;
.PHONY: chart-push-ci

chart-push-ga: push-preflight
	@echo ">>> This will dirty your local tree and should only be run in CI"
	@echo ">>> If running locally, you'll probably want to run make chart-clean after running this"
	@[ -z "${CHART_VERSION_SUFFIX}" ] || (echo "CHART_VERSION_SUFFIX must not be set for GA pushes" && exit 1)
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	@[ -n "${IMAGE_REPO}" ] || (echo "IMAGE_REPO must be set" && exit 1)
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		sed -i.bak -E "s/version: ([0-9]+\.[0-9]+\.[0-9]+).*/version: \1/g" $$chart/Chart.yaml && rm $$chart/Chart.yaml.bak ; \
		$(call _set_tag_and_repo,$$chart/values.yaml,${IMAGE_TAG},${IMAGE_REPO}) ; \
		$(call _push_chart,`basename $$chart`) ; \
	done ;
.PHONY: chart-push-ga

release/chart/tag:
	@set -e; { \
		if [ -n "$(IS_DIRTY)" ]; then \
			echo "release/chart/tag: tree must be clean" >&2 ;\
			exit 1 ;\
		fi; \
		chart_ver=`grep 'version:' $(AMBASSADOR_CHART)/Chart.yaml | awk ' { print $$2 }'` ; \
		chart_ver=chart-v$${chart_ver} ; \
		git tag -s -m "Tagging $${chart_ver}" -a $${chart_ver} ; \
		git push origin $${chart_ver} ; \
	}


release/changelog:
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		CHART_NAME=`basename $$chart` $(OSS_HOME)/charts/scripts/update_chart_changelog.sh ; \
	done ;
.PHONY: release/changelog

release/chart/update-images: doc-gen-preflight
	@[ -n "${IMAGE_TAG}" ] || (echo "IMAGE_TAG must be set" && exit 1)
	([[ "${IMAGE_TAG}" =~ .*\.0$$ ]] && $(MAKE) release/chart-bump/minor) || $(MAKE) release/chart-bump/revision
	for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$(call _set_tag,$$chart/values.yaml,${IMAGE_TAG}) ; \
		$(YQ) w -i $$chart/Chart.yaml 'appVersion' ${IMAGE_TAG} ; \
		IMAGE_TAG="${IMAGE_TAG}" CHART_NAME=`basename $$chart` $(OSS_HOME)/charts/scripts/image_tag_changelog_update.sh ; \
		CHART_NAME=`basename $$chart` $(OSS_HOME)/charts/scripts/update_chart_changelog.sh ; \
		$(call _docgen,$$chart) ; \
	done ;

# Both charts should have same versions for now. Just makes things a bit easier if we publish them together for now
release/chart-bump/revision:
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$(OSS_HOME)/charts/scripts/bump_chart_version.sh patch $$chart/Chart.yaml ; \
	done ;
.PHONY: release/chart-bump/revision

release/chart-bump/minor:
	@for chart in $(AMBASSADOR_CHART) $(EMISSARY_CHART) ; do \
		$(OSS_HOME)/charts/scripts/bump_chart_version.sh minor $$chart/Chart.yaml ; \
	done ;
.PHONY: release/chart-bump/minor

# This is pretty Draconian. Use with care.
chart-clean:
	git restore $(OSS_HOME)/charts/*/Chart.yaml $(OSS_HOME)/charts/*/values.yaml
	rm -f $(OSS_HOME)/charts/*/*.tgz $(OSS_HOME)/charts/*/index.yaml $(OSS_HOME)/charts/*/tmp.yaml
.PHONY: chart-clean

doc-gen-preflight:
	@if ! command -v chart-doc-gen 2> /dev/null ; then \
		printf 'chart-doc-gen not installed, see https://github.com/kubepack/chart-doc-gen'; \
	    false; \
	fi
.PHONY: doc-gen-preflight
