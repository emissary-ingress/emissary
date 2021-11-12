# Copyright 2019 Datawire. All rights reserved.
#
# Makefile snippet for building, tagging, and pushing Docker images.
#
## Eager inputs ##
#  - Variables     : docker.tag.$(GROUP)     # define %.docker.tag.$(GROUP) and %.docker.push.$(GROUP) targets
## Lazy inputs ##
#  - Target:       : $(NAME).docker          # build untagged image; define this for each image $(NAME)
## Outputs ##
#
#  - Variable      : HAVE_DOCKER             # non-empty if true, empty if false
#  - Variable      : docker.LOCALHOST        # "host.docker.internal" on Docker for Desktop, "localhost" on Docker CE
#
#  - Target        : %.docker.tag.$(GROUP)   # tag image as $(docker.tag.$(GROUP))
#  - Target        : %.docker.push.$(GROUP)  # push tag(s) $(docker.tag.$(GROUP))
#  - .PHONY Target : %.docker.clean          # remove image and tags
#
## common.mk targets ##
#  (none)
#
# This Makefile snippet helps you manage Docker images as files from a
# Makefile.  Think of it as glue.  It doesn't dictate how to build
# your images or what flags you pass to `docker build`--you must
# provide your own rule that calls `docker build`.  It doesn't dictate
# how your image tags are named.  It provides glue to keep track of
# those image, and flexibly but coherently push them to any of
# multiple remote Docker repositories.  All while being careful to not
# leave dangling images in your Docker cache that force you to run
# `docker image prune` an unreasonable amount.
#
# ## Building ##
#
#    For each Docker IMAGE you would like to build, you need to
#    provide your own build-rule for `IMAGE.docker`.  There are 2
#    requirements for the rule:
#
#     1. It must write a file named `IMAGE.docker` (for your value of
#        IMAGE) containing just the Image ID.  (This is trivially
#        accomplished using the --iidfile argument to `docker build`.)
#
#     2. It must only adjust the timestamp of IMAGE.docker if the
#        contents of the file change.  (This is trivially accomplished
#        using any of the `$(tools/*-ifchanged)` helper programs.)
#
#    The simplest version of that is:
#
#        IMAGE.docker: $(tools/move-ifchanged) FORCE
#        	docker build --iidfile=$@.tmp .
#        	$(tools/move-ifchanged) $@.tmp $@
#
#    If you have multiple `Dockerfile` at `IMAGE/Dockerfile`, you
#    might write a pattern rule:
#
#        %.docker: %/Dockerfile $(tools/move-ifchanged) FORCE
#        	docker build --iidfile=$@.tmp $*
#        	$(tools/move-ifchanged) $@.tmp $@
#
#    Unless you have a good reason to, you shouldn't concern yourself
#    with tagging the image in this rule.
#
#    See the "More build-rule examples" section below for more
#    examples.
#
# ## Tagging ##
#
#    You can tag an image after being built by depending on
#    `IMAGE.docker.tag.GROUP`, where you've set up GROUP by writing
#
#        docker.tag.GROUP = EXPR
#
#    _before_ including `docker.mk`, where GROUP is the suffix of the
#    target that you'd like to depend on in your Makefile, and EXPR is
#    a Makefile expression that evaluates to one-or-more tag names; it
#    is evaluated in the context of `IMAGE.docker.tag.GROUP`;
#    specifically:
#
#     * `$*` is set to IMAGE
#     * `$<` is set to a file containing the image ID
#
#    Additionally, you can override the EXPR on a per-image basis
#    by overriding the `docker.tag.GROUP` variable on a per-target
#    basis:
#
#        IMAGE.docker.tag.GROUP: docker.tag.GROUP = EXPR
#
#     > For example:
#     >
#     >   For the mast part, the Ambassador Pro images are
#     >    - built as  : `docker/$(NAME).docker`
#     >    - built from: `docker/$(NAME)/Dockerfile`
#     >    - pushed as : `docker.io/datawire/ambassador_pro:$(NAME)-$(VERSION)`
#     >   However, as an exception, the Ambassador Core image is
#     >    - built as  : `ambassador/ambassador.docker`
#     >    - pushed as : `docker.io/datawire/ambassador_pro:amb-core-$(VERSION)`
#     >
#     >   Additionally, we want to be able to push to a private
#     >   in-cluster registry for testing before we do a release.  The
#     >   tag names pushed to the cluster should be based on the image
#     >   ID, so that we don't need to configure a funny
#     >   ImagePullPolicy during testing.
#     >
#     >   We accomplish this by saying:
#     >
#     >       docker.tag.release = docker.io/datawire/ambassador_pro:$(notdir $*)-$(VERSION)
#     >       include build-aux/docker-cluster.mk # docker-cluster.mk sets the `docker.tag.cluster` variable
#     >       include build-aux/docker.mk
#     >       # The above will cause docker.mk to define targets:
#     >       #  - %.docker.tag.release
#     >       #  - %.docker.push.release
#     >       #  - %.docker.tag.cluster
#     >       #  - %.docker.push.cluster
#     >
#     >       # Override the release name a specific image.
#     >       # Release ambassador/ambassador.docker
#     >       #  - based on the above    : docker.io/datawire/ambassador_pro:ambassador-$(VERSION)
#     >       #  - after being overridden: docker.io/datawire/ambassador_pro:amb-core-$(VERSION)
#     >       ambassador/ambassador.docker.tag.release: docker.tag.release = docker.io/datawire/ambassador_pro:amb-core-$(VERSION)
#     >
#     >   and having our
#     >    - `build` target depend on `NAME.docker.tag.release` (for each NAME).
#
# ## Pushing ##
#
#    Pushing a tag: You can push tags that have been created with
#    `IMAGE.docker.tag.GROUP` (see above) by depending on
#    `IMAGE.docker.push.GROUP`.
#
#     > For example:
#     >
#     >   Based on the above Ambassador Pro example in the "Tagging"
#     >   section, we have our
#     >    - `check` target depend on `NAME.docker.push.cluster` (for each NAME).
#     >    - `release` target depend on `NAME.docker.push.release` (for each NAME).
#
# ## Cleaning ##
#
#     - Clean up: You can untag (if there are any tags) and remove an
#       image by having your `clean` target depend on
#       `SOMEPATH.docker.clean`.  Because docker.mk does not have a
#       listing of all the images you may ask it to build, these are
#       NOT automatically added to the common.mk 'clean' target, and
#       you MUST do that yourself.
#
# ## More build-rule examples ##
#
#     1. You might want to specify `docker build` arguments, like
#        `--build-arg=` or `-f`:
#
#            # Set a custom --build-arg, and use a funny Dockerfile name
#            myimage.docker: $(tools/move-ifchanged) FORCE
#            	docker build --iidfile=$(@D)/.tmp.$(@F).tmp --build-arg=FOO=BAR -f Dockerfile.myimage .
#            	$(tools/move-ifchanged) $(@D)/.tmp.$(@F).tmp $@
#
#     2. In `ambassador.git`, building the Envoy binary is slow, so we
#        might want to try pulling it from a build-cache Docker
#        repository, instead of building it locally:
#
#            # Building this is expensive, so try grabbing a cached
#            # version before trying to build it.  This goes ahead and
#            # tags the image, for caching purposes.
#            base-envoy.docker: $(tools/write-ifchanged) $(var.)BASE_ENVOY_IMAGE_CACHE
#            	if ! docker run --rm --entrypoint=true $(BASE_ENVOY_IMAGE_CACHE); then \
#            		$(MAKE) envoy-bin/envoy-static-stripped
#            		docker build -t $(BASE_ENVOY_IMAGE_CACHE) -f Dockerfile.base-envoy; \
#            	fi
#            	docker image inspect $(BASE_ENVOY_IMAGE_CACHE) --format='{{.Id}}' | $(tools/write-ifchanged) $@
#
#     3. In `apro.git`, we have many Docker images to build; each with
#        a Dockerfile at `docker/NAME/Dockerfile`.  We accomplish this
#        with a simple pattern rule only slightly more complex than
#        the one given in the "Building" section:
#
#            %.docker: %/Dockerfile $(tools/move-ifchanged) FORCE
#            # Try with --pull, fall back to without --pull
#            	docker build --iidfile=$(@D)/.tmp.$(@F).tmp --pull $* || docker build --iidfile=$(@D)/.tmp.$(@F).tmp $*
#            	$(tools/move-ifchanged) $(@D)/.tmp.$(@F).tmp $@
#
#        The `--pull` is a good way to ensure that we incorporate any
#        patches to the base images that we build ours FROM.  However,
#        sometimes `--pull` doesn't work, because in at least one case
#        the the Dockerfile refers to a local image ID hash from a
#        previously built image; trying to pull that ID hash will
#        fail.
#
#        For many of these images, we have have Makefile-built
#        artifacts that we would like to include in the image.  We
#        accomplish this by simply declaring dependencies off of the
#        `docker/NAME.docker` files, and writing rules to copy the
#        artifacts in to the `docker/NAME/` directory:
#
#            # In this example, the `docker/app-sidecar/Dockerfile` image
#            # needs an already-compiled `ambex` binary.
#
#            # Declare the dependency...
#            docker/app-sidecar.docker: docker/app-sidecar/ambex
#
#            # ... and copy it in to the Docker context
#            docker/app-sidecar/ambex: bin_linux_amd64/ambex
#            	cp $< $@

ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_docker.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_docker.mk))prelude.mk

#
# Inputs

_docker.tag.groups = $(patsubst docker.tag.%,%,$(filter docker.tag.%,$(.VARIABLES)))
# clean.groups is separate from tag.groups as a special-case for docker-cluster.mk
_docker.clean.groups += $(_docker.tag.groups)

#
# Output variables

HAVE_DOCKER      = $(call lazyonce,HAVE_DOCKER,$(shell which docker 2>/dev/null))
docker.LOCALHOST = $(if $(filter darwin,$(GOHOSTOS)),host.docker.internal,localhost)

#
# Output targets

%.docker.clean: $(addprefix %.docker.clean.,$(_docker.clean.groups))
	if [ -e $*.docker ]; then docker image rm "$$(cat $*.docker)" || true; fi
# It "shouldn't" need the ".*" suffix, but it makes it easier to hook in with things like a .stamp file
	rm -f $*.docker $*.docker.*
.PHONY: %.docker.clean

# Evaluate _docker.tag.rule with _docker.tag.group=TAG_GROUPNAME for
# each docker.tag.TAG_GROUPNAME variable.
#
# Add a set of %.docker.tag.TAG_GROUPNAME and
# %.docker.push.TAG_GROUPNAME targets that tag and push the docker image.
define _docker.tag.rule
  # file contents:
  #   line 1: image ID
  #   line 2: tag 1
  #   line 3: tag 2
  #   ...
  %.docker.tag.$(_docker.tag.group): %.docker $$(tools/write-dockertagfile) FORCE
  # The 'foreach' is to handle newlines as normal whitespace
	printf '%s\n' $$$$(cat $$<) $$(foreach v,$$(docker.tag.$(_docker.tag.group)),$$v) | $$(tools/write-dockertagfile) $$@

  # file contents:
  #   line 1: image ID
  #   line 2: tag 1
  #   line 3: tag 2
  #   ...
  %.docker.push.$(_docker.tag.group): %.docker.tag.$(_docker.tag.group) FORCE
	@set -e; { \
	  if cmp -s $$< $$@; then \
	    printf "$${CYN}==> $${GRN}Already pushed $${BLU}$$$$(sed -n 2p $$@)$${END}\n"; \
	  else \
	    printf "$${CYN}==> $${GRN}Pushing $${BLU}$$$$(sed -n 2p $$<)$${GRN}...$${END}\n"; \
	    sed 1d $$< | xargs -n1 docker push; \
	    cat $$< > $$@; \
	  fi; \
	}

  %.docker.clean.$(_docker.tag.group):
	if [ -e $$*.docker.tag.$(_docker.tag.group) ]; then docker image rm -- $$$$(sed 1d $$*.docker.tag.$(_docker.tag.group)) || true; fi
	rm -f $$*.docker.tag.$(_docker.tag.group) $$*.docker.push.$(_docker.tag.group)
  .PHONY: %.docker.clean.$(_docker.tag.group)
endef
$(foreach _docker.tag.group,$(_docker.tag.groups),$(eval $(_docker.tag.rule)))

endif
