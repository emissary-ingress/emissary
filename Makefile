NAME ?= ambassador

OSS_HOME:=$(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

include $(OSS_HOME)/builder/builder.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux-local/generate.mk

# Configure GNU Make itself
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE
