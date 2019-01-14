NAME            = ambassador-ratelimit
REGISTRY        = quay.io
NAMESPACE       = datawire
REPO            = $(NAMESPACE)/$(NAME)$(if $(findstring -,$(VERSION)),-dev)
RATELIMIT_IMAGE = $(REGISTRY)/$(REPO):ratelimit-$(VERSION)
PROXY_IMAGE     = $(REGISTRY)/$(REPO):proxy-$(VERSION)
SIDECAR_IMAGE   = $(REGISTRY)/$(REPO):sidecar-$(VERSION)

include build-aux/common.mk
include build-aux/go-mod.mk
include build-aux/go-version.mk
include build-aux/help.mk
include build-aux/teleproxy.mk
include build-aux/kubernaut-ui.mk
include build-aux/kubeapply.mk
include build-aux/k8s.mk

export PATH:=$(CURDIR)/bin_$(GOOS)_$(GOARCH):$(PATH)
export CGO_ENABLED=0

#

.DEFAULT_GOAL = help

RATELIMIT_VERSION=v1.3.0
lyft-pull: # Update vendor-ratelimit from github.com/lyft/ratelimit.git
	git subtree pull --squash --prefix=vendor-ratelimit https://github.com/lyft/ratelimit.git $(RATELIMIT_VERSION)
	cd vendor-ratelimit && rm -f go.mod go.sum && go mod init github.com/lyft/ratelimit && git add go.mod
	git commit -m 'Run: make lyft-pull' || true
.PHONY: lyft-pull

lyft.bins  = ratelimit:github.com/lyft/ratelimit/src/service_cmd
lyft.bins += ratelimit_client:github.com/lyft/ratelimit/src/client_cmd
lyft.bins += ratelimit_check:github.com/lyft/ratelimit/src/config_check_cmd

# This mimics _go-common.mk
define lyft.bin.rule
bin_%/.tmp.$(word 1,$(subst :, ,$(lyft.bin))).tmp: go-get FORCE
	go build -o $$@ -o $$@ $(word 2,$(subst :, ,$(lyft.bin)))
bin_%/$(word 1,$(subst :, ,$(lyft.bin))): bin_%/.tmp.$(word 1,$(subst :, ,$(lyft.bin))).tmp
	if cmp -s $$< $$@; then rm -f $$< || true; else mv -f $$< $$@; fi
endef
$(foreach lyft.bin,$(lyft.bins),$(eval $(lyft.bin.rule)))
build: $(addprefix bin_$(GOOS)_$(GOARCH)/,$(foreach lyft.bin,$(lyft.bins),$(word 1,$(subst :, ,$(lyft.bin)))))

# `docker build` mumbo-jumbo
build-image: image/ratelimit
build-image: image/ratelimit_client
build-image: $(addprefix image/,$(notdir $(go.bins)))
.PHONY: build-image
image/%: bin_linux_amd64/%
	@mkdir -p $(@D)
	cp $< $@
docker: build-image
	docker build . -t $(RATELIMIT_IMAGE)
	docker build . -f intercept/Dockerfile --target telepresence-proxy -t $(PROXY_IMAGE)
	docker build . -f intercept/Dockerfile --target telepresence-sidecar -t $(SIDECAR_IMAGE)
.PHONY: docker

# Utility targets
docker-run: docker
	docker run -it $(IMAGE)
.PHONY: docker-run

clean:
	rm -rf image
