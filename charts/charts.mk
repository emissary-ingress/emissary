EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress

release/push-chart: charts/emissary-ingress/Chart.yaml
release/push-chart: charts/emissary-ingress/values.yaml
release/push-chart: charts/emissary-ingress/README.md
ifneq ($(IS_PRIVATE),)
	echo "Private repo, not pushing chart" >&2
else
	CHART_NAME=$(notdir $(EMISSARY_CHART)) $(OSS_HOME)/charts/scripts/push_chart.sh
endif
.PHONY: release/push-chart
