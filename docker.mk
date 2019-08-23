# Copyright 2019 Datawire. All rights reserved.
#
# Makefile snippet for building, tagging, and pushing Docker images.
#
## Eager inputs ##
#  (none)
## Lazy inputs ##
#  (none)
## Outputs ##
#
#  - Executable    : WRITE_DOCKERTAGFILE ?= $(CURDIR)/build-aux/bin/write-dockertagfile
#
#  - Variable      : HAVE_DOCKER             # non-empty if true, empty if false
#  - Variable      : docker.LOCALHOST        # "host.docker.internal" on Docker for Desktop, "localhost" on Docker CE
#
#  - Target        : %.docker: %/Dockerfile               # builds image (untagged)
#  - .PHONY Target : %.docker.clean                       # remove image and tags
#  - Function: $(eval $(call docker.tag.rule,GROUP,EXPR)) # adds targets:
#                # : %.docker.tag.GROUP                   # tags image as EXPR
#                # : %.docker.push.GROUP                  # pushes tag EXPR
#
## common.mk targets ##
#  (none)
#
# ## Local docker build ##
#
#    To use this Makefile snippet naively, `Dockerfile`s must be in
#    sub-directories; it doesn't support out-of-the-box having a
#    `Dockerfile` in the root.  If you would like to have
#    `Dockerfile`s out of the root (or with other names, like
#    `Dockerfile.base-envoy`), then you must supply your own
#    `%.docker` target that
#      1. Calls `docker build --iidfile=TEMPFILE ...`
#      2. Calls `$(MOVE_IFCHANGED) TEMPFILE $@`
#
#    You can build a Docker image by depending on `SOMEPATH.docker`.
#    If you provide a custom `%.docker` rule, then of course exactly
#    what that builds will be different, but with the default built-in
#    rule: depending on `SOMEPATH.docker` will build
#    `SOMEPATH/Dockefile`.  This will build the image, but NOT tag it
#    (see below for tagging).
#
#    You can untag and remove an image by having your `clean` target
#    depend on `SOMEPATH.docker.clean`.
#
#    With the default built-in rule:
#
#     - If you need something to be done before the `docker build`,
#       make it a dependency of `SOMEPATH.docker`.
#
#     - If you need something (`FILE`) to be included in the build
#       context, copy it to `SOMEPATH/` by having
#       `SOMEPATH.docker` depend on `SOMEPATH/FILE`.
#
# ## Working with those untagged images ##
#
#     - Tagging: You can tag an image after being built by depending
#       on `SOMEPATH.docker.tag.GROUP`, where you've set up GROUP by
#       writing
#
#           $(eval $(call docker.tag.rule,GROUP,EXPR))
#
#       where GROUP is the suffix of the target that you'd like to
#       depend on in your Makefile, and EXPR is a Makefile expression
#       that evaluates to 1 or more tag names; it is evaluated in the
#       context of `SOMEPATH.docker.tag.GROUP`; specifically:
#        * `$*` is set to SOMEPATH
#        * `$<` is set to a file containing the image ID
#
#       Additionally, you can override the EXPR on a per-image basis
#       by overriding the `docker.tag.name.GROUP` variable on a
#       per-target basis:
#
#           SOMEPATH.docker.tag.GROUP: docker.tag.name.GROUP = EXPR
#
#     - Pushing a tag: You can push tags that have been created with
#       `SOMEPATH.docker.tag.GROUP` (see above) by depending on
#       `SOMEPATH.docker.push.GROUP`.
#
#          > For example:
#          >   The Ambassador Pro images:
#          >    - get built from: `docker/$(NAME)/Dockerfile`
#          >    - get pushed as : `quay.io/datawire/ambassador_pro:$(NAME)-$(VERSION)`
#          >
#          >   We accomplish this by saying:
#          >
#          >      $(eval $(call docker.tag.rule,quay,quay.io/datawire/ambassador_pro:$(notdir $*)-$(VERSION)))
#          >
#          >   and having our `build`   target depend on `NAME.docker.tag.quay` (for each NAME)
#          >   and having our `release` target depend on `NAME.docker.push.quay` (for each NAME)
#
#     - Clean up: You can untag (if there are any tags) and remove an
#       image by having your `clean` target depend on
#       `SOMEPATH.docker.clean`.  Because docker.mk does not have a
#       listing of all the images you may ask it to build, these are
#       NOT automatically added to the common.mk 'clean' target, and
#       you MUST do that yourself.
#
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_docker.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_docker.mk))prelude.mk

_docker.tag-groups =

#
# Executables

WRITE_DOCKERTAGFILE ?= $(build-aux.bindir)/write-dockertagfile

#
# Variables

HAVE_DOCKER      = $(call lazyonce,HAVE_DOCKER,$(shell which docker 2>/dev/null))
docker.LOCALHOST = $(if $(filter darwin,$(GOHOSTOS)),host.docker.internal,localhost)

#
# Targets

# file contents:
#   line 1: image ID
%.docker: %/Dockerfile $(MOVE_IFCHANGED) FORCE
# Try with --pull, fall back to without --pull
	docker build --iidfile=$(@D)/.tmp.$(@F).tmp --pull $* || docker build --iidfile=$(@D)/.tmp.$(@F).tmp $*
	$(MOVE_IFCHANGED) $(@D)/.tmp.$(@F).tmp $@

%.docker.clean:
	$(if $(_docker.tag-groups),$(MAKE) $(addprefix $@.,$(_docker.tag-groups)))
	if [ -e $*.docker ]; then docker image rm "$$(cat $*.docker)" || true; fi
	rm -f $*.docker
.PHONY: %.docker.clean

#
# Functions

# Usage: $(eval $(call docker.tag.rule,TAG_GROUPNAME,TAG_EXPRESSION))
#
# Add a set of %.docker.tag.TAG_GROUPNAME and
# %.docker.push.TAG_GROUPNAME targets that tag and push the docker image.
#
# TAG_EXPRESSION is evaluated in the context of
# %.docker.tag.TAG_GROUPNAME.
define docker.tag.rule
  # The 'foreach' is to handle newlines as normal whitespace
  docker.tag.name.$(strip $1) = $(foreach v,$(value $2),$v)
  _docker.tag-groups += $(strip $1)

  # file contents:
  #   line 1: image ID
  #   line 2: tag 1
  #   line 3: tag 2
  #   ...
  %.docker.tag.$(strip $1): %.docker $$(WRITE_DOCKERTAGFILE) FORCE
  # The 'foreach' is to handle newlines as normal whitespace
	printf '%s\n' $$$$(cat $$<) $$(foreach v,$$(docker.tag.name.$(strip $1)),$$v) | $$(WRITE_DOCKERTAGFILE) $$@

  # file contents:
  #   line 1: image ID
  #   line 2: tag 1
  #   line 3: tag 2
  #   ...
  %.docker.push.$(strip $1): %.docker.tag.$(strip $1)
	sed 1d $$< | xargs -n1 docker push
	cat $$< > $$@

  %.docker.clean.$(strip $1):
	if [ -e $$*.docker.tag.$(strip $1) ]; then docker image rm $$$$(cat $$*.docker.tag.$(strip $1)) || true; fi
	rm -f $$*.docker.tag.$(strip $1) $$*.docker.push.$(strip $1)
  .PHONY: %.docker.clean.$(strip $1)
endef

endif
