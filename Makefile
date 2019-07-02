NAME            = ambassador-pro
# For docker.mk
# If you change DOCKER_IMAGE, you'll also need to change the image
# names in `cmd/apictl/traffic.go`.
DOCKER_IMAGE    = quay.io/datawire/ambassador_pro:$(notdir $*)-$(VERSION)
# For Makefile
image.all       = $(sort $(patsubst %/Dockerfile,%,$(wildcard docker/*/Dockerfile)) docker/amb-sidecar-plugins)
image.norelease = docker/amb-sidecar-plugins docker/example-service docker/max-load $(filter docker/model-cluster-%,$(image.all))
image.nocluster = docker/apro-plugin-runner
# For k8s.mk
K8S_IMAGES      = $(filter-out $(image.nocluster),$(image.all))
K8S_DIRS        = k8s-sidecar k8s-standalone k8s-localdev
K8S_ENVS        = k8s-env.sh
# For go.mk
go.PLATFORMS    = linux_amd64 darwin_amd64
go.pkgs         = ./... github.com/lyft/ratelimit/...

export CGO_ENABLED = 0
export SCOUT_DISABLE = 1

# In order to work with Alpine's musl libc6-compat, things must be
# compiled for compatibility with LSB 3. Setting _FORTIFY_SOURCE=2
# with GNU libc causes the CGO 1.12 runtime to require LSB 4.
#
# Ubuntu 14.04 (which we use in CircleCI) patches their GCC to define
# _FORTIFY_SOURCE=2 by default.
export CGO_CPPFLAGS += -U_FORTIFY_SOURCE

include build-aux/kubernaut-ui.mk
# Include kubernaut-ui.mk before anything else, because it sets
# KUBECONFIG, which generally is eager.
include build-aux/go-mod.mk
include build-aux/go-version.mk
include build-aux/k8s.mk
include build-aux/teleproxy.mk
include build-aux/pidfile.mk
include build-aux/help.mk

.DEFAULT_GOAL = help

status: ## Report on the status of Kubernaut and Teleproxy
status: status-pro-tel
.PHONY: status

pull-docs: ## Update ./docs from https://github.com/datawire/ambassador-docs
	{ \
		git fetch https://github.com/datawire/ambassador-docs master && \
		docs_head=$$(git rev-parse FETCH_HEAD) && \
		git subtree merge --prefix=docs "$${docs_head}" && \
		git subtree split --prefix=docs --rejoin --onto="$${docs_head}"; \
	}
push-docs: ## Publish ./docs to https://github.com/datawire/ambassador-docs
	{ \
		git fetch https://github.com/datawire/ambassador-docs master && \
		docs_old=$$(git rev-parse FETCH_HEAD) && \
		docs_new=$$(git subtree split --prefix=docs --rejoin --onto="$${docs_old}") && \
		git push git@github.com:datawire/ambassador-docs.git "$${docs_new}:refs/heads/$(or $(PUSH_BRANCH),master)"; \
	}
.PHONY: pull-docs push-docs

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

lyft.bin.name = $(word 1,$(subst :, ,$(lyft.bin)))
lyft.bin.pkg  = $(word 2,$(subst :, ,$(lyft.bin)))
$(foreach lyft.bin,$(lyft.bins),$(eval $(call go.bin.rule,$(lyft.bin.name),$(lyft.bin.pkg))))
go-build: $(foreach _go.PLATFORM,$(go.PLATFORMS),$(foreach lyft.bin,$(lyft.bins), bin_$(_go.PLATFORM)/$(lyft.bin.name) ))

# https://github.com/golangci/golangci-lint/issues/587
go-lint: _go-lint-lyft
_go-lint-lyft: $(GOLANGCI_LINT) go-get $(go.lock)
	cd vendor-ratelimit && $(go.lock)$(GOLANGCI_LINT) run -c ../.golangci.yml ./...
.PHONY: _go-lint-lyft

#
# Plugins

apro-abi.txt: bin_linux_amd64/amb-sidecar
	$(if $(CI),@set -e; if test -e $@; then echo 'This should not happen in CI: $@ rebuild triggered by $+' >&2; false; fi)
	{ \
		echo '# _GOVERSION=$(go.goversion)'; \
		echo "# GOPATH=$$(go env GOPATH)"; \
		echo '# GOOS=linux'; \
		echo '# GOARCH=amd64'; \
		echo '# CGO_ENABLED=1'; \
		echo '# GO111MODULE=on'; \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go list -deps -f='{{if not .Standard}}{{.Module}}{{end}}' ./cmd/amb-sidecar | sort -u | grep -v -e '=>' -e '/apro$$'; \
	} > $@
build: apro-abi.txt

plugins = $(patsubst plugins/%/go.mod,%,$(wildcard plugins/*/go.mod))

# We use $(shell find ...) instead of FORCE here because not even the
# .cache trick will enable linker caching for -buildmode=plugin on
# macOS (verified with go 1.11.4 and 1.11.5).
define plugin.rule
bin_%/.cache.$(plugin.name).so: plugins/$(plugin.name)/go.mod $$(shell find plugins/$(plugin.name))
	cd $$(<D) && $$(go.GOBUILD) -buildmode=plugin -o $(abspath $$@) .
bin_%/$(plugin.name).so: bin_%/.cache.$(plugin.name).so
	@{ \
		PS4=''; set -x; \
		if ! cmp -s $$< $$@; then \
			$(if $(CI),if test -e $$@; then false This should not happen in CI: $$@ should not change; fi &&) \
			cp -f $$< $$@; \
		fi; \
	}
endef
$(foreach plugin.name,$(plugins),$(eval $(plugin.rule)))

# This is gross.  There are several use-cases this aims to keep happy:
#
#                          |   amb-sidecar: plugins?    |    compile test plugins?   |
#          host            | linux_amd64 | darwin_amd64 | linux_amd64 | darwin_amd64 |
# +------------------------+-------------+--------------+-------------+--------------|
# | linux                  |     yes(A,B)|     no       |     yes(A,B)|     no       |
# | darwin w/ Docker (dev) |     yes(A)  |     yes(B)   |     yes(A)  |     yes(B)   |
# | darwin w/o Docker (CI) |     no      |     yes      |     no      |     yes      |
#
# A: Needed for in-cluster
# B: Needed for Telepresence local-dev

# always do plugins on native-builds
go-build: $(foreach p,$(plugins),bin_$(GOHOSTOS)_$(GOHOSTARCH)/$p.so)
_cgo_files = amb-sidecar apro-plugin-runner $(addsuffix .so,$(plugins))
$(addprefix bin_$(GOHOSTOS)_$(GOHOSTARCH)/,$(_cgo_files)): CGO_ENABLED=1

# but cross-builds are the complex story
ifneq ($(GOHOSTOS)_$(GOHOSTARCH),linux_amd64)
ifneq ($(HAVE_DOCKER),)

go-build: $(foreach p,$(plugins),bin_linux_amd64/$p.so)

# For cross-compiled CGO binaries, we'll compile them in Docker.
$(addprefix bin_linux_amd64/,$(_cgo_files)): CGO_ENABLED = 1
$(addprefix bin_linux_amd64/,$(_cgo_files)): go.GOBUILD = $(_cgo_GOBUILD)
_cgo_GOBUILD  = docker run --rm
_cgo_GOBUILD += --env GOOS
_cgo_GOBUILD += --env GOARCH
_cgo_GOBUILD += --env GO111MODULE
_cgo_GOBUILD += --env CGO_ENABLED
# Map this directory in to the container.  Except for $@, it should be
# read-only, so it should be safe to speed things up with "delegated".
_cgo_GOBUILD += --volume $(CURDIR):$(CURDIR):rw,delegated
# We could map in $(shell go env GOPATH) and $(shell go env GOCACHE),
# but osxfs is slow enough that it's worth it to just maintain
# separate in-Docker caches.
_cgo_GOBUILD += --volume apro-gocache:/mnt/gocache:rw
_cgo_GOBUILD += --env GOPATH=/mnt/gocache/go-workspace
_cgo_GOBUILD += --env GOCACHE=/mnt/gocache/go-build
# We use $$PWD here instead of $(CURDIR) so that the shell (not Make)
# expands it, so that it behaves correctly if the command `cd`s to a
# subdirectory first.
_cgo_GOBUILD += --workdir $$PWD
# It doesn't really matter which version of docker.io/library/golang
# we choose, but matching the host's Go version seems more future-safe
# than hard-coding a version.
_cgo_GOBUILD += docker.io/library/golang:$(patsubst go%,%,$(filter go1%,$(shell go version)))
_cgo_GOBUILD += go build

endif
endif

#
# Docker images

build: $(if $(HAVE_DOCKER),$(addsuffix .docker,$(image.all)))

# This assumes that if there's a Go binary with the same name as the
# Docker image, then the image wants that binary.  That's a safe
# assumption so far, and forces us to name things in a consistent
# manner.
define docker.bins_rule
$(if $(filter $(notdir $(image)),$(notdir $(go.bins))),$(image).docker: $(image)/$(notdir $(image)) $(image)/$(notdir $(image)).opensource.tar.gz)
$(image)/%: bin_linux_amd64/%
	cp $$< $$@
$(image)/clean:
	rm -f $(image)/$(notdir $(image))
.PHONY: $(image)/clean
clean: $(image)/clean
endef
$(foreach image,$(image.all),$(eval $(docker.bins_rule)))

_gocache_volume_clobber:
	if docker volume ls | grep -q apro-gocache; then docker volume rm apro-gocache; fi
.PHONY: _gocache_volume_clobber
clobber: _gocache_volume_clobber

docker/app-sidecar.docker: docker/app-sidecar/ambex
docker/app-sidecar/ambex:
	curl -o $@ --fail 'https://s3.amazonaws.com/datawire-static-files/ambex/0.1.0/ambex'
	chmod 755 $@

docker/amb-sidecar-plugins/Dockerfile: docker/amb-sidecar-plugins/Dockerfile.gen docker/amb-sidecar.docker
	$^ > $@
docker/amb-sidecar-plugins.docker: docker/amb-sidecar.docker # ".SECONDARY:" (in common.mk) coming back to bite us
docker/amb-sidecar-plugins.docker: $(foreach p,$(plugins),docker/amb-sidecar-plugins/$p.so)

docker/consul_connect_integration.docker: docker/consul_connect_integration/kubectl

docker/max-load.docker: docker/max-load/03-ambassador.yaml
docker/max-load.docker: docker/max-load/kubeapply
docker/max-load.docker: docker/max-load/kubectl
docker/max-load.docker: docker/max-load/test.sh
docker/max-load/kubeapply:
	curl -o $@ --fail https://s3.amazonaws.com/datawire-static-files/kubeapply/$(KUBEAPPLY_VERSION)/linux/amd64/kubeapply
	chmod 755 $@

docker/%/kubectl:
	curl -o $@ --fail 'https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl'
	chmod 755 $@

#
# Deploy

# Generate the TLS secret
%/cert.pem %/key.pem: %/namespace.txt
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.$$(cat $<).svc.cluster.local"
%/04-ambassador-certs.yaml: %/cert.pem %/key.pem %/namespace.txt
	kubectl --namespace="$$(cat $*/namespace.txt)" create secret tls --dry-run --output=yaml ambassador-certs --cert $*/cert.pem --key $*/key.pem > $@

%/03-auth0-secret.yaml: %/namespace.txt $(K8S_ENVS)
	$(if $(K8S_ENVS),set -a && $(foreach k8s_env,$(abspath $(K8S_ENVS)), . $(k8s_env) && ))kubectl --namespace="$$(cat $*/namespace.txt)" create secret generic --dry-run --output=yaml auth0-secret --from-literal=oauth2-client-secret="$$IDP_AUTH0_CLIENT_SECRET" > $@

deploy: $(addsuffix /04-ambassador-certs.yaml,$(K8S_DIRS)) k8s-standalone/03-auth0-secret.yaml
apply: $(addsuffix /04-ambassador-certs.yaml,$(K8S_DIRS)) k8s-standalone/03-auth0-secret.yaml

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
			--expose 8500 --expose 38888 \
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
$(KUBECONFIG).clean: kill-pro-tel
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
	@echo '  env $$(cat pro-env.sh)' "bin_$(GOHOSTOS)_$(GOHOSTARCH)/amb-sidecar auth"
.PHONY: help-local-dev
run-auth: ## (LocalDev) Build and launch the auth service locally
run-auth: bin_$(GOHOSTOS)_$(GOHOSTARCH)/amb-sidecar
	env $$(cat pro-env.sh) APP_LOG_LEVEL=debug bin_$(GOHOSTOS)_$(GOHOSTARCH)/amb-sidecar main
.PHONY: run-auth

#
# Check

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

tests/cluster/external.tap: $(GOTEST2TAP)

tests/cluster/oauth-e2e/node_modules: tests/cluster/oauth-e2e/package.json $(wildcard tests/cluster/oauth-e2e/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@
check tests/cluster/oauth-e2e.tap: tests/cluster/oauth-e2e/node_modules

#
# Load-testing

infra/loadtest-cluster/.terraform: FORCE
	cd infra/loadtest-cluster && terraform init
infra/loadtest-cluster/loadtest.kubeconfig: infra/loadtest-cluster/.terraform FORCE
	cd infra/loadtest-cluster && terraform plan -out create.tfplan && terraform apply create.tfplan
infra/loadtest-cluster/loadtest.kubeconfig.clean: %.clean:
	if [ -e $* ]; then cd infra/loadtest-cluster && terraform plan -destroy -out destroy.tfplan && terraform apply destroy.tfplan; fi
	rm -f $*

loadtest-destroy: ## Destroy the load-testing cluster
loadtest-destroy: infra/loadtest-cluster/loadtest.kubeconfig.clean
loadtest-clean: ## Remove loadtest files
loadtest-clean: loadtest-destroy
	rm -rf infra/loadtest-cluster/.terraform
	rm -f infra/loadtest-cluster/*tfplan

loadtest-apply: ## Apply YAML to the load-testing cluster
loadtest-deploy: ## Push images and apply YAML to the load-testing cluster
loadtest-shell: ## Run a shell with loadtest variables set
loadtest-proxy: ## Launch teleproxy to the loadtest cluster
loadtest-apply loadtest-deploy loadtest-shell loadtest-proxy: loadtest-%: infra/loadtest-cluster/loadtest.kubeconfig
	$(MAKE) DOCKER_K8S_ENABLE_PVC=true KUBECONFIG=$$PWD/infra/loadtest-cluster/loadtest.kubeconfig K8S_DIRS=k8s-load $*

.PHONY: loadtest-%

#
# Clean

clean: $(addsuffix .clean,$(wildcard docker/*.docker)) loadtest-clean
	rm -f apro-abi.txt
	rm -f tests/*.log tests/*.tap tests/*/*.log tests/*/*.tap
	rm -f docker/amb-sidecar-plugins/Dockerfile docker/amb-sidecar-plugins/*.so
	rm -f docker/*/*.opensource.tar.gz
	rm -f k8s-*/??-ambassador-certs.yaml k8s-*/*.pem
	rm -f k8s-*/??-auth0-secret.yaml
	rm -f docker/*.knaut-push
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
	rm -f docker/*/ambex
	rm -f docker/*/kubeapply
	rm -f docker/*/kubectl
	rm -rf tests/cluster/oauth-e2e/node_modules

#
# Release

RELEASE_DRYRUN ?=
release.bins = apictl apictl-key apro-plugin-runner playpen
release.images = $(filter-out $(image.norelease),$(image.all))

release: ## Cut a release; upload binaries to S3 and Docker images to Quay
release: build
release: $(foreach platform,$(go.PLATFORMS),$(foreach bin,$(release.bins),release/bin_$(platform)/$(bin)))
release: release/apro-abi.txt
release: $(addsuffix .docker.push$(if $(RELEASE_DRYRUN),.dryrun),$(release.images))
.PHONY: release

%.docker.push.dryrun: %.docker
	@echo 'DRYRUN docker push (( $< ))'
.PHONY: %.docker.push.dryrun

_release_os   = $(word 2,$(subst _, ,$(@D)))
_release_arch = $(word 3,$(subst _, ,$(@D)))
release/%: % %.opensource.tar.gz
	$(if $(RELEASE_DRYRUN),@echo DRYRUN )aws s3 cp --acl public-read $<                   's3://datawire-static-files/$(@F)/$(VERSION)/$(_release_os)/$(_release_arch)/$(@F)'
	$(if $(RELEASE_DRYRUN),@echo DRYRUN )aws s3 cp --acl public-read $<.opensource.tar.gz 's3://datawire-static-files/$(@F)/$(VERSION)/$(_release_os)/$(_release_arch)/$(@F).opensource.tar.gz'
release/apro-abi.txt: release/%: %
	$(if $(RELEASE_DRYRUN),@echo DRYRUN )aws s3 cp --acl public-read $< 's3://datawire-static-files/apro-abi/apro-abi@$(VERSION).txt'
.PHONY: release/%
