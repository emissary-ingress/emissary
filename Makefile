NAME=ambassador-pro

include build-aux/kubeapply.mk
include build-aux/help.mk
include build-aux/teleproxy.mk
include build-aux/kubernaut-ui.mk
.DEFAULT_GOAL = help

VERSION=0.0.2

PRD_DOCKER_REGISTRY = quay.io/datawire
PRD_DOCKER_REPO = ambassador-pro
PRD_VERSION = $(or $(TRAVIS_TAG),$(VERSION))
PRD_IMAGE = $(PRD_DOCKER_REGISTRY)/$(PRD_DOCKER_REPO):$(PRD_VERSION)

DEV_DOCKER_REGISTRY = localhost:31000
DEV_DOCKER_REPO = ambassador-pro
DEV_VERSION = $(or $(TRAVIS_COMMIT),$(shell git describe --no-match --always --abbrev=40 --dirty))
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

# The main "output" of the Makefile is actually a Docker image, not a
# file.
.PHONY: build
build: ## docker build -t $(DEV_IMAGE)
	docker build . -t $(DEV_IMAGE)

%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
key.pem: $(CURDIR)/key.pem
cert.pem: $(CURDIR)/cert.pem

scripts/02-ambassador-certs.yaml: cert.pem key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert cert.pem --key key.pem > $@

.PHONY: deploy
deploy: ## Deploy $(DEV_IMAGE) to a kubernaut.io cluster
deploy: build $(KUBEAPPLY) $(KUBECONFIG) env.sh scripts/02-ambassador-certs.yaml
	$(KUBEAPPLY) -f scripts/00-registry.yaml
	{ \
	    kubectl port-forward --namespace=docker-registry deployment/registry 31000:5000 & \
	    trap "kill $$!; wait" EXIT; \
	    while ! curl -i http://localhost:31000/ 2>/dev/null; do sleep 1; done; \
	    docker push $(DEV_IMAGE); \
	}
	set -a && IMAGE=$(DEV_IMAGE) && . ./env.sh && $(KUBEAPPLY) $(addprefix -f ,$(wildcard scripts/*.yaml))

.PHONY: push-tagged-image
push-tagged-image: ## docker push $(PRD_IMAGE)
#push-tagged-image: build
	docker tag $(DEV_IMAGE) $(PRD_IMAGE)
	docker push $(PRD_IMAGE)

.PHONY: run
run: ## Run ambassador-oauth locally
run: install
	@echo " >>> running oauth server"
	ambassador-oauth 

.PHONY: install
install: ## Compile ambassador-oauth (to $GOBIN)
install: vendor
	@echo " >>> building"
	go install ./cmd/...

.PHONY: clean
clean: ## Clean
	@echo " >>> cleaning compiled objects and binaries"
	rm -f key.pem cert.pem scripts/??-ambassador-certs.yaml
	go clean -i ./...

.PHONY: test
test: ## Check: unit tests
	@echo " >>> testing code.."
	go test ./...

vendor: ## Update the ./vendor/ directory based on Gopkg.toml
	@echo " >>> installing dependencies"
	dep ensure -vendor-only

format: ## Adjust the source code per `go fmt`
	@echo " >>> running format"
	go fmt ./...

check_format: ## Check: go fmt
	@echo " >>> checking format"
	if [ $$(go fmt $$(go list ./... | grep -v vendor/)) ]; then exit 1; fi

e2e_build: ## Build a oauth-client Docker image, for e2e testing
	@echo " >>> building docker for e2e testing"
	docker build -t e2e/test:latest e2e

e2e_test: ## Check: e2e tests
e2e_test: e2e_build deploy
	@echo " >>> running e2e tests"
	$(MAKE) proxy
	docker run --rm e2e/test:latest
	$(MAKE) unproxy
