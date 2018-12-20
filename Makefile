NAME=ambassador-ratelimit
PROFILE ?= dev

pkg = github.com/datawire/ambassador-ratelimit
bins = apictl apictl-key

include build-aux/shell.mk
include build-aux/go.mk
include build-aux/k8s.mk
include build-aux/kubernaut.mk
include build-aux/proxy.mk

export GOPATH
export GOBIN
export PATH:=$(GOBIN):$(PATH)

RATELIMIT_REPO=$(GOPATH)/src/github.com/lyft/ratelimit
RATELIMIT_VERSION=v1.3.0

PATCH=$(CURDIR)/ratelimit.patch

lyft-pull:
	git subtree pull --squash --prefix=vendor-ratelimit https://github.com/lyft/ratelimit.git $(RATELIMIT_VERSION)
	cd vendor-ratelimit && rm -f go.mod go.sum && go mod init github.com/lyft/ratelimit && git add go.mod
	git commit -m 'Run: make lyft-pull' || true
.PHONY: lyft-pull

$(RATELIMIT_REPO):
	mkdir -p $(RATELIMIT_REPO) && git clone https://github.com/lyft/ratelimit $(RATELIMIT_REPO)
	cd $(RATELIMIT_REPO) && git checkout -q $(RATELIMIT_VERSION)
	cd $(RATELIMIT_REPO) && git apply $(PATCH)

$(RATELIMIT_REPO)/vendor: $(RATELIMIT_REPO)
	cd $(RATELIMIT_REPO) && glide install

lyft-build: $(RATELIMIT_REPO)/vendor $(BIN)
	$(GO) install github.com/lyft/ratelimit/src/service_cmd && mv service_cmd ratelimit
	$(GO) install github.com/lyft/ratelimit/src/client_cmd && mv client_cmd ratelimit_client
	$(GO) install github.com/lyft/ratelimit/src/config_check_cmd && mv config_check_cmd ratelimit_check
.PHONY: lyft-build

lyft-build-image: $(RATELIMIT_REPO)/vendor $(BIN)
	$(IMAGE_GO) build -o image/ratelimit github.com/lyft/ratelimit/src/service_cmd
	$(IMAGE_GO) build -o image/ratelimit_client github.com/lyft/ratelimit/src/client_cmd
.PHONY: lyft-build-image

docker: env build-image lyft-build-image
	docker build . -t $(RATELIMIT_IMAGE)
	docker build intercept --target telepresence-proxy -t $(PROXY_IMAGE)
	docker build intercept --target telepresence-sidecar -t $(SIDECAR_IMAGE)
.PHONY: docker

docker-run: docker
	docker run -it $(IMAGE)
.PHONY: docker-run

# This is for managing minor diffs to upstream code. If we need
# anything more than minor diffs this probably won't work so well. We
# really don't want to have more than minor diffs though without a
# good reason.
diff:
	cd ${RATELIMIT_REPO} && git diff > $(PATCH)
.PHONY: diff

clean: $(CLUSTER).clean
	rm -rf ratelimit ratelimit_client image
.PHONY: clean

clobber: clean proxy.clobber k8s.clobber
	rm -rf $(RATELIMIT_REPO)
.PHONY: clobber
