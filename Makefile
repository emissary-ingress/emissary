all: bump

VERSION=0.7.0

VERSIONED = \
	.bumpversion.cfg \
	BUILDING.md \
	Makefile \
	ambassador-rest.yaml \
	ambassador.yaml \
	ambassador/VERSION.py \
	templates/ambassador-rest.yaml.sh \

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
	$(MAKE) bump LEVEL=patch
	$(MAKE) artifacts

new-minor:
	$(MAKE) bump LEVEL=minor
	$(MAKE) artifacts

new-major:
	$(MAKE) bump LEVEL=major
	$(MAKE) artifacts

tag:
	git commit $(VERSIONED) -m "v$(VERSION)"
	git tag -a v$(VERSION) -m "v$(VERSION)"

ambassador-rest.yaml: .ALWAYS
	sh templates/ambassador-rest.yaml.sh > ambassador-rest.yaml

ambassador.yaml: ambassador-store.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-rest.yaml > ambassador.yaml

docker-images: ambassador-image statsd-image

ambassador-image: .ALWAYS
	scripts/docker_build_maybe_push ambassador $(VERSION) ambassador

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push statsd $(VERSION) statsd
