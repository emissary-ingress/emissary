# file: Makefile

GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT=$(shell git rev-parse --short HEAD)

VERSION=$(GIT_COMMIT)

DOCKER_REGISTRY ?= quay.io
DOCKER_OPTS =

AMBASSADOR_DOCKER_REPO ?= datawire/ambassador-gh369
AMBASSADOR_DOCKER_TAG ?= $(GIT_COMMIT)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

STATSD_DOCKER_REPO ?= datawire/ambassador-statsd-gh369
STATSD_DOCKER_TAG ?= $(GIT_COMMIT)
STATSD_DOCKER_IMAGE ?= $(STATSD_DOCKER_REPO):$(STATSD_DOCKER_TAG)

clean:
	rm -rf docs/yaml docs/_book docs/_site docs/node_modules
	rm -rf helm/*.tgz
	rm -rf app.json
	rm -rf ambassador/ambassador/VERSION.py*
	rm -rf ambassador/build ambassador/dist ambassador/ambassador.egg-info ambassador/__pycache__
	find . \( -name .coverage -o -name .cache -o -name __pycache__ \) -print0 | xargs -0 rm -rf
	find ambassador/tests \
		\( -name '*.out' -o -name 'envoy.json' -o -name 'intermediate.json' \) -print0 \
		| xargs -0 rm -f
	rm -rf end-to-end/ambassador-deployment.yaml end-to-end/ambassador-deployment-mounts.yaml
	find end-to-end \( -name 'check-*.json' -o -name 'envoy.json' \) -print0 | xargs -0 rm -f

print-vars:
	@echo "GIT_BRANCH      = $(GIT_BRANCH)"
	@echo "GIT_COMMIT      = $(GIT_COMMIT)"
	@echo "DOCKER_REGISTRY = $(DOCKER_REGISTRY)"
	@echo "DOCKER_REPO     = $(DOCKER_REPO)"
	@echo "DOCKER_TAG      = $(DOCKER_TAG)"
	@echo "DOCKER_IMAGE    = $(DOCKER_IMAGE)"

ambassador-docker-image:
	docker build $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) ./ambassador

statsd-docker-image:
	docker build $(DOCKER_OPTS) -t $(STATSD_DOCKER_IMAGE) ./statsd

docker-images: ambassador-docker-image statsd-docker-image

docker-push:
	docker tag $(AMBASSADOR_DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)
	docker tag $(STATSD_DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE)

	docker push $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE)

	@if [[ "$(VERSION)" != "$(GIT_COMMIT)" ]]; then \
		docker push $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE); \
		docker push $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE); \
	fi

version:
	$(call check_defined, VERSION, VERSION is not set)
	@echo "Generating and templating version information -> $(VERSION)"
	sed -e "s/{{VERSION}}/$(VERSION)/g" < VERSION-template.py > ambassador/ambassador/VERSION.py

e2e-versioned-manifests:
	$(eval AMBASSADOR_DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE))
	$(eval STATSD_DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE))

	sed -e "s|{{AMBASSADOR_DOCKER_IMAGE}}|$(AMBASSADOR_DOCKER_IMAGE)|g;s|{{STATSD_DOCKER_IMAGE}}|$(STATSD_DOCKER_IMAGE)|g" \
		< end-to-end/ambassador-no-mounts.yaml \
		> end-to-end/ambassador-deployment.yaml

	sed -e "s|{{AMBASSADOR_DOCKER_IMAGE}}|$(AMBASSADOR_DOCKER_IMAGE)|g;s|{{STATSD_DOCKER_IMAGE}}|$(STATSD_DOCKER_IMAGE)|g" \
		< end-to-end/ambassador-with-mounts.yaml \
		> end-to-end/ambassador-deployment-mounts.yaml

website-yaml:
	$(eval AMBASSADOR_DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE))
	$(eval STATSD_DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE))

	mkdir -p docs/yaml
	cp -R templates/* docs/yaml
	find ./docs/yaml \
		-type f \
		-exec sed \
			-i''\
			-e 's|{{AMBASSADOR_DOCKER_IMAGE}}|$(AMBASSADOR_DOCKER_IMAGE)|g;s|{{STATSD_DOCKER_IMAGE}}|$(STATSD_DOCKER_IMAGE)|g' \
			{} \;

website: website-yaml
	VERSION=$(VERSION) bash docs/build-website.sh

e2e: ambassador-docker-image statsd-docker-image docker-push e2e-versioned-manifests
	bash end-to-end/testall.sh

setup-develop:
	cd ambassador && python setup.py develop

test: setup-develop version
	cd ambassador && pytest --tb=short --cov=ambassador --cov-report term-missing

# --------------------
# Function Definitions
# --------------------

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = $(strip $(foreach 1,$1, $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),, $(error Undefined $1$(if $2, ($2))))