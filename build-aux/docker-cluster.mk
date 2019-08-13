# Copyright 2019 Datawire. All rights reserved.
#
# Makefile snippet for pushing Docker images to kubernaut.io clusters.
#
## Eager inputs ##
#  - Variable: KUBECONFIG (optional)
## Lazy inputs ##
#  - Variable: DOCKER_K8S_ENABLE_PVC ?=
## Outputs ##
#  - Target        : %.docker.knaut-push     # pushes to private in-kubernaut-cluster registry
## common.mk targets ##
#  - clean

# ## Local docker build ##
#
#    See docker.mk
#
# ## Pushing to a private Kubernaut registry ##
#
#     > NOTE: On macOS, you will need to add
#     > host.docker.internal:31000` to Docker's list of "Insecure
#     > registries" in order to push to kubernaut.io clusters.  Ask
#     > Abhay how to do that.
#
#    You can push to kubernaut by depending on
#    `somedir.docker.knaut-push`.  It will be known in-cluster as
#    `$$(cat somedir.docker.knaut-push)`.  You will need to substitute
#    that value in your YAML (kubeapply can help with this).
#
#    The private in-kubernaut-cluster registry is known as
#    "localhost:31000" to the cluster Nodes.
#
#    As a preliminary measure for supporting this functionality on
#    non-Kubernaut clusters, if DOCKER_K8S_ENABLE_PVC is 'true', then
#    the in-cluster registry will use a PersistentVolumeClaim (instead
#    of a hostPath) for storage.  Kubernaut does not support
#    PersistentVolumeClaims, but since Kubernaut clusters only have a
#    single Node, a hostPath is an acceptable hack there; it isn't on
#    other clusters.
#
# ## Pushing to a public registry ##
#
#    See docker.mk
#
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_docker-cluster.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_docker-cluster.mk))docker.mk
include $(dir $(_docker-cluster.mk))kubeapply.mk

DOCKER_K8S_ENABLE_PVC ?=

_docker.port-forward = $(dir $(_docker-cluster.mk))docker-port-forward

# %.docker.knaut-push file contents:
#
#  line 1: in-cluster tag name (hash-based)
%.docker.knaut-push: %.docker $(KUBEAPPLY) $(FLOCK) $(KUBECONFIG)
# the FLOCK for KUBEAPPLY is to work around https://github.com/datawire/teleproxy/issues/77
	DOCKER_K8S_ENABLE_PVC=$(DOCKER_K8S_ENABLE_PVC) $(FLOCK) $(_docker.port-forward).lock $(KUBEAPPLY) -f $(dir $(_docker-cluster.mk))docker-registry.yaml
	{ \
	    trap "kill $$($(FLOCK) $(_docker.port-forward).lock sh -c 'kubectl port-forward --namespace=docker-registry $(if $(filter true,$(DOCKER_K8S_ENABLE_PVC)),statefulset,deployment)/registry 31000:5000 >$(_docker.port-forward).log 2>&1 & echo $$!')" EXIT; \
	    while ! curl -i http://localhost:31000/ 2>/dev/null; do sleep 1; done; \
	    docker push "$$(sed -n 3p $<)"; \
	}
	sed -n '3{ s/^[^:]*:/127.0.0.1:/; p; }' $< > $@

# This `go run` bit is gross, compared to just depending on and using
# $(FLOCK).  But if the user runs `make clobber`, the prelude.mk
# cleanup might delete $(FLOCK) before we get to run it.
_clean-docker-cluster:
	cd $(dir $(_docker-cluster.mk))bin-go/flock && GO111MODULE=on go run . $(abspath $(_docker.port-forward).lock) rm $(abspath $(_docker.port-forward).lock)
	rm -f $(_docker.port-forward).log
	rm -f $(dir $(_docker-cluster.mk))docker-registry.yaml.o
clean: _clean-docker-cluster
.PHONY: _clean-docker-cluster

endif
