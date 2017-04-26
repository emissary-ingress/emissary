all: bump

VERSION=0.5.1

VERSIONED = \
	.bumpversion.cfg \
	BUILDING.md \
	Makefile \
	ambassador-rest.yaml \
	ambassador-sds.yaml \
	ambassador.yaml \
	ambassador/VERSION.py \
	statsd-sink.yaml \
	templates/ambassador-rest.yaml.sh \
	templates/ambassador-sds.yaml.sh \
	templates/statsd-sink.yaml.sh

.ALWAYS:

artifacts: docker-images ambassador.yaml statsd-sink.yaml

bump:
	@if [ -z "$$LEVEL" ]; then \
	    echo "LEVEL must be set" >&2 ;\
	    exit 1 ;\
	fi

	@if [ -z "$$DOCKER_REGISTRY" ]; then \
	    echo "DOCKER_REGISTRY must be set" >&2 ;\
	    exit 1 ;\
	fi

	@echo "Bumping to new $$LEVEL version..."
	bump2version --no-tag --no-commit "$$LEVEL"

new-patch:
	$(MAKE) bump artifacts LEVEL=patch

new-minor:
	$(MAKE) bump artifacts LEVEL=minor

new-major:
	$(MAKE) bump artifacts LEVEL=major

tag:
	git commit $(VERSIONED)
	git tag -a v$(VERSION)

ambassador-sds.yaml: .ALWAYS
	sh templates/ambassador-sds.yaml.sh > ambassador-sds.yaml

ambassador-rest.yaml: .ALWAYS
	sh templates/ambassador-rest.yaml.sh > ambassador-rest.yaml

ambassador.yaml: ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-sds.yaml ambassador-rest.yaml > ambassador.yaml

statsd-sink.yaml: .ALWAYS
	sh templates/statsd-sink.yaml.sh > statsd-sink.yaml

docker-images: ambassador-image sds-image statsd-image prom-statsd-exporter

ambassador-image: .ALWAYS
	scripts/docker_build_maybe_push dwflynn ambassador $(VERSION) ambassador

sds-image: .ALWAYS
	scripts/docker_build_maybe_push dwflynn ambassador-sds $(VERSION) sds

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push ark3 statsd $(VERSION) statsd

prom-statsd-exporter: .ALWAYS
	scripts/docker_build_maybe_push ark3 prom-statsd-exporter $(VERSION) prom-statsd-exporter
