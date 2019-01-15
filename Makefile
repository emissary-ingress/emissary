NAME            = ambassador-pro
# For docker.mk
DOCKER_IMAGE    = quay.io/datawire/$(NAME):$(word 2,$(subst -, ,$(notdir $*)))-$(VERSION)
# For k8s.mk
K8S_IMAGES      = docker/ambassador-oauth docker/ambassador-ratelimit docker/traffic-proxy docker/traffic-sidecar
K8S_DIRS        = k8s e2e-oauth/k8s
K8S_ENVS        = k8s/env.sh e2e-oauth/env.sh

export CGO_ENABLED = 0

include build-aux/go-mod.mk
include build-aux/go-version.mk
include build-aux/k8s.mk
include build-aux/teleproxy.mk
include build-aux/help.mk

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

docker/ambassador-oauth.docker: docker/ambassador-oauth/ambassador-oauth
docker/ambassador-oauth/ambassador-oauth: bin_linux_amd64/ambassador-oauth
	cp $< $@

#
# Check

# Generate the TLS secret
%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
e2e-oauth/k8s/02-ambassador-certs.yaml: e2e-oauth/k8s/cert.pem e2e-oauth/k8s/key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert e2e-oauth/k8s/cert.pem --key e2e-oauth/k8s/key.pem > $@

deploy: e2e-oauth/k8s/02-ambassador-certs.yaml
apply: e2e-oauth/k8s/02-ambassador-certs.yaml

e2e-oauth/node_modules: e2e-oauth/package.json $(wildcard e2e-oauth/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@

check-e2e: ## Check: e2e tests
check-e2e: e2e-oauth/node_modules deploy
	$(MAKE) proxy
	cd e2e-oauth && npm test
	$(MAKE) unproxy
.PHONY: check-e2e

ifneq ($(shell which docker 2>/dev/null),)
check: check-e2e
else
check:
	@echo 'SKIPPING OAUTH E2E TESTS'
endif

#
# Clean

clean:
	rm -f docker/traffic-proxy/proxy
	rm -f docker/traffic-sidecar/sidecar
	rm -f docker/ambassador-ratelimit/apictl
	rm -f docker/ambassador-ratelimit/ratelimit
	rm -f docker/ambassador-ratelimit/ratelimit_check
	rm -f docker/ambassador-ratelimit/ratelimit_client
	rm -f e2e-oauth/k8s/??-ambassador-certs.yaml e2e-oauth/k8s/*.pem
clobber:
	rm -f docker/traffic-sidecar/ambex
	rm -rf e2e-oauth/node_modules

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

push-tagged-image: ## docker push
push-tagged-image: docker/ambassador-oauth.docker.push
