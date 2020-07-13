# Sanitize the environment a bit.
undefines += ENV      # bad configuration mechansim
undefines += BASH_ENV # bad configuration mechansim, but CircleCI insists on it
undefines += CDPATH   # should not be exported, but some people do
undefines += IFS      # should not be exported, but some people do
ifeq ($(filter undefine,$(.FEATURES)),)
  # Make 3.81 didn't have an 'undefine' directive
  $(foreach v,$(undefines),$(if $(filter $v,$(.VARIABLES)),$(eval $v =)))
else
  # Make 3.82 added undefine
  undefine $(undefines)
endif

NAME ?= ambassador

OSS_HOME := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/cxx/envoy.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux-local/generate.mk

# Configure GNU Make itself
SHELL = bash
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE
