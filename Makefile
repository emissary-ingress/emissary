# Real early setup
OSS_HOME := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))# Do this *before* 'include'ing anything else
include build-aux/init-sanitize-env.mk
include build-aux/init-configure-make-itself.mk
include build-aux/prelude.mk # In Haskell, "Prelude" is what they call the stdlib builtins that get get imported by default before anything else
include build-aux/tools.mk

# Bootstrapping the build env
ifneq ($(MAKECMDGOALS),$(OSS_HOME)/build-aux/go-version.txt)
  $(_prelude.go.ensure)
  ifeq ($(shell go env GOPATH),$(shell go env GOROOT))
    $(error Your $$GOPATH (where *your* Go stuff goes) and $$GOROOT (where Go *itself* is installed) are both set to the same directory ($(shell go env GOROOT)); it is remarkable that it has not blown up catastrophically before now)
  endif

  VERSION := $(or $(VERSION),$(shell go run ./tools/src/goversion))
  $(if $(filter v2.%,$(VERSION)),\
    ,$(error VERSION variable is invalid: It must be a v2.* string, but is '$(VERSION)'))
  $(if $(findstring +,$(VERSION)),\
    $(error VERSION variable is invalid: It must not contain + characters, but is '$(VERSION)'),)
  export VERSION

  CHART_VERSION := $(or $(CHART_VERSION),$(shell go run ./tools/src/goversion --dir-prefix=chart))
  $(if $(filter v7.%,$(CHART_VERSION)),\
    ,$(error CHART_VERSION variable is invalid: It must be a v7.* string, but is '$(CHART_VERSION)'))
  export CHART_VERSION

  include build-aux/version-hack.mk

  $(info [make] VERSION=$(VERSION))
  $(info [make] CHART_VERSION=$(CHART_VERSION))
endif

ifeq ($(SOURCE_DATE_EPOCH)$(shell git status --porcelain),)
  SOURCE_DATE_EPOCH := $(shell git log -1 --pretty=%ct)
endif
ifneq ($(SOURCE_DATE_EPOCH),)
  export SOURCE_DATE_EPOCH
  $(info [make] SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH))
endif

# Everything else...

NAME ?= emissary
_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

include $(OSS_HOME)/build-aux/ci.mk
include $(OSS_HOME)/build-aux/check.mk
include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/build-aux/main.mk
include $(OSS_HOME)/_cxx/envoy.mk
include $(OSS_HOME)/charts/charts.mk
include $(OSS_HOME)/manifests/manifests.mk
include $(OSS_HOME)/releng/release.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux/generate.mk
include $(OSS_HOME)/build-aux/lint.mk

include $(OSS_HOME)/docs/yaml.mk

test-chart-values.yaml: docker/$(LCNAME).docker.push.remote
	{ \
	  echo 'image:'; \
	  sed -E -n '2s/^(.*):.*/  repository: \1/p' < $<; \
	  sed -E -n '2s/.*:/  tag: /p' < $<; \
	} >$@

test-chart: $(tools/k3d) $(tools/kubectl) test-chart-values.yaml
	PATH=$(abspath $(tools.bindir)):$(PATH) $(MAKE) -C charts/emissary-ingress HELM_TEST_VALUES=$(abspath test-chart-values.yaml) $@
.PHONY: test-chart

lint-chart:
	$(MAKE) -C charts/emissary-ingress $@
.PHONY: lint-chart

.git/hooks/prepare-commit-msg:
	ln -s $(OSS_HOME)/tools/hooks/prepare-commit-msg $(OSS_HOME)/.git/hooks/prepare-commit-msg

githooks: .git/hooks/prepare-commit-msg

preflight-dev-kubeconfig:
	@if [ -z "$(DEV_KUBECONFIG)" ] ; then \
		echo "DEV_KUBECONFIG must be set"; \
		exit 1; \
	fi
.PHONY: preflight-dev-kubeconfig

deploy: push preflight-cluster
	$(MAKE) deploy-only
.PHONY: deploy

deploy-only: preflight-dev-kubeconfig $(tools/kubectl) $(OSS_HOME)/manifests/emissary/emissary-crds.yaml
	mkdir -p $(OSS_HOME)/build/helm/ && \
	($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) create ns ambassador || true) && \
	helm template ambassador --output-dir $(OSS_HOME)/build/helm -n ambassador charts/emissary-ingress/ \
		--set createNamespace=true \
		--set service.selector.service=ambassador \
		--set replicaCount=1 \
		--set enableAES=false \
		--set image.fullImageOverride=$$(sed -n 2p docker/$(LCNAME).docker.push.remote) && \
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) apply -f $(OSS_HOME)/manifests/emissary/emissary-crds.yaml && \
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) -n emissary-system wait --for condition=available --timeout=90s deploy emissary-apiext && \
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) apply -f $(OSS_HOME)/build/helm/emissary-ingress/templates && \
	rm -rf $(OSS_HOME)/build/helm
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) -n ambassador wait --for condition=available --timeout=90s deploy --all
	@printf "$(GRN)Your ambassador service IP:$(END) $(BLD)$$($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')$(END)\n"
	@printf "$(GRN)Your ambassador image:$(END) $(BLD)$$($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) get -n ambassador deploy ambassador -o 'go-template={{(index .spec.template.spec.containers 0).image}}')$(END)\n"
	@printf "$(GRN)Your built image:$(END) $(BLD)$$(sed -n 2p docker/$(LCNAME).docker.push.remote)$(END)\n"
.PHONY: deploy-only
