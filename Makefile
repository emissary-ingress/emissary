all: dev

VERSION=$(shell python scripts/versioner.py --magic-pre)

.ALWAYS:

dev: version-check reg-check versions docker-images yaml-files

travis-images: version-check reg-check versions docker-images

travis-website: version-check website

version-check:
	@if [ -z "$(VERSION)" ]; then \
		echo "Nothing needs to be built" >&2 ;\
		exit 1 ;\
	fi

reg-check:
	@if [ -z "$$DOCKER_REGISTRY" ]; then \
	    echo "DOCKER_REGISTRY must be set" >&2 ;\
	    exit 1 ;\
	fi

versions:
	@echo "Building $(VERSION)"
	for file in actl ambassador; do \
	    sed -e "s/{{VERSION}}/$(VERSION)/g" < VERSION-template.py > $$file/VERSION.py; \
	done

artifacts: docker-images website

tag:
	git tag -a v$(VERSION) -m "v$(VERSION)"

yaml-files:
	VERSION=$(VERSION) sh scripts/build-yaml.sh

docker-images: ambassador-image statsd-image cli-image

ambassador-image: .ALWAYS
	scripts/docker_build_maybe_push ambassador $(VERSION) ambassador

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push statsd $(VERSION) statsd

cli-image: .ALWAYS
	scripts/docker_build_maybe_push actl $(VERSION) actl

website: yaml-files
	VERSION=$(VERSION) docs/build-website.sh
