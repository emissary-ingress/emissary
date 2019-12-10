# Copyright 2019 Datawire. All rights reserved.
#
# Makefile snippet for installing `go-bindata`
#
## Eager inputs ##
#  (none)
## Lazy inputs ##
#  (none)
## Outputs ##
#  - Executable: GO_BINDATA ?= $(CURDIR)/build-aux/bin/go-bindata
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_go-bindata.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_go-bindata.mk))prelude.mk

GO_BINDATA ?= $(build-aux.bindir)/go-bindata
$(eval $(call build-aux.bin-go.rule, go-bindata, github.com/shuLhan/go-bindata/cmd/go-bindata))

endif
