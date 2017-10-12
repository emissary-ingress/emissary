all: dev

VERSION=$(shell python scripts/versioner.py --bump --magic-pre)

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
	for file in actl ambassador-core; do \
	    sed -e "s/{{VERSION}}/$(VERSION)/g" < VERSION-template.py > $$file/VERSION.py; \
	done

artifacts: docker-images website

tag:
	git tag -a v$(VERSION) -m "v$(VERSION)"

yaml-files:
	VERSION=$(VERSION) sh scripts/build-yaml.sh

ambassador-test:
	sh scripts/ambassador-test.sh

docker-images: ambassador-core-image ambassador-image statsd-image cli-image

ambassador-core-image: ambassador-test .ALWAYS
	scripts/docker_build_maybe_push ambassador-core $(VERSION) ambassador-core

ambassador-image: ambassador-test .ALWAYS
	VERSION=$(VERSION) python scripts/template.py < ambassador/Dockerfile.template > ambassador/Dockerfile
	scripts/docker_build_maybe_push ambassador $(VERSION) ambassador

statsd-image: .ALWAYS
	scripts/docker_build_maybe_push statsd $(VERSION) statsd

cli-image: .ALWAYS
	scripts/docker_build_maybe_push actl $(VERSION) actl

website: yaml-files
	VERSION=$(VERSION) docs/build-website.sh

clean:
	rm -rf docs/yaml docs/_book docs/_site docs/node_modules
	rm -rf app.json
	rm -rf ambassador-core/__pycache__  ambassador-core/envoy-test.json
	rm -rf ambassador/__pycache__  ambassador/Dockerfile
