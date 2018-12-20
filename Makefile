NAME=ambassador-ratelimit
PROFILE ?= dev

pkg = github.com/datawire/ambassador-ratelimit
bins = apictl apictl-key

include build-aux/common.mk
include build-aux/shell.mk
include build-aux/go.mk
include build-aux/k8s.mk
include build-aux/kubernaut.mk
include build-aux/proxy.mk

export GOPATH
export GOBIN
export PATH:=$(GOBIN):$(PATH)

RATELIMIT_VERSION=v1.3.0

lyft-pull:
	git subtree pull --squash --prefix=vendor-ratelimit https://github.com/lyft/ratelimit.git $(RATELIMIT_VERSION)
	cd vendor-ratelimit && rm -f go.mod go.sum && go mod init github.com/lyft/ratelimit && git add go.mod
	git commit -m 'Run: make lyft-pull' || true
.PHONY: lyft-pull

lyft-build: ## Build programs imported from github.com/lyft/ratelimit
lyft-build: bin_$(GOOS)_$(GOARCH)/ratelimit
lyft-build: bin_$(GOOS)_$(GOARCH)/ratelimit_client
lyft-build: bin_$(GOOS)_$(GOARCH)/ratelimit_check
.PHONY: lyft-build

lyft-build-image: image/ratelimit
lyft-build-image: image/ratelimit_client
.PHONY: lyft-build-image
image/%: bin_linux_amd64/%
	@mkdir -p $(@D)
	cp $< $@

bin_%/ratelimit       : FORCE ; go build -o $@ github.com/lyft/ratelimit/src/service_cmd
bin_%/ratelimit_client: FORCE ; go build -o $@ github.com/lyft/ratelimit/src/client_cmd
bin_%/ratelimit_check : FORCE ; go build -o $@ github.com/lyft/ratelimit/src/config_check_cmd

docker: env build-image lyft-build-image
	docker build . -t $(RATELIMIT_IMAGE)
	docker build intercept --target telepresence-proxy -t $(PROXY_IMAGE)
	docker build intercept --target telepresence-sidecar -t $(SIDECAR_IMAGE)
.PHONY: docker

docker-run: docker
	docker run -it $(IMAGE)
.PHONY: docker-run

clean: $(CLUSTER).clean
	rm -rf -- bin_* image
.PHONY: clean

clobber: clean proxy.clobber k8s.clobber
.PHONY: clobber
