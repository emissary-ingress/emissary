all: bump

VERSION=0.9.1

# Make sure to update this list and .bumpversion.cfg at the same time.

VERSIONED = \
	.bumpversion.cfg \
	BUILDING.md \
	Makefile \
	ambassador-rest.yaml \
	ambassador.yaml \
	istio/ambassador.yaml \
	ambassador/VERSION.py \
	actl/VERSION.py \
	templates/ambassador-rest.yaml.sh \
	templates/ambassador-istio.yaml.sh \
	docs/user-guide/getting-started.md \
	docs/user-guide/with-istio.md

.ALWAYS:

artifacts: docker-images ambassador.yaml istio/ambassador.yaml

reg-check:
	@if [ -z "$$DOCKER_REGISTRY" ]; then \
	    echo "DOCKER_REGISTRY must be set" >&2 ;\
	    exit 1 ;\
	fi

bump: reg-check
	@if [ -z "$$LEVEL" ]; then \
	    echo "LEVEL must be set" >&2 ;\
	    exit 1 ;\
	fi

	@echo "Bumping to new $$LEVEL version..."
	bump2version --no-tag --no-commit "$$LEVEL"
	@echo "Building version $$(python ambassador/VERSION.py)"

dev-bump: reg-check
	@echo "Bumping for development..."
	bump2version --allow-dirty --no-tag --no-commit \
	    --new-version `git describe --tags | sed s/^v//` commit
	@echo "Building version $$(python ambassador/VERSION.py)"

new-commit:
	$(MAKE) dev-bump
	$(MAKE) artifacts

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
	git commit $(VERSIONED) -m "v$(VERSION) [ci skip]"
	git tag -a v$(VERSION) -m "v$(VERSION)"

ambassador-rest.yaml: .ALWAYS
	sh templates/ambassador-rest.yaml.sh > ambassador-rest.yaml

istio/ambassador.yaml: .ALWAYS
	sh templates/ambassador-istio.yaml.sh > istio/ambassador.yaml

ambassador.yaml: ambassador-store.yaml ambassador-rest.yaml
	cat ambassador-store.yaml ambassador-rest.yaml > ambassador.yaml

docker-images: ambassador-image statsd-image cli-image

ambassador-image: .ALWAYS
	scripts/docker_build_maybe_push ambassador $(VERSION) ambassador

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push statsd $(VERSION) statsd

cli-image: .ALWAYS
	scripts/docker_build_maybe_push actl $(VERSION) actl
