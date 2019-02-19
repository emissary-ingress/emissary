NAME            = ambassador-pro
# For docker.mk
# If you change DOCKER_IMAGE, you'll also need to change the image
# names in `cmd/apictl/traffic.go`.
DOCKER_IMAGE    = quay.io/datawire/ambassador_pro:$(notdir $*)-$(VERSION)
# For k8s.mk
K8S_IMAGES      = $(patsubst %/Dockerfile,%,$(wildcard docker/*/Dockerfile))
K8S_DIRS        = k8s-sidecar k8s-standalone k8s-localdev
K8S_ENVS        = k8s-env.sh
# For go.mk
go.PLATFORMS    = linux_amd64 darwin_amd64

export CGO_ENABLED = 0
export SCOUT_DISABLE = 1

include build-aux/go-mod.mk
include build-aux/go-version.mk
include build-aux/k8s.mk
include build-aux/teleproxy.mk
include build-aux/pidfile.mk
include build-aux/help.mk

.DEFAULT_GOAL = help

ifeq ($(GOOS)_$(GOARCH),linux_amd64)
bin_linux_amd64/amb-sidecar: CGO_ENABLED=1
endif

status: ## Report on the status of Kubernaut and Teleproxy
status: status-pro-tel
.PHONY: status

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

lyft.bins  = ratelimit_client:github.com/lyft/ratelimit/src/client_cmd
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
#
# This assumes that each Docker image wants a binary with the same
# name as the image.  That's a safe assumption so far, and forces us
# to name things in a consistent manor.
define docker.bins_rule
$(image).docker: $(image)/$(notdir $(image))
$(image)/%: bin_linux_amd64/%
	cp $$< $$@
endef
$(foreach image,$(K8S_IMAGES),$(eval $(docker.bins_rule)))

docker/app-sidecar.docker: docker/app-sidecar/ambex
docker/app-sidecar/ambex:
	cd $(@D) && wget -q 'https://s3.amazonaws.com/datawire-static-files/ambex/0.1.0/ambex'
	chmod 755 $@

#
# Deploy

# Generate the TLS secret
%/cert.pem %/key.pem: %/namespace.txt
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.$$(cat $<).svc.cluster.local"
%/02-ambassador-certs.yaml: %/cert.pem %/key.pem %/namespace.txt
	kubectl --namespace="$$(cat $*/namespace.txt)" create secret tls --dry-run --output=yaml ambassador-certs --cert $*/cert.pem --key $*/key.pem > $@

deploy: $(addsuffix /02-ambassador-certs.yaml,$(K8S_DIRS))
apply: $(addsuffix /02-ambassador-certs.yaml,$(K8S_DIRS))

#
# Local Dev

launch-pro-tel: ## (LocalDev) Launch Telepresence for the APro pod
launch-pro-tel: build-aux/tel-pro.pid
.PHONY: launch-pro-tel
build-aux/tel-pro.pid: apply proxy
	@if ! curl -s -o /dev/null ambassador-pro.localdev:38888; then \
		echo "Launching Telepresence..."; \
		rm -f pro-env.tmp; \
		telepresence \
			--logfile build-aux/tel-pro.log --env-file pro-env.tmp \
			--namespace localdev -d ambassador-pro -m inject-tcp --mount false \
			--expose 6070 --expose 8080 --expose 8081 --expose 38888 \
			--run python3 -m http.server --bind 127.0.0.1 38888 \
			> /dev/null 2>&1 & echo $$! > build-aux/tel-pro.pid ; \
	fi
	@for i in $$(seq 127); do \
		if curl -s -o /dev/null ambassador-pro.localdev:38888; then \
			exit 0; \
		fi; \
		sleep 1; \
	done; echo "ERROR: Telepresence failed. See build-aux/tel-pro.log"; exit 1
	@if [ -s pro-env.tmp ]; then \
		echo "KUBECONFIG=$(KUBECONFIG)" >> pro-env.tmp; \
		echo "RLS_RUNTIME_DIR=$(or $(XDG_RUNTIME_DIR),$(TMPDIR),/tmp)/amb" >> pro-env.tmp; \
		mv -f pro-env.tmp pro-env.sh; \
	elif ! grep -q "^KUBECONFIG=" pro-env.sh; then \
		echo "ERROR: Telepresence did not populate pro-env.tmp"; \
		echo "See build-aux/tel-pro.log"; \
		exit 1; \
	fi
	@echo "Telepresence UP!"
kill-pro-tel: ## (LocalDev) Kill the running Telepresence
kill-pro-tel: build-aux/tel-pro.pid.clean
	rm -f pro-env.sh pro-env.tmp
.PHONY: kill-pro-tel
tail-pro-tel: ## (LocalDev) Tail the logs of the running/last Telepresence
	tail build-aux/tel-pro.log
.PHONY: tail-pro-tel
status-pro-tel: ## (LocalDev) Fail if Telepresence is not running
status-pro-tel: status-proxy
	@if curl -s -o /dev/null ambassador-pro.localdev:38888; then \
		echo "Telepresence okay!"; \
	else \
		echo "Telepresence is not running."; \
		exit 1; \
	fi
.PHONY: status-pro-tel
clean: kill-pro-tel
help-local-dev: ## (LocalDev) Describe how to use local dev features
	@echo "In the localdev namespace, the pro container has been replaced with"
	@echo "Telepresence. You will need to run the relevant binaries on your own"
	@echo "machine if you wish to use the Ambassador in this namespace."
	@echo "  https://ambassador.localdev.svc.cluster.local/"
	@echo
	@echo "A copy of the remote environment is available in pro-env.sh and"
	@echo "KUBECONFIG is also set in that file."
	@echo
	@echo "make run-auth        rebuild and run auth with debug logging"
	@echo "make launch-pro-tel  relaunch Telepresence if needed"
	@echo
	@echo "Launch auth manually:"
	@echo '  env $$(cat pro-env.sh)' "bin_$(GOOS)_$(GOARCH)/amb-sidecar auth"
.PHONY: help-local-dev
run-auth: ## (LocalDev) Build and launch the auth service locally
run-auth: bin_$(GOOS)_$(GOARCH)/amb-sidecar
	env $$(cat pro-env.sh) APP_LOG_LEVEL=debug bin_$(GOOS)_$(GOARCH)/amb-sidecar main
.PHONY: run-auth

#
# Check

HAVE_DOCKER := $(shell which docker 2>/dev/null)

check: $(if $(HAVE_DOCKER),deploy proxy)
test-suite.tap: tests/local.tap tests/cluster.tap

check-local: ## Check: Run only tests that do not talk to the cluster
check-local: lint go-build
	$(MAKE) tests/local-all.tap.summary
.PHONY: check-local
tests/local-all.tap: build-aux/go-test.tap tests/local.tap
	@./build-aux/tap-driver cat $^ > $@
tests/local.tap: $(patsubst %.test,%.tap,$(wildcard tests/local/*.test))
tests/local.tap: $(patsubst %.tap.gen,%.tap,$(wildcard tests/local/*.tap.gen))
tests/local.tap:
	@./build-aux/tap-driver cat $^ > $@

tests/cluster.tap: $(patsubst %.test,%.tap,$(wildcard tests/cluster/*.test))
tests/cluster.tap: $(patsubst %.tap.gen,%.tap,$(wildcard tests/cluster/*.tap.gen))
tests/cluster.tap:
	@./build-aux/tap-driver cat $^ > $@

tests/cluster/oauth-e2e/node_modules: tests/cluster/oauth-e2e/package.json $(wildcard tests/cluster/oauth-e2e/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@
check tests/cluster/oauth-e2e.tap: tests/cluster/oauth-e2e/node_modules

#
# Clean

clean:
	rm -f tests/*.log tests/*.tap tests/*/*.log tests/*/*.tap
	rm -f docker/traffic-proxy/traffic-proxy
	rm -f docker/app-sidecar/app-sidecar
	rm -f docker/amb-sidecar/amb-sidecar
	rm -f docker/consul_connect_integration/consul_connect_integration
	rm -f k8s-*/??-ambassador-certs.yaml k8s-*/*.pem
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-02-07
	rm -rf tests/oauth-e2e/node_modules
	rmdir tests/oauth-e2e || true
# 2019-01-23
	rm -f docker/traffic-proxy/proxy
# 2019-01-23
	rm -f docker/app-sidecar/sidecar
# 2019-01-23 386e530eca29f38a0bbf4dd1b4ccf97f4e577230
	rm -f docker/amb-sidecar/oauth
	rm -f docker/amb-sidecar/apictl
	rm -f docker/amb-sidecar/ratelimit
# 2019-01-23 5962fe6f1fd0ed7969b63a0a90e062c2f648a6ed
	rm -f docker/amb-sidecar/ambassador-oauth
# 2019-01-22 978512decab17696b82ad962a04de6770e7f1458
	rm -f docker/amb-sidecar-ratelimit/apictl
	rm -f docker/amb-sidecar-ratelimit/ratelimit
	rm -f docker/amb-sidecar-ratelimit/ratelimit_check
	rm -f docker/amb-sidecar-ratelimit/ratelimit_client
	rm -f docker/amb-sidecar-oauth/ambassador-oauth
# 2019-01-18 0abb1c9e4bdc8ce04034c16d796bf3b67cce68f5
	rm -f tests/oauth-e2e/k8s/??-ambassador-certs.yaml tests/oauth-e2e/k8s/*.pem
# 2019-01-18 f9210ead4d2e67c51ebdcde48372658a862d3612
	rm -f e2e-oauth/k8s/??-ambassador-certs.yaml e2e-oauth/k8s/*.pem
	rm -rf e2e-oauth/node_modules
# 2019-01-18 d33436c1bfeaa187f649a21917443c5eb9ec3617
	rm -f docker/traffic-sidecar/sidecar
	rm -f docker/ambassador-ratelimit/apictl
	rm -f docker/ambassador-ratelimit/ratelimit
	rm -f docker/ambassador-ratelimit/ratelimit_check
	rm -f docker/ambassador-ratelimit/ratelimit_client
	rm -f docker/ambassador-oauth/ambassador-oauth
	rm -f docker/traffic-sidecar/ambex
clobber:
	rm -f docker/app-sidecar/ambex
	rm -rf tests/cluster/oauth-e2e/node_modules

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
