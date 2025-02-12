IMAGE_REPO ?= docker.io/emissaryingress/emissary

all: charts

charts: emissary-crds-chart emissary-ingress

# These are just aliases
emissary-crds-chart: emissary-crds-chart-$(VERSION).tgz
emissary-ingress: emissary-ingress-$(VERSION).tgz

version-check:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION must be set (e.g. VERSION=1.0.0-alpha.3)" >&2 ;\
		exit 1; \
	fi
.PHONY: version-check

helm-registry-check:
	@if [ -z "$(HELM_REGISTRY)" ]; then \
		echo "HELM_REGISTRY must be set (e.g. HELM_REGISTRY=oci://docker.io/dwflynn)" >&2 ;\
		exit 1; \
	fi
.PHONY: helm-registry-check

emissary-crds-chart-$(VERSION).tgz: version-check emissary-crds
	bash bin/build-chart emissary-crds $(VERSION) $(IMAGE_REPO) $$(pwd)
	ls -l emissary-crds-chart-$(VERSION).tgz

emissary-ingress-$(VERSION).tgz: version-check emissary-chart
	bash bin/build-chart emissary-chart $(VERSION) $(IMAGE_REPO) $$(pwd)
	ls -l emissary-ingress-$(VERSION).tgz

push-chart: version-check helm-registry-check charts
	if [ -n "$(HELM_REGISTRY)" ]; then \
		helm push emissary-crds-chart-$(VERSION).tgz $(HELM_REGISTRY); \
		helm push emissary-ingress-$(VERSION).tgz $(HELM_REGISTRY); \
	else \
		echo "HELM_REGISTRY not set, not pushing"; \
	fi

clean:
	rm -rf emissary-crds-chart-*
	rm -rf emissary-ingress-*

