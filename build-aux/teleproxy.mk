# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet for calling `teleproxy`
#
## Eager inputs ##
#  - Variable: KUBECONFIG
#  - Variable: TELEPROXY_LOG ?= ./build-aux/teleproxy.log
## Lazy inputs ##
#  - Variable: KUBE_URL
## Outputs ##
#  - Executable: TELEPROXY ?= $(CURDIR)/build-aux/bin/teleproxy
#  - Variable: TELEPROXY_LOG ?= ./build-aux/teleproxy.log
#  - .PHONY Target: proxy
#  - .PHONY Target: unproxy
#  - .PHONY Target: status-proxy
## common.mk targets ##
#  - clean
## kubernaut-ui.mk targets ##
#  - $(KUBECONFIG).clean
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_teleproxy.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_teleproxy.mk))prelude.mk

OSS_HOME ?= $(build-aux.dir)/..

TELEPROXY = $(tools/teleproxy)
TELEPROXY_LOG ?= $(dir $(_teleproxy.mk))teleproxy.log
KUBE_URL = https://kubernetes/api/

tools/teleproxy = $(build-aux.bindir)/teleproxy
ifeq ($(GOHOSTOS),darwin)
$(tools/teleproxy).no-suid: CGO_ENABLED = 1
endif
$(tools/teleproxy).no-suid: FORCE
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $(abspath $@) github.com/datawire/ambassador/cmd/teleproxy
$(tools/teleproxy): $(tools/teleproxy).no-suid
	@PS4=; set -ex; { \
		if ! cmp -s $< $@; then \
			if [ -n "$${CI}" -a -e $@ ]; then \
				echo 'error: This should not happen in CI: $@ should not change' >&2; \
				exit 1; \
			fi; \
			sudo cp -f $< $@; \
			sudo chown 0:0 $@; \
			sudo chmod go-w,a+sx $@; \
		fi \
	}

proxy: ## (Kubernaut) Launch teleproxy in the background
proxy: $(KUBECONFIG) $(TELEPROXY)
	@if ! curl -sk $(KUBE_URL); then \
		echo "Starting proxy"; \
		set -x; \
		kubectl delete pods/teleproxy || true; \
		$(TELEPROXY) > $(TELEPROXY_LOG) 2>&1 & \
	else \
		echo "Proxy appears to already be running"; \
	fi
	@for i in $$(seq 127); do \
		echo "Checking proxy ($$i): $(KUBE_URL)"; \
		if curl -sk $(KUBE_URL); then \
			exit 0; \
		fi; \
		sleep 1; \
	done; echo "ERROR: proxy did not come up"; exit 1
	@printf '\n\nProxy UP!\n'
.PHONY: proxy

unproxy: ## (Kubernaut) Shut down 'proxy'
	curl -s --connect-timeout 5 127.254.254.254/api/shutdown || true
	@sleep 1
.PHONY: unproxy

status-proxy: ## (Kubernaut) Fail if cluster connectivity is broken or Teleproxy is not running
status-proxy: status-cluster
	@if curl -o /dev/null -s --connect-timeout 1 127.254.254.254; then \
		if curl -o /dev/null -sk $(KUBE_URL); then \
			echo "Proxy okay!"; \
		else \
			echo "Proxy up but connectivity check failed."; \
			exit 1; \
		fi; \
	else \
		echo "Proxy not running."; \
		exit 1; \
	fi
.PHONY: status-proxy

$(KUBECONFIG).clean: unproxy

clean: _clean-teleproxy
_clean-teleproxy: $(if $(wildcard $(TELEPROXY_LOG)),unproxy)
	rm -f $(TELEPROXY_LOG)
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2018-07-01
	rm -f $(dir $(_teleproxy.mk))teleproxy
.PHONY: _clean-teleproxy

endif
