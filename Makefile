# Sanitize the environment a bit.
unexport ENV      # bad configuration mechanism
unexport BASH_ENV # bad configuration mechanism, but CircleCI insists on it
unexport CDPATH   # should not be exported, but some people do
unexport IFS      # should not be exported, but some people do

# In the days before Bash 2.05 (April 2001), Bash had a hack in it
# where it would load the interactive-shell configuration when run
# from sshd, I guess to work around buggy sshd implementations that
# didn't run the shell as login or interactive or something like that.
# But that hack was removed in Bash 2.05 in 2001.  And the changelog
# indicates that the heuristics it used to decide whether to do that
# were buggy to begin with, and it would often trigger when it
# shouldn't.  BUT DEBIAN PATCHES BASH TO ADD THAT HACK BACK IN!  And,
# more importantly, Ubuntu 20.04 (which our CircleCI uses) inherits
# that patch from Debian.  And the heuristic that Bash uses
# incorrectly triggers inside of Make in our CircleCI jobs!  So, unset
# SSH_CLIENT and SSH2_CLIENT to disable that.
unexport SSH_CLIENT
unexport SSH2_CLIENT

NAME ?= emissary

OSS_HOME := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

include $(OSS_HOME)/build-aux/tools.mk
include $(OSS_HOME)/build-aux/ci.mk
include $(OSS_HOME)/build-aux/check.mk
include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/_cxx/envoy.mk
include $(OSS_HOME)/charts/charts.mk
include $(OSS_HOME)/manifests/manifests.mk
include $(OSS_HOME)/releng/release.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux/generate.mk
include $(OSS_HOME)/build-aux/lint.mk

include $(OSS_HOME)/docs/yaml.mk

ifneq ($(MAKECMDGOALS),$(OSS_HOME)/build-aux/go-version.txt)
  $(_prelude.go.ensure)
  ifeq ($(shell go env GOPATH),$(shell go env GOROOT))
    $(error Your $$GOPATH (where *your* Go stuff goes) and $$GOROOT (where Go *itself* is installed) are both set to the same directory ($(shell go env GOROOT)); it is remarkable that it has not blown up catastrophically before now)
  endif
endif

test-chart-values.yaml: docker/$(LCNAME).docker.push.remote
	{ \
	  echo 'image:'; \
	  sed -E -n '2s/^(.*):.*/  repository: \1/p' < $<; \
	  sed -E -n '2s/.*:/  tag: /p' < $<; \
	} >$@

test-chart: $(tools/k3d) test-chart-values.yaml
	PATH=$(abspath $(tools.bindir)):$(PATH) $(MAKE) -C charts/emissary-ingress HELM_TEST_VALUES=$(abspath test-chart-values.yaml) $@
.PHONY: test-chart

lint-chart:
	$(MAKE) -C charts/emissary-ingress $@
.PHONY: lint-chart

# Configure GNU Make itself
SHELL = bash
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE

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

deploy-only: preflight-dev-kubeconfig $(tools/kubectl)
	mkdir -p $(OSS_HOME)/build/helm/ && \
	($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) create ns ambassador || true) && \
	helm template ambassador --include-crds --output-dir $(OSS_HOME)/build/helm -n ambassador charts/emissary-ingress/ \
		--set createNamespace=true \
		--set service.selector.service=ambassador \
		--set replicaCount=1 \
		--set enableAES=false \
		--set image.fullImageOverride=$$(sed -n 2p docker/$(LCNAME).docker.push.remote) && \
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) apply -f $(OSS_HOME)/build/helm/emissary-ingress/crds/ && \
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) apply -f $(OSS_HOME)/build/helm/emissary-ingress/templates && \
	rm -rf $(OSS_HOME)/build/helm
	$(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) -n ambassador wait --for condition=available --timeout=90s deploy --all
	@printf "$(GRN)Your ambassador service IP:$(END) $(BLD)$$($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')$(END)\n"
	@printf "$(GRN)Your ambassador image:$(END) $(BLD)$$($(tools/kubectl) --kubeconfig $(DEV_KUBECONFIG) get -n ambassador deploy ambassador -o 'go-template={{(index .spec.template.spec.containers 0).image}}')$(END)\n"
	@printf "$(GRN)Your built image:$(END) $(BLD)$$(sed -n 2p docker/$(LCNAME).docker.push.remote)$(END)\n"
.PHONY: deploy-only
