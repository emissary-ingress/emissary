# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet for calling `teleproxy`
#
## Eager inputs ##
#  - Variable: KUBECONFIG
## Outputs ##
#  - Executable: $(tools/telepresence) = $(CURDIR)/build-aux/bin/telepresence
#  - .PHONY Target: proxy
#  - .PHONY Target: unproxy
#  - .PHONY Target: status-proxy
## common.mk targets ##
#  - clean
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_teleproxy.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_teleproxy.mk))prelude.mk

OSS_HOME ?= $(build-aux.dir)/..

tools/telepresence = $(build-aux.bindir)/telepresence
ifeq ($(GOHOSTOS),darwin)
$(tools/telepresence): CGO_ENABLED = 1
endif
$(tools/telepresence): FORCE
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $(abspath $@) github.com/telepresenceio/telepresence/v2/cmd/telepresence

proxy: ## (Telepresence) Launch telepresence in the background
proxy: $(KUBECONFIG) $(tools/telepresence)
	$(tools/telepresence) connect
.PHONY: proxy

unproxy: ## (Telepresence) Shut down 'proxy'
	$(tools/telepresence) quit || true
.PHONY: unproxy

status-proxy: ## (Telepresence) Fail if cluster connectivity is broken or Teleproxy is not running
status-proxy: status-cluster
	$(tools/telepresence) status
.PHONY: status-proxy

$(KUBECONFIG).clean: unproxy

clean: unproxy

endif
