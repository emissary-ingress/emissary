OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# We'll set REGISTRY_ERR in builder.mk
docker.tag.dev = $(if $(DEV_REGISTRY),$(DEV_REGISTRY)/$*:$(patsubst sha256:%,%,$(shell cat $<)),$(REGISTRY_ERR))

# All Docker images that we know how to build
images.all =
# The subset of $(images.all) that we will deploy to the
# DEV_KUBECONFIG cluster.
images.cluster =

images.all += $(patsubst docker/%/Dockerfile,%,$(wildcard docker/*/Dockerfile)) test-auth-tls
images.cluster += $(filter test-%,$(images.all))

include $(OSS_HOME)/build-aux/prelude.mk
include $(OSS_HOME)/build-aux/docker.mk
include $(OSS_HOME)/builder/builder.mk
include $(OSS_HOME)/build-aux-local/version.mk

$(call module,ambassador,$(OSS_HOME))

sync: python/ambassador/VERSION.py

test-%.docker.stamp: docker/test-%/Dockerfile FORCE
	docker build --quiet --iidfile=$@ $(<D)
test-auth-tls.docker.stamp: docker/test-auth/Dockerfile FORCE
	docker build --quiet --build-arg TLS=--tls --iidfile=$@ $(<D)

.SECONDARY:
