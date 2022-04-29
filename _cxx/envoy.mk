#
# Variables that the dev might set in the env or CLI

# Set to non-empty to enable compiling Envoy as-needed.
YES_I_AM_OK_WITH_COMPILING_ENVOY ?=
# Adjust to run just a subset of the tests.
ENVOY_TEST_LABEL ?= //test/...
# Set RSYNC_EXTRAS=Pv or something to increase verbosity.
RSYNC_EXTRAS ?=

#
# Variables that are meant to be set by editing this file

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
  ENVOY_REPO ?= $(if $(IS_PRIVATE),git@github.com:datawire/envoy-private.git,https://github.com/datawire/envoy.git)
  ENVOY_COMMIT ?= 4ce93dc3ace00ae9108b179d0afaceac13f4602a
  ENVOY_COMPILATION_MODE ?= opt
  # Increment BASE_ENVOY_RELVER on changes to `docker/base-envoy/Dockerfile`, or Envoy recipes.
  # You may reset BASE_ENVOY_RELVER when adjusting ENVOY_COMMIT.
  BASE_ENVOY_RELVER ?= 0

  ENVOY_DOCKER_REPO ?= $(if $(IS_PRIVATE),quay.io/datawire-private/base-envoy,docker.io/emissaryingress/base-envoy)
  ENVOY_DOCKER_VERSION ?= $(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)
  ENVOY_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-$(ENVOY_DOCKER_VERSION)
  ENVOY_FULL_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-full-$(ENVOY_DOCKER_VERSION)
# END LIST OF VARIABLES REQUIRING `make update-base`.

# How to set ENVOY_GO_CONTROL_PLANE_COMMIT: In github.com/envoyproxy/go-control-plane.git, the majority
# of commits have a commit message of the form "Mirrored from envoyproxy/envoy @ ${envoy.git_commit}".
# Look for the most recent one that names a commit that is an ancestor of our ENVOY_COMMIT.  If there
# are commits not of that form immediately following that commit, you can take them in too (but that's
# pretty uncommon).  Since that's a simple sentence, but it can be tedious to go through and check
# which commits are ancestors, I added `make guess-envoy-go-control-plane-commit` to do that in an
# automated way!  Still look at the commit yourself to make sure it seems sane; blindly trusting
# machines is bad, mmkay?
ENVOY_GO_CONTROL_PLANE_COMMIT = v0.9.6

# Set ENVOY_DOCKER_REPO to the list of mirrors that we should
# sanity-check that things get pushed to.
ifneq ($(IS_PRIVATE),)
  # If $(IS_PRIVATE), then just the private repo...
  ENVOY_DOCKER_REPOS = $(ENVOY_DOCKER_REPO)
else
  # ...otherwise, this list of repos:
  ENVOY_DOCKER_REPOS  = docker.io/emissaryingress/base-envoy
  ENVOY_DOCKER_REPOS += gcr.io/datawire/ambassador-base
endif

#
# Intro

include $(OSS_HOME)/build-aux/prelude.mk

# for builder.mk...
export ENVOY_DOCKER_TAG

check-envoy-version: ## Check that Envoy version has been pushed to the right places
check-envoy-version: $(OSS_HOME)/_cxx/envoy
	# First, we're going to check whether the envoy commit is tagged, which
	# is one of the things that has to happen before landing a PR that bumps
	# the ENVOY_COMMIT.
	cd $< && unset GIT_DIR GIT_WORK_TREE && git describe --tags --exact-match
	# Now, we're going to check that the Envoy Docker images have been
	# pushed to all of the mirrors, which is another thing that has to
	# happen before landing a PR that bumps the ENVOY_COMMIT.
	#
	# We "could" use `docker manifest inspect` instead of `docker
	# pull` to test that these exist without actually pulling
	# them... except that gcr.io doesn't allow `manifest inspect`.
	# So just go ahead and do the `pull` :(
	@PS4=; set -ex; $(foreach ENVOY_DOCKER_REPO,$(ENVOY_DOCKER_REPOS), docker pull $(ENVOY_DOCKER_TAG) >/dev/null; )
	@PS4=; set -ex; $(foreach ENVOY_DOCKER_REPO,$(ENVOY_DOCKER_REPOS), docker pull $(ENVOY_FULL_DOCKER_TAG) >/dev/null; )
.PHONY: check-envoy-version

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

#
# Envoy sources and build container

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
.PHONY: $(OSS_HOME)/_cxx/envoy.clean
clobber: $(OSS_HOME)/_cxx/envoy.clean

$(OSS_HOME)/_cxx/envoy-build-image.txt: $(OSS_HOME)/_cxx/envoy $(WRITE_IFCHANGED) FORCE
	@PS4=; set -ex -o pipefail; { \
	    pushd $</ci; \
	    echo "$$(pwd)"; \
	    . envoy_build_sha.sh; \
	    popd; \
	    echo docker.io/envoyproxy/envoy-build-ubuntu:$$ENVOY_BUILD_SHA | $(WRITE_IFCHANGED) $@; \
	}
$(OSS_HOME)/_cxx/envoy-build-image.txt.clean: %.clean:
	rm -f $*
.PHONY: $(OSS_HOME)/_cxx/envoy-build-image.txt.clean
clean: $(OSS_HOME)/_cxx/envoy-build-image.txt.clean

$(OSS_HOME)/_cxx/envoy-build-container.txt: $(OSS_HOME)/_cxx/envoy-build-image.txt FORCE
	@PS4=; set -ex; { \
	    if [ $@ -nt $< ] && docker exec $$(cat $@) true; then \
	        exit 0; \
	    fi; \
	    if [ -e $@ ]; then \
	        docker kill $$(cat $@) || true; \
	    fi; \
	    docker run --detach --rm --privileged --volume=envoy-build:/root:rw $$(cat $<) tail -f /dev/null > $@; \
	}
$(OSS_HOME)/_cxx/envoy-build-container.txt.clean: %.clean:
	if [ -e $* ]; then docker kill $$(cat $*) || true; fi
	rm -f $*
	if docker volume inspect envoy-build &>/dev/null; then docker volume rm envoy-build >/dev/null; fi
.PHONY: $(OSS_HOME)/_cxx/envoy-build-container.txt.clean
clean: $(OSS_HOME)/_cxx/envoy-build-container.txt.clean

#
# Things that run in the Envoy build container
#
# We do everything with rsync and a persistent build-container
# (instead of using a volume), because
#  1. Docker for Mac's osxfs is very slow, so volumes are bad for
#     macOS users.
#  2. Volumes mounts just straight-up don't work for people who use
#     Minikube's dockerd.
ENVOY_SYNC_HOST_TO_DOCKER = rsync -a$(RSYNC_EXTRAS) --partial --delete --blocking-io -e "docker exec -i" $(OSS_HOME)/_cxx/envoy/ $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt):/root/envoy
ENVOY_SYNC_DOCKER_TO_HOST = rsync -a$(RSYNC_EXTRAS) --partial --delete --blocking-io -e "docker exec -i" $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt):/root/envoy/ $(OSS_HOME)/_cxx/envoy/

ENVOY_BASH.cmd = bash -c 'PS4=; set -ex; $(ENVOY_SYNC_HOST_TO_DOCKER); trap '\''$(ENVOY_SYNC_DOCKER_TO_HOST)'\'' EXIT; '$(call quote.shell,$1)
ENVOY_BASH.deps = $(OSS_HOME)/_cxx/envoy-build-container.txt

ENVOY_DOCKER.env += PATH=/opt/llvm/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ENVOY_DOCKER.env += CC=clang
ENVOY_DOCKER.env += CXX=clang++
ENVOY_DOCKER.env += CLANG_FORMAT=/opt/llvm/bin/clang-format
ENVOY_DOCKER_EXEC = docker exec --workdir=/root/envoy $(foreach e,$(ENVOY_DOCKER.env), --env=$e ) $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt)

$(OSS_HOME)/docker/base-envoy/envoy-static: $(ENVOY_BASH.deps) FORCE
	mkdir -p $(@D)
	@PS4=; set -ex; { \
	    if [ '$(ENVOY_COMMIT)' != '-' ] && docker run --rm --entrypoint=true $(ENVOY_FULL_DOCKER_TAG); then \
	        rsync -a$(RSYNC_EXTRAS) --partial --blocking-io -e 'docker run --rm -i' $$(docker image inspect $(ENVOY_FULL_DOCKER_TAG) --format='{{.Id}}' | sed 's/^sha256://'):/usr/local/bin/envoy-static $@; \
	    else \
	        if [ -z '$(YES_I_AM_OK_WITH_COMPILING_ENVOY)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: Envoy compilation triggered, but $$YES_I_AM_OK_WITH_COMPILING_ENVOY is not set'; \
	            exit 1; \
	        fi; \
	        $(call ENVOY_BASH.cmd, \
	            $(ENVOY_DOCKER_EXEC) bazel build --verbose_failures -c $(ENVOY_COMPILATION_MODE) --config=clang //source/exe:envoy-static; \
	            rsync -a$(RSYNC_EXTRAS) --partial --blocking-io -e 'docker exec -i' $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt):/root/envoy/bazel-bin/source/exe/envoy-static $@; \
	        ); \
	    fi; \
	}
$(OSS_HOME)/docker/base-envoy/envoy-static-stripped: %-stripped: % FORCE
	@PS4=; set -ex; { \
	    if [ '$(ENVOY_COMMIT)' != '-' ] && docker run --rm --entrypoint=true $(ENVOY_FULL_DOCKER_TAG); then \
	        rsync -a$(RSYNC_EXTRAS) --partial --blocking-io -e 'docker run --rm -i' $$(docker image inspect $(ENVOY_FULL_DOCKER_TAG) --format='{{.Id}}' | sed 's/^sha256://'):/usr/local/bin/$(@F) $@; \
	    else \
	        if [ -z '$(YES_I_AM_OK_WITH_COMPILING_ENVOY)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: Envoy compilation triggered, but $$YES_I_AM_OK_WITH_COMPILING_ENVOY is not set'; \
	            exit 1; \
	        fi; \
	        rsync -a$(RSYNC_EXTRAS) --partial --blocking-io -e 'docker exec -i' $< $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt):/tmp/$(<F); \
	        docker exec $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt) strip /tmp/$(<F) -o /tmp/$(@F); \
	        rsync -a$(RSYNC_EXTRAS) --partial --blocking-io -e 'docker exec -i' $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt):/tmp/$(@F) $@; \
	    fi; \
	}
$(OSS_HOME)/docker/base-envoy/envoy-static.clean $(OSS_HOME)/docker/base-envoy/envoy-static-stripped.clean: %.clean
	rm -f $*
.PHONY: $(OSS_HOME)/docker/base-envoy/envoy-static.clean $(OSS_HOME)/docker/base-envoy/envoy-static-stripped.clean
clobber: $(OSS_HOME)/docker/base-envoy/envoy-static.clean $(OSS_HOME)/docker/base-envoy/envoy-static-stripped.clean

check-envoy: ## Run the Envoy test suite
check-envoy: $(ENVOY_BASH.deps)
	@echo 'Testing envoy with Bazel label: "$(ENVOY_TEST_LABEL)"'; \
	$(call ENVOY_BASH.cmd, \
	     $(ENVOY_DOCKER_EXEC) bazel test --config=clang --test_output=errors --verbose_failures -c dbg --test_env=ENVOY_IP_TEST_VERSIONS=v4only $(ENVOY_TEST_LABEL); \
	 )
.PHONY: check-envoy

envoy-shell: ## Run a shell in the Envoy build container
envoy-shell: $(ENVOY_BASH.deps)
	$(call ENVOY_BASH.cmd, \
	    docker exec -it --workdir=/root/envoy $(foreach e,$(ENVOY_DOCKER.env), --env=$e ) $$(cat $(OSS_HOME)/_cxx/envoy-build-container.txt) /bin/bash || true; \
	)
.PHONY: envoy-shell

#
# Recipes used by `make generate`; files that get checked in to Git (i.e. protobufs and Go code)
#
# These targets are depended on by `make generate` in `build-aux/generate.mk`.

# Raw protobufs
$(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb: $(OSS_HOME)/api/%: $(OSS_HOME)/_cxx/envoy
	rsync --recursive --delete --delete-excluded --prune-empty-dirs --include='*/' --include='*.proto' --exclude='*' $</api/$*/ $@

# Go generated from the protobufs
$(OSS_HOME)/_cxx/envoy/build_go: $(ENVOY_BASH.deps) FORCE
	$(call ENVOY_BASH.cmd, \
	    $(ENVOY_DOCKER_EXEC) python3 -c 'from tools.api.generate_go_protobuf import generateProtobufs; generateProtobufs("/root/envoy/build_go")'; \
	)
	test -d $@ && touch $@
$(OSS_HOME)/pkg/api/pb $(OSS_HOME)/pkg/api/envoy: $(OSS_HOME)/pkg/api/%: $(OSS_HOME)/_cxx/envoy/build_go
	rm -rf $@
	@PS4=; set -ex; { \
	  unset GIT_DIR GIT_WORK_TREE; \
	  tmpdir=$$(mktemp -d); \
	  trap 'rm -rf "$$tmpdir"' EXIT; \
	  cp -r $</$* "$$tmpdir"; \
	  find "$$tmpdir" -type f \
	    -exec chmod 644 {} + \
	    -exec sed -E -i.bak \
	      -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/datawire/ambassador/pkg/api/envoy,g' \
	      -e 's,github\.com/envoyproxy/go-control-plane/pb,github.com/datawire/ambassador/pkg/api/pb,g' \
	      -- {} +; \
	  find "$$tmpdir" -name '*.bak' -delete; \
	  mv "$$tmpdir/$*" $@; \
	}

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
	  cp -r $$(git ls-files ':[A-Z]*' ':!Dockerfile*' ':!Makefile') pkg/* "$$tmpdir"; \
	  find "$$tmpdir" -name '*.go' -exec sed -E -i.bak \
	    -e 's,github\.com/envoyproxy/go-control-plane/pkg,github.com/datawire/ambassador/pkg/envoy-control-plane,g' \
	    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/datawire/ambassador/pkg/api/envoy,g' \
	    -- {} +; \
	  find "$$tmpdir" -name '*.bak' -delete; \
	  mv "$$tmpdir" $(abspath $@); \
	}
	cd $(OSS_HOME) && gofmt -w -s ./pkg/envoy-control-plane/

#
# `make update-base`: Recompile Envoy and do all of the related things.

update-base: $(OSS_HOME)/docker/base-envoy/envoy-static $(OSS_HOME)/docker/base-envoy/envoy-static-stripped $(OSS_HOME)/_cxx/envoy-build-image.txt
	@PS4=; set -ex; { \
	    if [ '$(ENVOY_COMMIT)' != '-' ] && docker pull $(ENVOY_FULL_DOCKER_TAG); then \
	        echo 'Already up-to-date: $(ENVOY_FULL_DOCKER_TAG)'; \
	        ENVOY_VERSION_OUTPUT=$$(docker run -it --entrypoint envoy-static $(ENVOY_FULL_DOCKER_TAG) --version | grep "version:"); \
	        ENVOY_VERSION_EXPECTED="envoy-static .*version:.* $(ENVOY_COMMIT)/.*"; \
	        if ! echo "$$ENVOY_VERSION_OUTPUT" | grep "$$ENVOY_VERSION_EXPECTED"; then \
	            { set +x; } &>/dev/null; \
	            echo "error: Envoy base image $(ENVOY_FULL_DOCKER_TAG) contains envoy-static binary that reported an unexpected version string!" \
	                 "See ENVOY_VERSION_OUTPUT and ENVOY_VERSION_EXPECTED in the output above. This error is usually not recoverable." \
	                 "You may need to rebuild the Envoy base image after either updating ENVOY_COMMIT or bumping BASE_ENVOY_RELVER" \
	                 "(or both, depending on what you are doing)."; \
	            exit 1; \
	        fi; \
	    else \
	        if [ -z '$(YES_I_AM_OK_WITH_COMPILING_ENVOY)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: Envoy compilation triggered, but $$YES_I_AM_OK_WITH_COMPILING_ENVOY is not set'; \
	            exit 1; \
	        fi; \
	        docker build --build-arg=base=$$(cat $(OSS_HOME)/_cxx/envoy-build-image.txt) -f $(OSS_HOME)/docker/base-envoy/Dockerfile -t $(ENVOY_FULL_DOCKER_TAG) $(OSS_HOME)/docker/base-envoy; \
	        if [ '$(ENVOY_COMMIT)' != '-' ]; then \
	            ENVOY_VERSION_OUTPUT=$$(docker run -it --entrypoint envoy-static $(ENVOY_FULL_DOCKER_TAG) --version | grep "version:"); \
	            ENVOY_VERSION_EXPECTED="envoy-static .*version:.* $(ENVOY_COMMIT)/.*"; \
	            if ! echo "$$ENVOY_VERSION_OUTPUT" | grep "$$ENVOY_VERSION_EXPECTED"; then \
	                { set +x; } &>/dev/null; \
	                echo "error: Envoy base image $(ENVOY_FULL_DOCKER_TAG) contains envoy-static binary that reported an unexpected version string!" \
	                     "See ENVOY_VERSION_OUTPUT and ENVOY_VERSION_EXPECTED in the output above. This error is usually not recoverable." \
	                     "You may need to rebuild the Envoy base image after either updating ENVOY_COMMIT or bumping BASE_ENVOY_RELVER" \
	                     "(or both, depending on what you are doing)."; \
	                exit 1; \
	            fi; \
	            docker push $(ENVOY_FULL_DOCKER_TAG); \
	        fi; \
	    fi; \
	}
	@PS4=; set -ex; { \
	    if [ '$(ENVOY_COMMIT)' != '-' ] && docker pull $(ENVOY_DOCKER_TAG); then \
	        echo 'Already up-to-date: $(ENVOY_DOCKER_TAG)'; \
	        ENVOY_VERSION_OUTPUT=$$(docker run -it --entrypoint envoy-static-stripped $(ENVOY_DOCKER_TAG) --version | grep "version:"); \
	        ENVOY_VERSION_EXPECTED="envoy-static-stripped .*version:.* $(ENVOY_COMMIT)/.*"; \
	        if ! echo "$$ENVOY_VERSION_OUTPUT" | grep "$$ENVOY_VERSION_EXPECTED"; then \
	            { set +x; } &>/dev/null; \
	            echo "error: Envoy base image $(ENVOY_DOCKER_TAG) contains envoy-static-stripped binary that reported an unexpected version string!" \
	                 "See ENVOY_VERSION_OUTPUT and ENVOY_VERSION_EXPECTED in the output above. This error is usually not recoverable." \
	                 "You may need to rebuild the Envoy base image after either updating ENVOY_COMMIT or bumping BASE_ENVOY_RELVER" \
	                 "(or both, depending on what you are doing)."; \
	            exit 1; \
	        fi; \
	    else \
	        if [ -z '$(YES_I_AM_OK_WITH_COMPILING_ENVOY)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: Envoy compilation triggered, but $$YES_I_AM_OK_WITH_COMPILING_ENVOY is not set'; \
	            exit 1; \
	        fi; \
	        docker build -f $(OSS_HOME)/docker/base-envoy/Dockerfile.stripped -t $(ENVOY_DOCKER_TAG) $(OSS_HOME)/docker/base-envoy; \
	        if [ '$(ENVOY_COMMIT)' != '-' ]; then \
	            ENVOY_VERSION_OUTPUT=$$(docker run -it --entrypoint envoy-static-stripped $(ENVOY_DOCKER_TAG) --version | grep "version:"); \
	            ENVOY_VERSION_EXPECTED="envoy-static-stripped .*version:.* $(ENVOY_COMMIT)/.*"; \
	            if ! echo "$$ENVOY_VERSION_OUTPUT" | grep "$$ENVOY_VERSION_EXPECTED"; then \
	                { set +x; } &>/dev/null; \
	                echo "error: Envoy base image $(ENVOY_DOCKER_TAG) contains envoy-static-stripped binary that reported an unexpected version string!" \
	                     "See ENVOY_VERSION_OUTPUT and ENVOY_VERSION_EXPECTED in the output above. This error is usually not recoverable." \
	                     "You may need to rebuild the Envoy base image after either updating ENVOY_COMMIT or bumping BASE_ENVOY_RELVER" \
	                     "(or both, depending on what you are doing)."; \
	                exit 1; \
	            fi; \
	            docker push $(ENVOY_DOCKER_TAG); \
	        fi; \
	    fi; \
	}
# `make generate` has to come *after* the above, because builder.sh will
# try to use the images that the above create.
	$(MAKE) generate
.PHONY: update-base
