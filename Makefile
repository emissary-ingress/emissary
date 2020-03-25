NAME ?= ambassador

OSS_HOME:=$(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/cxx/envoy.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux-local/generate.mk

# Configure GNU Make itself
SHELL = bash
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE
