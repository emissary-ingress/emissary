#
# Variables that the dev might set in the env or CLI


# Adjust to run just a subset of the tests.
ENVOY_TEST_LABEL ?= //contrib/golang/... //test/...
export ENVOY_TEST_LABEL

#
# Variables that are meant to be set by editing this file

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
ENVOY_REPO ?= https://github.com/datawire/envoy.git

# https://github.com/datawire/envoy/tree/rebase/release/v1.31.3
ENVOY_COMMIT ?= 628f5afc75a894a08504fa0f416269ec50c07bf9

ENVOY_COMPILATION_MODE ?= opt
# Increment BASE_ENVOY_RELVER on changes to `docker/base-envoy/Dockerfile`, or Envoy recipes.
# You may reset BASE_ENVOY_RELVER when adjusting ENVOY_COMMIT.
BASE_ENVOY_RELVER ?= 0

# Set to non-empty to enable compiling Envoy in FIPS mode.
FIPS_MODE ?=
export FIPS_MODE

# ENVOY_DOCKER_REPO ?= docker.io/emissaryingress/base-envoy
ENVOY_DOCKER_REPO ?= gcr.io/datawire/ambassador-base
ENVOY_DOCKER_VERSION ?= $(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)$(if $(FIPS_MODE),.FIPS)
ENVOY_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-$(ENVOY_DOCKER_VERSION)
# END LIST OF VARIABLES REQUIRING `make update-base`.

# How to set ENVOY_GO_CONTROL_PLANE_COMMIT: In github.com/envoyproxy/go-control-plane.git, the majority
# of commits have a commit message of the form "Mirrored from envoyproxy/envoy @ ${envoy.git_commit}".
# Look for the most recent one that names a commit that is an ancestor of our ENVOY_COMMIT.  If there
# are commits not of that form immediately following that commit, you can take them in too (but that's
# pretty uncommon).  Since that's a simple sentence, but it can be tedious to go through and check
# which commits are ancestors, I added `make guess-envoy-go-control-plane-commit` to do that in an
# automated way!  Still look at the commit yourself to make sure it seems sane; blindly trusting
# machines is bad, mmkay?
ENVOY_GO_CONTROL_PLANE_COMMIT = f888b4f71207d0d268dee7cb824de92848da9ede

# Set ENVOY_DOCKER_REPO to the list of mirrors to check
ENVOY_DOCKER_REPOS  = docker.io/emissaryingress/base-envoy
ENVOY_DOCKER_REPOS += gcr.io/datawire/ambassador-base

# Intro
include $(OSS_HOME)/build-aux/prelude.mk

# for builder.mk...
export ENVOY_DOCKER_TAG


#
#################### Envoy cxx and build image targets  #####################

$(OSS_HOME)/_cxx/envoy: FORCE
	@echo "Getting Envoy sources..."
# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
	@PS4=; set -ex; { \
	    unset GIT_DIR GIT_WORK_TREE; \
	    git init $@; \
	    cd $@; \
	    if git remote get-url origin &>/dev/null; then \
	        git remote set-url origin $(ENVOY_REPO); \
	    else \
	        git remote add origin $(ENVOY_REPO); \
	    fi; \
	    if [[ $(ENVOY_REPO) == http://github.com/* || $(ENVOY_REPO) == https://github.com/* || $(ENVOY_REPO) == git://github.com/* ]]; then \
	        git remote set-url --push origin git@github.com:$(word 3,$(subst /, ,$(ENVOY_REPO)))/$(patsubst %.git,%,$(word 4,$(subst /, ,$(ENVOY_REPO)))).git; \
	    fi; \
	    git fetch --tags origin; \
	    if [ $(ENVOY_COMMIT) != '-' ]; then \
	        git checkout $(ENVOY_COMMIT); \
	    elif ! git rev-parse HEAD >/dev/null 2>&1; then \
	        git checkout origin/master; \
	    fi; \
	}

$(OSS_HOME)/_cxx/envoy.clean: %.clean:
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $*)
clobber: $(OSS_HOME)/_cxx/envoy.clean

# cleanup existing build outputs
$(OSS_HOME)/_cxx/envoy-docker-build.clean: %.clean:
	$(if $(filter-out -,$(ENVOY_COMMIT)),sudo rm -rf $*)
clobber: $(OSS_HOME)/_cxx/envoy-docker-build.clean

$(OSS_HOME)/_cxx/envoy-build-image.txt: $(OSS_HOME)/_cxx/envoy $(tools/write-ifchanged) FORCE
	@PS4=; set -ex -o pipefail; { \
	    pushd $</ci; \
	    echo "$$(pwd)"; \
	    . envoy_build_sha.sh; \
	    popd; \
	    echo docker.io/envoyproxy/envoy-build-ubuntu:$$ENVOY_BUILD_SHA | $(tools/write-ifchanged) $@; \
	}
clean: $(OSS_HOME)/_cxx/envoy-build-image.txt.rm

# cleanup build artifacts
clean: $(OSS_HOME)/docker/base-envoy/envoy-static.rm
clean: $(OSS_HOME)/docker/base-envoy/envoy-static-stripped.rm
clean: $(OSS_HOME)/docker/base-envoy/envoy-static.dwp.rm

################################# Compile Custom Envoy Protos ######################################

# copy raw protos and compiled go protos into emissary-ingress
.PHONY compile-envoy-protos:
compile-envoy-protos: $(OSS_HOME)/_cxx/envoy-build-image.txt
	$(OSS_HOME)/_cxx/tools/compile-protos.sh

################################# Envoy Build PhonyTargets #########################################

# helper to trigger the clone of the datawire/envoy repository
.PHONY: clone-envoy
clone-envoy: $(OSS_HOME)/_cxx/envoy

# clean up envoy resources
.PHONY: clean-envoy
clean-envoy:
	cd $(OSS_HOME)/_cxx/envoy && ./ci/run_envoy_docker.sh "./ci/do_ci.sh 'clean'"

# Check to see if we have already built and push an image for the
.PHONY: verify-base-envoy
verify-base-envoy:
	@PS4=; set -ex; { \
	    if docker pull $(ENVOY_DOCKER_TAG); then \
	        echo 'Already up-to-date: $(ENVOY_DOCKER_TAG)'; \
	        ENVOY_VERSION_OUTPUT=$$(docker run --platform="$(BUILD_ARCH)" --rm -it --entrypoint envoy-static-stripped $(ENVOY_DOCKER_TAG) --version | grep "version:"); \
	        ENVOY_VERSION_EXPECTED="envoy-static-stripped .*version:.* $(ENVOY_COMMIT)/.*"; \
	        if ! echo "$$ENVOY_VERSION_OUTPUT" | grep "$$ENVOY_VERSION_EXPECTED"; then \
	            { set +x; } &>/dev/null; \
	            echo "error: Envoy base image $(ENVOY_DOCKER_TAG) contains envoy-static-stripped binary that reported an unexpected version string!" \
	                 "See ENVOY_VERSION_OUTPUT and ENVOY_VERSION_EXPECTED in the output above. This error is usually not recoverable." \
	                 "You may need to rebuild the Envoy base image after either updating ENVOY_COMMIT or bumping BASE_ENVOY_RELVER" \
	                 "(or both, depending on what you are doing)."; \
							exit 1; \
	        fi; \
					echo "Nothing to build at this time"; \
					exit 0; \
	    fi; \
	}

# builds envoy using release settings, see https://github.com/envoyproxy/envoy/blob/main/ci/README.md for additional
# details on configuring builds
.PHONY: build-envoy
build-envoy: $(OSS_HOME)/_cxx/envoy-build-image.txt
	$(OSS_HOME)/_cxx/tools/build-envoy.sh

# build the base-envoy containers and tags them locally, this requires running `build-envoy` first.
.PHONY: build-base-envoy-image
build-base-envoy-image: $(OSS_HOME)/_cxx/envoy-build-image.txt
	docker build --platform="$(BUILD_ARCH)" -f $(OSS_HOME)/docker/base-envoy/Dockerfile.stripped -t $(ENVOY_DOCKER_TAG) $(OSS_HOME)/docker/base-envoy

# Allows pushing the docker image independent of building envoy and docker containers
# Note, bump the BASE_ENVOY_RELVER and re-build before pushing when making non-commit changes to have a unique image tag.
.PHONY: push-base-envoy-image
push-base-envoy-image:
	docker push $(ENVOY_DOCKER_TAG)


# `make update-base`: Recompile Envoy and do all of the related things.
.PHONY: update-base
update-base: $(OSS_HOME)/_cxx/envoy-build-image.txt
	$(MAKE) verify-base-envoy
	$(MAKE) build-envoy
	$(MAKE) build-base-envoy-image
	$(MAKE) push-base-envoy-image
	$(MAKE) compile-envoy-protos

.PHONY: check-envoy
check-envoy: $(OSS_HOME)/_cxx/envoy-build-image.txt
	$(OSS_HOME)/_cxx/tools/test-envoy.sh;

.PHONY: envoy-shell
envoy-shell: $(OSS_HOME)/_cxx/envoy-build-image.txt
	cd $(OSS_HOME)/_cxx/envoy && ./ci/run_envoy_docker.sh bash || true;

################################# Go-control-plane Targets ####################################
#
# Recipes used by `make generate`; files that get checked into Git (i.e. protobufs and Go code)
#
# These targets are depended on by `make generate` in `build-aux/generate.mk`.


# See the comment on ENVOY_GO_CONTROL_PLANE_COMMIT at the top of the file for more explanation on how this target works.
guess-envoy-go-control-plane-commit: # Have the computer suggest a value for ENVOY_GO_CONTROL_PLANE_COMMIT
guess-envoy-go-control-plane-commit: $(OSS_HOME)/_cxx/envoy $(OSS_HOME)/_cxx/go-control-plane
	@echo
	@echo '######################################################################'
	@echo
	@set -e; { \
	  (cd $(OSS_HOME)/_cxx/go-control-plane && git log --format='%H %s' origin/main) | sed -n 's, Mirrored from envoyproxy/envoy @ , ,p' | \
	  while read -r go_commit cxx_commit; do \
	    if (cd $(OSS_HOME)/_cxx/envoy && git merge-base --is-ancestor "$$cxx_commit" $(ENVOY_COMMIT) 2>/dev/null); then \
	      echo "ENVOY_GO_CONTROL_PLANE_COMMIT = $$go_commit"; \
	      break; \
	    fi; \
	  done; \
	}
.PHONY: guess-envoy-go-control-plane-commit

# The unmodified go-control-plane
$(OSS_HOME)/_cxx/go-control-plane: FORCE
	@echo "Getting Envoy go-control-plane sources..."
# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
	@PS4=; set -ex; { \
	    unset GIT_DIR GIT_WORK_TREE; \
	    git init $@; \
	    cd $@; \
	    if git remote get-url origin &>/dev/null; then \
	        git remote set-url origin https://github.com/envoyproxy/go-control-plane; \
	    else \
	        git remote add origin https://github.com/envoyproxy/go-control-plane; \
	    fi; \
	    git fetch --tags origin; \
	    git checkout $(ENVOY_GO_CONTROL_PLANE_COMMIT); \
	}

# The go-control-plane patched for our version of the protobufs
$(OSS_HOME)/pkg/envoy-control-plane: $(OSS_HOME)/_cxx/go-control-plane FORCE
	rm -rf $@
	@PS4=; set -ex; { \
	  unset GIT_DIR GIT_WORK_TREE; \
	  tmpdir=$$(mktemp -d); \
	  trap 'rm -rf "$$tmpdir"' EXIT; \
	  cd "$$tmpdir"; \
	  cd $(OSS_HOME)/_cxx/go-control-plane; \
	  cp -r $$(git ls-files ':[A-Z]*' ':!Dockerfile*' ':!Makefile') pkg/* ratelimit "$$tmpdir"; \
	  find "$$tmpdir" -name '*.go' -exec sed -E -i.bak \
	    -e 's,github\.com/envoyproxy/go-control-plane/pkg,github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane,g' \
	    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/emissary-ingress/emissary/v3/pkg/api/envoy,g' \
			-e 's,github\.com/envoyproxy/go-control-plane/ratelimit,github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/ratelimit,g' \
	    -- {} +; \
	  sed -i.bak -e 's/^package/\n&/' "$$tmpdir/log/log_test.go"; \
	  find "$$tmpdir" -name '*.bak' -delete; \
	  mv "$$tmpdir" $(abspath $@); \
	}
	cd $(OSS_HOME) && gofmt -w -s ./pkg/envoy-control-plane/

######################### Envoy Version and Mirror Check  #######################

old_envoy_commits = $(shell { \
	  { \
	    git log --patch --format='' -G'^ *ENVOY_COMMIT' -- _cxx/envoy.mk; \
	    git log --patch --format='' -G'^ *ENVOY_COMMIT' -- cxx/envoy.mk; \
	    git log --patch --format='' -G'^ *ENVOY_COMMIT' -- Makefile; \
	  } | sed -En 's/^.*ENVOY_COMMIT *\?= *//p'; \
	  git log --patch --format='' -G'^ *ENVOY_BASE_IMAGE' 511ca54c3004019758980ba82f708269c373ba28 -- Makefile | sed -n 's/^. *ENVOY_BASE_IMAGE.*-g//p'; \
	  git log --patch --format='' -G'FROM.*envoy.*:' 7593e7dca9aea2f146ddfd5a3676bcc30ee25aff -- Dockerfile | sed -n '/FROM.*envoy.*:/s/.*://p' | sed -e 's/ .*//' -e 's/.*-g//' -e 's/.*-//' -e '/^latest$$/d'; \
	} | uniq)

lost_history += 251b7d345 # mentioned in a605b62ee (wip - patched and fixed authentication, Gabriel, 2019-04-04)
lost_history += 27770bf3d # mentioned in 026dc4cd4 (updated envoy image, Gabriel, 2019-04-04)

check-envoy-version: ## Check that Envoy version has been pushed to the right places
check-envoy-version: $(OSS_HOME)/_cxx/envoy
	# First, we're going to check whether the Envoy commit is tagged, which
	# is one of the things that has to happen before landing a PR that bumps
	# the ENVOY_COMMIT.
	#
	# We strictly check for tags matching 'datawire-*' to remove the
	# temptation to jump the gun and create an 'ambassador-*' or
	# 'emissary-*' tag before we know that's actually the commit that will
	# be in the released Ambassador/Emissary.
	#
	# Also, don't just check the tip of the PR ('HEAD'), also check that all
	# intermediate commits in the PR are also (ancestors of?) a tag.  We
	# don't want history to get lost!
	set -e; { \
	  cd $<; unset GIT_DIR GIT_WORK_TREE; \
	  for commit in HEAD $(filter-out $(lost_history),$(old_envoy_commits)); do \
	   echo "=> checking Envoy commit $$commit"; \
	   desc=$$(git describe --tags --contains --match='datawire-*' "$$commit"); \
	   [[ "$$desc" == datawire-* ]]; \
	   echo "   got $$desc"; \
	  done; \
	}
	# Now, we're going to check that the Envoy Docker images have been
	# pushed to all of the mirrors, which is another thing that has to
	# happen before landing a PR that bumps the ENVOY_COMMIT.
	#
	# We "could" use `docker manifest inspect` instead of `docker
	# pull` to test that these exist without actually pulling
	# them... except that gcr.io doesn't allow `manifest inspect`.
	# So just go ahead and do the `pull` :(
	$(foreach ENVOY_DOCKER_REPO,$(ENVOY_DOCKER_REPOS), docker pull $(ENVOY_DOCKER_TAG) >/dev/null$(NL))
.PHONY: check-envoy-version
