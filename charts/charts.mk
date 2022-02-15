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

release/promote-chart-passed:
	@set -ex; { \
	  commit=$$(git rev-parse HEAD); \
	  printf "$(CYN)==> $(GRN)Promoting $(BLU)$$commit$(GRN) in S3...$(END)\n"; \
	  echo "PASSED" | aws s3 cp - s3://$(AWS_S3_BUCKET)/chart-builds/$$commit; \
	}
.PHONY: release/promote-chart-passed
