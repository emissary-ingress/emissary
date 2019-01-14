NAME         = ambassador-ratelimit
# For docker.mk
DOCKER_IMAGE = quay.io/datawire/$(NAME)$(if $(findstring -,$(VERSION)),-dev):$(word 2,$(subst -, ,$(notdir $*)))-$(VERSION)
# For k8s.mk
K8S_IMAGES   = docker/ambassador-ratelimit docker/traffic-proxy docker/traffic-sidecar
K8S_ENV      = k8s/env.sh

include build-aux/common.mk
include build-aux/go-mod.mk
include build-aux/go-version.mk
include build-aux/help.mk
include build-aux/teleproxy.mk
include build-aux/docker.mk
include build-aux/k8s.mk

export PATH:=$(CURDIR)/bin_$(GOOS)_$(GOARCH):$(PATH)
export CGO_ENABLED=0

.DEFAULT_GOAL = help

#
# Lyft

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

#
# Docker images

docker/traffic-proxy.docker: docker/traffic-proxy/proxy
docker/traffic-proxy/%: bin_linux_amd64/%
	cp $< $@

docker/traffic-sidecar.docker: docker/traffic-sidecar/ambex
docker/traffic-sidecar.docker: docker/traffic-sidecar/sidecar
docker/traffic-sidecar/ambex:
	cd $(@D) && wget -q 'https://s3.amazonaws.com/datawire-static-files/ambex/0.1.0/ambex'
	chmod 755 $@
docker/traffic-sidecar/%: bin_linux_amd64/%
	cp $< $@

docker/ambassador-ratelimit.docker: docker/ambassador-ratelimit/apictl
docker/ambassador-ratelimit.docker: docker/ambassador-ratelimit/ratelimit
docker/ambassador-ratelimit.docker: docker/ambassador-ratelimit/ratelimit_check
docker/ambassador-ratelimit.docker: docker/ambassador-ratelimit/ratelimit_client
docker/ambassador-ratelimit/%: bin_linux_amd64/%
	cp $< $@

clean:
	rm -f docker/traffic-proxy/proxy
	rm -f docker/traffic-sidecar/sidecar
	rm -f docker/ambassador-ratelimit/apictl
	rm -f docker/ambassador-ratelimit/ratelimit
	rm -f docker/ambassador-ratelimit/ratelimit_check
	rm -f docker/ambassador-ratelimit/ratelimit_client
clobber:
	rm -f docker/traffic-sidecar/ambex

#
# Release

.PHONY: release release-%

release: ## Cut a release; upload binaries to S3 and Docker images to Quay
release: release-bin release-docker
release-bin: ## Upload binaries to S3
release-bin: release-apictl
release-bin: release-apictl-key
release-docker: ## Upload Docker images to Quay
release-docker: docker/ambassador-ratelimit.docker.push
release-docker: docker/traffic-proxy.docker.push
release-docker: docker/traffic-sidecar.docker.push

release-apictl release-apictl-key: release-%: bin_$(GOOS)_$(GOARCH)/%
	aws s3 cp --acl public-read $< 's3://datawire-static-files/$*/$(VERSION)/$(GOOS)/$(GOARCH)/$*'
