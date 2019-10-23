OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

include $(OSS_HOME)/builder/builder.mk

$(call module,ambassador,$(OSS_HOME))
