OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux-local/version.mk

$(call module,ambassador,$(OSS_HOME))

compile: python/ambassador/VERSION.py
