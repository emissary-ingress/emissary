OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# We'll set REGISTRY_ERR in builder.mk
docker.tag.dev = $(if $(DEV_REGISTRY),$(DEV_REGISTRY)/$*:$(patsubst sha256:%,%,$(shell cat $<)),$(REGISTRY_ERR))

include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux/docker.mk
include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/build-aux-local/version.mk

$(call module,ambassador,$(OSS_HOME))

compile: python/ambassador/VERSION.py

.SECONDARY:
