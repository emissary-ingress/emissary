NAME            = ambassador-pro
# For docker.mk
DOCKER_IMAGE    = quay.io/datawire/$(NAME):$(notdir $*)-$(VERSION)
# For k8s.mk
K8S_IMAGES      = $(patsubst %/Dockerfile,%,$(wildcard docker/*/Dockerfile))
K8S_DIRS        = k8s-sidecar k8s-standalone
K8S_ENVS        = k8s-sidecar/env.sh tests/oauth-e2e/env.sh
# For go.mk
go.PLATFORMS    = linux_amd64 darwin_amd64

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
	cd vendor-ratelimit && rm -f go.mod && go mod init github.com/lyft/ratelimit && go mod tidy && go mod download && git add go.mod go.sum
	git commit -m 'Run: make lyft-pull' || true
.PHONY: lyft-pull

go-get: go-get-lyft
go-get-lyft:
	cd vendor-ratelimit && go mod download
.PHONY: go-get-lyft

lyft.bins  = ratelimit:github.com/lyft/ratelimit/src/service_cmd
lyft.bins += ratelimit_client:github.com/lyft/ratelimit/src/client_cmd
lyft.bins += ratelimit_check:github.com/lyft/ratelimit/src/config_check_cmd

# This mimics _go-common.mk
define lyft.bin.rule
bin_%/.tmp.$(word 1,$(subst :, ,$(lyft.bin))).tmp: go-get FORCE
	go build -o $$@ -o $$@ $(word 2,$(subst :, ,$(lyft.bin)))
bin_%/$(word 1,$(subst :, ,$(lyft.bin))): bin_%/.tmp.$(word 1,$(subst :, ,$(lyft.bin))).tmp
	if cmp -s $$< $$@; then rm -f $$< || true; else $(if $(CI),test ! -e $$@ && )mv -f $$< $$@; fi
endef
$(foreach lyft.bin,$(lyft.bins),$(eval $(lyft.bin.rule)))
build: $(addprefix bin_$(GOOS)_$(GOARCH)/,$(foreach lyft.bin,$(lyft.bins),$(word 1,$(subst :, ,$(lyft.bin)))))

#
# Docker images

docker/consul_connect_integration.docker: docker/consul_connect_integration/consul_connect_integration
docker/consul_connect_integration/%: bin_linux_amd64/%
	cp $< $@

docker/traffic-proxy.docker: docker/traffic-proxy/proxy
docker/traffic-proxy/%: bin_linux_amd64/%
	cp $< $@

docker/app-sidecar.docker: docker/app-sidecar/ambex
docker/app-sidecar.docker: docker/app-sidecar/sidecar
docker/app-sidecar/ambex:
	cd $(@D) && wget -q 'https://s3.amazonaws.com/datawire-static-files/ambex/0.1.0/ambex'
	chmod 755 $@
docker/app-sidecar/%: bin_linux_amd64/%
	cp $< $@

docker/amb-sidecar-ratelimit.docker: docker/amb-sidecar-ratelimit/apictl
docker/amb-sidecar-ratelimit.docker: docker/amb-sidecar-ratelimit/ratelimit
docker/amb-sidecar-ratelimit.docker: docker/amb-sidecar-ratelimit/ratelimit_check
docker/amb-sidecar-ratelimit.docker: docker/amb-sidecar-ratelimit/ratelimit_client
docker/amb-sidecar-ratelimit/%: bin_linux_amd64/%
	cp $< $@

docker/amb-sidecar-oauth.docker: docker/amb-sidecar-oauth/ambassador-oauth
docker/amb-sidecar-oauth/ambassador-oauth: bin_linux_amd64/ambassador-oauth
	cp $< $@

#
# Check

docker_tests =

# Generate the TLS secret
%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
k8s-standalone/02-ambassador-certs.yaml: k8s-standalone/cert.pem k8s-standalone/key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert k8s-standalone/cert.pem --key k8s-standalone/key.pem > $@

deploy: k8s-standalone/02-ambassador-certs.yaml
apply: k8s-standalone/02-ambassador-certs.yaml

tests/oauth-e2e/node_modules: tests/oauth-e2e/package.json $(wildcard tests/oauth-e2e/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@

check-consul-connect-integration:
	KUBECONFIG=$(KUBECONFIG) go run e2e-consul/main.go

check-e2e: ## Check: e2e tests
check-e2e: e2e-oauth/node_modules deploy proxy
	cd e2e-oauth && npm test
	$(MAKE) check-consul-connect-integration
.PHONY: check-e2e
check-intercept: ## Check: apictl traffic intercept
check-intercept: deploy proxy
	KUBECONFIG=$(KUBECONFIG) ./loop-intercept.sh
docker_tests += check-intercept

check-e2e: ## Check: oauth e2e tests
check-e2e: tests/oauth-e2e/node_modules deploy proxy
	cd tests/oauth-e2e && npm test
docker_tests += check-e2e

.PHONY: $(docker_tests)
ifneq ($(shell which docker 2>/dev/null),)
check: $(docker_tests)
else
check:
	@echo 'SKIPPING TESTS THAT REQUIRE DOCKER'
endif

#
# Clean

clean:
	rm -f docker/traffic-proxy/proxy
	rm -f docker/app-sidecar/sidecar
	rm -f docker/amb-sidecar-ratelimit/apictl
	rm -f docker/amb-sidecar-ratelimit/ratelimit
	rm -f docker/amb-sidecar-ratelimit/ratelimit_check
	rm -f docker/amb-sidecar-ratelimit/ratelimit_client
	rm -f docker/amb-sidecar-oauth/ambassador-oauth
	rm -f k8s-standalone/??-ambassador-certs.yaml k8s-standalone/*.pem
clobber:
	rm -f docker/app-sidecar/ambex
	rm -rf tests/oauth-e2e/node_modules

#
# Release

.PHONY: release release-%

release: ## Cut a release; upload binaries to S3 and Docker images to Quay
release: release-bin release-docker
release-bin: ## Upload binaries to S3
release-bin: $(foreach platform,$(go.PLATFORMS), release/bin_$(platform)/apictl     )
release-bin: $(foreach platform,$(go.PLATFORMS), release/bin_$(platform)/apictl-key )
release-docker: ## Upload Docker images to Quay
release-docker: $(addsuffix .docker.push,$(K8S_IMAGES))

_release_os   = $(word 2,$(subst _, ,$(@D)))
_release_arch = $(word 3,$(subst _, ,$(@D)))
release/%: %
	aws s3 cp --acl public-read $< 's3://datawire-static-files/$(@F)/$(VERSION)/$(_release_os)/$(_release_arch)/$(@F)'
