EMISSARY_CHART = $(OSS_HOME)/charts/emissary-ingress

CHART_S3_BUCKET = $(or $(AWS_S3_BUCKET),datawire-static-files)
CHART_S3_PREFIX = $(if $(findstring -,$(CHART_VERSION)),charts-dev,charts)
release/push-chart: build-output/charts/emissary-ingress-$(patsubst v%,%,$(CHART_VERSION)).tgz
ifneq ($(IS_PRIVATE),)
	echo "Private repo, not pushing chart" >&2
else
	@if curl -k -L https://s3.amazonaws.com/$(CHART_S3_BUCKET)/$(CHART_S3_PREFIX)/index.yaml | grep -F $(<F); then \
	  printf 'Chart version %s is already in the index\n' '$(CHART_VERSION)' >&2; \
	  exit 1; \
	fi
	{ aws s3api put-object \
	  --bucket $(CHART_S3_BUCKET) \
	  --key $(CHART_S3_PREFIX)/$(<F) \
	  --body $<; }
endif
.PHONY: release/push-chart
