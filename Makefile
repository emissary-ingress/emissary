# file: Makefile

.FORCE:
.PHONY: .FORCE clean version setup-develop print-vars docker-login docker-push docker-images docker-tags publish-website e2e

# GIT_BRANCH on TravisCI needs to be set via some external custom logic. Default to Git native mechanism or use what is
# defined already.
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git rev-parse --short HEAD)

ifndef VERSION
VERSION = $(GIT_COMMIT)
endif

DOCKER_REGISTRY ?= quay.io
DOCKER_OPTS =

NETLIFY_SITE=datawire-ambassador

AMBASSADOR_DOCKER_REPO ?= datawire/ambassador-gh369
AMBASSADOR_DOCKER_TAG ?= $(GIT_COMMIT)
AMBASSADOR_DOCKER_IMAGE ?= $(AMBASSADOR_DOCKER_REPO):$(AMBASSADOR_DOCKER_TAG)

STATSD_DOCKER_REPO ?= datawire/ambassador-statsd-gh369
STATSD_DOCKER_TAG ?= $(GIT_COMMIT)
STATSD_DOCKER_IMAGE ?= $(STATSD_DOCKER_REPO):$(STATSD_DOCKER_TAG)

RELEASE_TYPE ?= unstable

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
	@echo "GIT_BRANCH              = $(GIT_BRANCH)"
	@echo "GIT_COMMIT              = $(GIT_COMMIT)"
	@echo "VERSION                 = $(VERSION)"
	@echo "DOCKER_REGISTRY         = $(DOCKER_REGISTRY)"
	@echo "DOCKER_OPTS             = $(DOCKER_OPTS)"
	@echo "RELEASE_TYPE            = $(RELEASE_TYPE)"
	@echo "AMBASSADOR_DOCKER_REPO  = $(AMBASSADOR_DOCKER_REPO)"
	@echo "AMBASSADOR_DOCKER_TAG   = $(AMBASSADOR_DOCKER_TAG)"
	@echo "AMBASSADOR_DOCKER_IMAGE = $(AMBASSADOR_DOCKER_IMAGE)"
	@echo "STATSD_DOCKER_REPO      = $(STATSD_DOCKER_REPO)"
	@echo "STATSD_DOCKER_TAG       = $(STATSD_DOCKER_TAG)"
	@echo "STATSD_DOCKER_IMAGE     = $(STATSD_DOCKER_IMAGE)"

ambassador-docker-image:
	docker build $(DOCKER_OPTS) -t $(AMBASSADOR_DOCKER_IMAGE) ./ambassador

statsd-docker-image:
	docker build $(DOCKER_OPTS) -t $(STATSD_DOCKER_IMAGE) ./statsd

docker-login:
	@if [ -z $(DOCKER_USERNAME) ]; then echo 'DOCKER_USERNAME not defined'; exit 1; fi
	@if [ -z $(DOCKER_PASSWORD) ]; then echo 'DOCKER_PASSWORD not defined'; exit 1; fi

	@printf "$(DOCKER_PASSWORD)" | docker login -u="$(DOCKER_USERNAME)" --password-stdin $(DOCKER_REGISTRY)

docker-images: ambassador-docker-image statsd-docker-image

docker-push: docker-tags
	docker push $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE)

docker-tags: docker-images
	docker tag $(AMBASSADOR_DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)
	docker tag $(STATSD_DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(STATSD_DOCKER_IMAGE)

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

e2e: docker-images docker-push e2e-versioned-manifests
	bash end-to-end/testall.sh

setup-develop:
	cd ambassador && python setup.py develop

test: version setup-develop
	cd ambassador && pytest --tb=short --cov=ambassador --cov-report term-missing

release: docker-tags
	@if [[ "$(VERSION)" != "$(GIT_COMMIT)" ]]; then \
		docker pull $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)
		docker pull $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_IMAGE)

		docker push $(DOCKER_REGISTRY)/$(AMBASSADOR_DOCKER_REPO):$(VERSION); \
		docker push $(DOCKER_REGISTRY)/$(STATSD_DOCKER_REPO):$(VERSION); \
	else \
		@printf "`make release` can only be run when VERSION is explicitly set to a different value than GIT_COMMIT!\n"
	fi

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

publish-website:
	RELEASE_TYPE=$(RELEASE_TYPE) \
	NETLIFY_SITE=$(NETLIFY_SITE) \
		bash ./releng/publish-website.sh


# ------------------------------------------------------------------------------
# CI Targets
# ------------------------------------------------------------------------------

# ------------------------------------------------------------------------------
# Function Definitions
# ------------------------------------------------------------------------------

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = $(strip $(foreach 1,$1, $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),, $(error Undefined $1$(if $2, ($2))))