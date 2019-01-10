NAME=ambassador-pro

include build-aux/go-workspace.mk
include build-aux/kubeapply.mk
include build-aux/help.mk
include build-aux/teleproxy.mk
include build-aux/kubernaut-ui.mk
.DEFAULT_GOAL = help

export CGO_ENABLED = 0

VERSION=0.0.2

PRD_DOCKER_REGISTRY = quay.io/datawire
PRD_DOCKER_REPO = ambassador-pro
PRD_VERSION = $(or $(TRAVIS_TAG),$(VERSION))
PRD_IMAGE = $(PRD_DOCKER_REGISTRY)/$(PRD_DOCKER_REPO):$(PRD_VERSION)

ifeq ($(GOOS),darwin)
LOCALHOST = host.docker.internal
else
LOCALHOST = localhost
endif
DEV_DOCKER_REGISTRY = $(LOCALHOST):31000
DEV_DOCKER_REPO = ambassador-pro
DEV_VERSION = $(or $(TRAVIS_COMMIT),$(shell git describe --match NoThInGEvErMaTcHeS --always --abbrev=40 --dirty))
DEV_IMAGE = $(DEV_DOCKER_REGISTRY)/$(DEV_DOCKER_REPO):$(DEV_VERSION)

define help.body
# Unlike most Makefiles, the output of `make build` isn't a file, but
# is the Docker image $$(DEV_IMAGE).
#
#   DEV_IMAGE = $(value DEV_IMAGE)
#             = $(DEV_IMAGE)
#
#   PRD_IMAGE = $(value PRD_IMAGE)
#             = $(PRD_IMAGE)
#
#   GOBIN     = $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
endef

#
# Main

# The main "output" of the Makefile is actually a Docker image, not a
# file.
build: docker/ambassador-pro/ambassador-oauth
	docker build -t $(DEV_IMAGE) docker/ambassador-pro
docker/ambassador-pro/ambassador-oauth: bin_linux_amd64/ambassador-oauth
	cp $< $@

clean:
	rm -f key.pem cert.pem scripts/??-ambassador-certs.yaml

#
# Check

%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
key.pem: $(CURDIR)/key.pem
cert.pem: $(CURDIR)/cert.pem

scripts/02-ambassador-certs.yaml: cert.pem key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert cert.pem --key key.pem > $@

deploy: ## Deploy $(DEV_IMAGE) to a kubernaut.io cluster
deploy: build $(KUBEAPPLY) $(KUBECONFIG) env.sh scripts/02-ambassador-certs.yaml
	$(KUBEAPPLY) -f scripts/00-registry.yaml
	{ \
	    kubectl port-forward --namespace=docker-registry deployment/registry 31000:5000 & \
	    trap "kill $$!; wait" EXIT; \
	    while ! curl -i http://localhost:31000/ 2>/dev/null; do sleep 1; done; \
	    docker push $(DEV_IMAGE); \
	}
	set -a && IMAGE=$(foreach LOCALHOST,localhost,$(DEV_IMAGE)) && . ./env.sh && $(KUBEAPPLY) $(addprefix -f ,$(wildcard scripts/*.yaml))
.PHONY: deploy

e2e/node_modules: e2e/package.json $(wildcard e2e/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@

check-e2e: ## Check: e2e tests
check-e2e: e2e/node_modules deploy
	$(MAKE) proxy
	cd e2e && npm test
	$(MAKE) unproxy
.PHONY: check-e2e
check: check-e2e

#
# Utility targets

push-tagged-image: ## docker push $(PRD_IMAGE)
#push-tagged-image: build
	docker tag $(DEV_IMAGE) $(PRD_IMAGE)
	docker push $(PRD_IMAGE)
.PHONY: push-tagged-image

run: ## Run ambassador-oauth locally
run: bin_$(GOOS)_$(GOARCH)/ambassador-oauth
	./$<
.PHONY: run
