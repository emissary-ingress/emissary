# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet for calling `telepresence`
#
## Eager inputs ##
#  - Variable: KUBECONFIG
## Outputs ##
#  - .PHONY Target: proxy
#  - .PHONY Target: unproxy
#  - .PHONY Target: status-proxy
## common.mk targets ##
#  - clean
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_teleproxy.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_teleproxy.mk))prelude.mk

proxy: ## (Telepresence) Launch telepresence in the background
proxy: $(KUBECONFIG) $(tools/telepresence)
	$(tools/telepresence) connect
.PHONY: proxy

unproxy: ## (Telepresence) Shut down 'proxy'
	$(tools/telepresence) quit || true
.PHONY: unproxy

status-proxy: ## (Telepresence) Fail if cluster connectivity is broken or telepresence is not running
status-proxy: status-cluster
	$(tools/telepresence) status
.PHONY: status-proxy

$(KUBECONFIG).clean: unproxy

clean-proxy: unproxy

endif
