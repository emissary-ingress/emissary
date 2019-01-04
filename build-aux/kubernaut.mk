<<<<<<< HEAD
## Quickstart (and pretty much everything else you need to know)
=======
# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet for managing kubernaut.io clusters.
#
## Inputs ##
#   - Variable: GUBERNAUT ?= gubernaut
## Outputs ##
#   - Target       : `%.knaut`
#   - .PHONY Target: `%.knaut.clean`
## common.mk targets ##
#   - clobber
#
# Creating the NAME.knaut creates the Kubernaut claim.  The file may
# be used as a KUBECONFIG file.
#
# Calling the NAME.knaut.clean file releases the claim, and removes
# the NAME.knaut file.
#
# The GUBERNAUT variable may be used to adjust the gubernaut command
# called; by default it looks up 'gubernaut' in $PATH.
#
## Quickstart ##
>>>>>>> origin
#
#  1. Put this file in your source tree and include it from your
#     Makefile, e.g.:
#
#     ...
#     include kubernaut.mk
#     ...
#
#  2. Run `make foo.knaut` to (re)acquire a cluster.
#
#  3. Use `kubectl -kubeconfig foo.knaut ...` to use a cluster.
#
#  4. Run `make foo.knaut.clean` to release the cluster.
#
#  5. If you care, the claim name is in foo.knaut.claim. This will use
#     a UUID if the CI environment variable is set.
#
#  6. Incorporate <blah>.knaut[.clean] targets into your Makefile as
#     needed
#
#     tests: test-cluster.knaut
#             KUBECONFIG=test-cluster.knaut py.test ...
#
#     clean: test-cluster.knaut.clean
#
<<<<<<< HEAD

%.knaut.claim :
	@if [ -z $${CI+x} ]; then \
		echo $(@:%.knaut.claim=%)-$${USER} > $@; \
	else \
		echo $(@:%.knaut.claim=%)-$${USER}-$(shell uuidgen) > $@; \
	fi
.PRECIOUS: %.knaut.claim
.SECONDARY: %.knaut.claim

KUBERNAUT=GO111MODULE=off go run build-aux/gubernaut.go
KUBERNAUT_CLAIM_NAME=$(shell cat $(@:%.knaut=%.knaut.claim))

%.knaut : %.knaut.claim
	$(KUBERNAUT) -release $(KUBERNAUT_CLAIM_NAME)
	$(KUBERNAUT) -claim $(KUBERNAUT_CLAIM_NAME) -output $@

%.knaut.clean :
	if [ -e $(@:%.clean=%.claim) ]; then $(KUBERNAUT) -release $$(cat $(@:%.clean=%.claim)); fi
	rm -f $(@:%.knaut.clean=%.knaut)
	rm -f $(@:%.clean=%.claim)
.PHONY: %.knaut.clean
=======
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
include $(dir $(lastword $(MAKEFILE_LIST)))/common.mk

GUBERNAUT = GO111MODULE=off go run build-aux/gubernaut.go

%.knaut.claim:
	echo $(subst /,_,$*)-$${USER}-$$(uuidgen) > $@
%.knaut: %.knaut.claim
	$(GUBERNAUT) -release $$(cat $<)
	$(GUBERNAUT) -claim $$(cat $<) -output $@

%.knaut.clean:
	if [ -e $*.claim ]; then $(GUBERNAUT) -release $$(cat $*.claim); fi
	rm -f $*.knaut $*.knaut.claim
.PHONY: %.knaut.clean

clobber: $(addsuffix .clean,$(wildcard *.knaut))

endif
>>>>>>> origin
