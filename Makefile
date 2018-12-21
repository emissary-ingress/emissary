NAME=ambassador-ratelimit
PROFILE ?= dev

include build-aux/common.mk
include build-aux/go.mk

include build-aux/shell.mk
include build-aux/k8s.mk
include build-aux/kubernaut.mk
include build-aux/proxy.mk

export PATH:=$(CURDIR)/bin_$(GOOS)_$(GOARCH):$(PATH)

RATELIMIT_VERSION=v1.3.0
lyft-pull:
	git subtree pull --squash --prefix=vendor-ratelimit https://github.com/lyft/ratelimit.git $(RATELIMIT_VERSION)
	cd vendor-ratelimit && rm -f go.mod go.sum && go mod init github.com/lyft/ratelimit && git add go.mod
	git commit -m 'Run: make lyft-pull' || true
.PHONY: lyft-pull

build: bin_$(GOOS)_$(GOARCH)/ratelimit
build: bin_$(GOOS)_$(GOARCH)/ratelimit_client
build: bin_$(GOOS)_$(GOARCH)/ratelimit_check

bin_%/ratelimit       : FORCE ; GO111MODULE=on go build -o $@ github.com/lyft/ratelimit/src/service_cmd
bin_%/ratelimit_client: FORCE ; GO111MODULE=on go build -o $@ github.com/lyft/ratelimit/src/client_cmd
bin_%/ratelimit_check : FORCE ; GO111MODULE=on go build -o $@ github.com/lyft/ratelimit/src/config_check_cmd

# `docker build` mumbo-jumbo
build-image: image/ratelimit
build-image: image/ratelimit_client
build-image: $(addprefix image/,$(notdir $(go.bins)))
.PHONY: build-image
image/%: bin_linux_amd64/%
	@mkdir -p $(@D)
	cp $< $@
docker: env build-image
	docker build . -t $(RATELIMIT_IMAGE)
	docker build intercept --target telepresence-proxy -t $(PROXY_IMAGE)
	docker build intercept --target telepresence-sidecar -t $(SIDECAR_IMAGE)
.PHONY: docker

# Utility targets
docker-run: docker
	docker run -it $(IMAGE)
.PHONY: docker-run

clean: $(CLUSTER).clean
	rm -rf image

clobber: proxy.clobber k8s.clobber
