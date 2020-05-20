srcdir := $(OSS_HOME)/cxx

include $(OSS_HOME)/build-aux/prelude.mk

YES_I_AM_OK_WITH_COMPILING_ENVOY ?=
YES_I_AM_UPDATING_THE_BASE_IMAGES ?=
ENVOY_TEST_LABEL ?= //test/...

_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
  ENVOY_REPO ?= $(if $(IS_PRIVATE),git@github.com:datawire/envoy-private.git,git://github.com/datawire/envoy.git)
  ENVOY_COMMIT ?= 33ff697fc665b25463c7d64916ee6736782e2e76
  ENVOY_COMPILATION_MODE ?= opt
  # Increment BASE_ENVOY_RELVER on changes to `docker/base-envoy/Dockerfile`, or Envoy recipes.
  # You may reset BASE_ENVOY_RELVER when adjusting ENVOY_COMMIT.
  BASE_ENVOY_RELVER ?= 1

  ENVOY_DOCKER_REPO ?= docker.io/datawire$(if $(IS_PRIVATE),-private)/ambassador-base
  ENVOY_DOCKER_VERSION ?= $(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)
  ENVOY_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-$(ENVOY_DOCKER_VERSION)
# END LIST OF VARIABLES REQUIRING `make update-base`.

# for builder.sh...
export ENVOY_DOCKER_TAG

#
# Envoy build

$(srcdir)/envoy: FORCE
	@echo "Getting Envoy sources..."
# Migrate from old layouts
	@set -e; { \
	    if ! test -d $@; then \
	        if test -d $(OSS_HOME)/envoy; then \
	            set -x; \
	            mv $(OSS_HOME)/envoy $@; \
	        elif test -d $(OSS_HOME)/envoy-src; then \
	            set -x; \
	            mv $(OSS_HOME)/envoy-src $@; \
	        fi; \
	    fi; \
	}
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

$(srcdir)/envoy-build-image.txt: $(srcdir)/envoy $(WRITE_IFCHANGED) FORCE
	@PS4=; set -ex -o pipefail; { \
	    pushd $</ci; \
	    echo "$$(pwd)"; \
	    . envoy_build_sha.sh; \
	    popd; \
	    echo docker.io/envoyproxy/envoy-build-ubuntu@sha256:$$ENVOY_BUILD_SHA | $(WRITE_IFCHANGED) $@; \
	}

$(srcdir)/envoy-build-container.txt: $(srcdir)/envoy-build-image.txt FORCE
	@PS4=; set -ex; { \
	    if [ $@ -nt $< ] && docker exec $$(cat $@) true; then \
	        exit 0; \
	    fi; \
	    if [ -e $@ ]; then \
	        docker kill $$(cat $@) || true; \
	    fi; \
	    docker run --detach --rm --privileged --volume=envoy-build:/root:rw $$(cat $<) tail -f /dev/null > $@; \
	}

%-container.txt.clean:
	@PS4=; set -ex; { \
	    if [ -e $*-container.txt ]; then \
	        docker kill $$(cat $*-container.txt) || true; \
	    fi; \
	}
	rm -f $*-container.txt
.PHONY: %-container.txt.clean

# We do everything with rsync and a persistent build-container
# (instead of using a volume), because
#  1. Docker for Mac's osxfs is very slow, so volumes are bad for
#     macOS users.
#  2. Volumes mounts just straight-up don't work for people who use
#     Minikube's dockerd.
ENVOY_SYNC_HOST_TO_DOCKER = rsync -Pav --delete --blocking-io -e "docker exec -i" $(srcdir)/envoy/ $$(cat $(srcdir)/envoy-build-container.txt):/root/envoy
ENVOY_SYNC_DOCKER_TO_HOST = rsync -Pav --delete --blocking-io -e "docker exec -i" $$(cat $(srcdir)/envoy-build-container.txt):/root/envoy/ $(srcdir)/envoy/

ENVOY_BASH.cmd = bash -c 'PS4=; set -ex; $(ENVOY_SYNC_HOST_TO_DOCKER); trap '\''$(ENVOY_SYNC_DOCKER_TO_HOST)'\'' EXIT; '$(call quote.shell,$1)
ENVOY_BASH.deps = $(srcdir)/envoy-build-container.txt

$(OSS_HOME)/docker/base-envoy/envoy-static: $(ENVOY_BASH.deps) FORCE
	mkdir -p $(@D)
	@PS4=; set -ex; { \
	    if [ '$(ENVOY_COMMIT)' != '-' ] && docker run --rm --entrypoint=true $(ENVOY_DOCKER_TAG); then \
	        rsync -Pav --blocking-io -e 'docker run --rm -i' $$(docker image inspect $(ENVOY_DOCKER_TAG) --format='{{.Id}}' | sed 's/^sha256://'):/usr/local/bin/envoy-static $@; \
	    else \
	        if [ -z '$(YES_I_AM_UPDATING_THE_BASE_IMAGES)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: failed to pull $(ENVOY_DOCKER_TAG), but $$YES_I_AM_UPDATING_THE_BASE_IMAGES is not set'; \
	            echo '       If you are trying to update the base images, then set that variable to a non-empty value.'; \
	            echo '       If you are not trying to update the base images, then check your network connection and Docker credentials.'; \
	            exit 1; \
	        fi; \
	        if [ -z '$(YES_I_AM_OK_WITH_COMPILING_ENVOY)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: Envoy compilation triggered, but $$YES_I_AM_OK_WITH_COMPILING_ENVOY is not set'; \
	            exit 1; \
	        fi; \
	        $(call ENVOY_BASH.cmd, \
	            docker exec --workdir=/root/envoy $$(cat $(srcdir)/envoy-build-container.txt) /bin/bash -c "export CC=/opt/llvm/bin/clang && export CXX=/opt/llvm/bin/clang++ && bazel build --verbose_failures -c $(ENVOY_COMPILATION_MODE) --config=clang //source/exe:envoy-static;"; \
	            rsync -Pav --blocking-io -e 'docker exec -i' $$(cat $(srcdir)/envoy-build-container.txt):/root/envoy/bazel-bin/source/exe/envoy-static $@; \
	        ); \
	    fi; \
	}
%-stripped: % $(srcdir)/envoy-build-container.txt
	@PS4=; set -ex; { \
	    rsync -Pav --blocking-io -e 'docker exec -i' $< $$(cat $(srcdir)/envoy-build-container.txt):/tmp/$(<F); \
	    docker exec $$(cat $(srcdir)/envoy-build-container.txt) strip /tmp/$(<F) -o /tmp/$(@F); \
	    rsync -Pav --blocking-io -e 'docker exec -i' $$(cat $(srcdir)/envoy-build-container.txt):/tmp/$(@F) $@; \
	}

check-envoy: ## Run the Envoy test suite
check-envoy: $(ENVOY_BASH.deps)
	  @echo 'Testing envoy with Bazel label: "$(ENVOY_TEST_LABEL)"'; \
	  $(call ENVOY_BASH.cmd, \
	     docker exec --workdir=/root/envoy $$(cat $(srcdir)/envoy-build-container.txt) /bin/bash -c 'export CC=/opt/llvm/bin/clang && export CXX=/opt/llvm/bin/clang++ && bazel test --config=clang --test_output=errors --verbose_failures -c dbg --test_env=ENVOY_IP_TEST_VERSIONS=v4only $(ENVOY_TEST_LABEL);' \
	 )
.PHONY: check-envoy

envoy-shell: ## Run a shell in the Envoy build container
envoy-shell: $(ENVOY_BASH.deps)
	$(call ENVOY_BASH.cmd, \
	    docker exec -it $$(cat $(srcdir)/envoy-build-container.txt) /bin/bash || true; \
	)
.PHONY: envoy-shell

#
# Envoy generate

$(OSS_HOME)/api/envoy: $(srcdir)/envoy
	rsync --recursive --delete --delete-excluded --prune-empty-dirs --include='*/' --include='*.proto' --exclude='*' $</api/envoy/ $@

update-base: $(srcdir)/envoy-build-image.txt $(OSS_HOME)/docker/base-envoy/envoy-static $(OSS_HOME)/docker/base-envoy/envoy-static-stripped
	docker build --build-arg=base=$$(cat $(srcdir)/envoy-build-image.txt) -t $(ENVOY_DOCKER_TAG) $(OSS_HOME)/docker/base-envoy
	$(MAKE) generate
	if [ '$(ENVOY_COMMIT)' != '-' ]; then docker push $(ENVOY_DOCKER_TAG); fi
.PHONY: update-base

#
# Envoy clean

clean: _clean-envoy
clobber: _clobber-envoy

_clean-envoy: _clean-envoy-old
_clean-envoy: $(srcdir)/envoy-build-container.txt.clean
	rm -f $(srcdir)/envoy-build-image.txt
_clobber-envoy: _clean-envoy
	rm -f $(OSS_HOME)/docker/base-envoy/envoy-static
	rm -f $(OSS_HOME)/docker/base-envoy/envoy-static-stripped
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(srcdir)/envoy)
.PHONY: _clean-envoy _clobber-envoy

# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.

# 2019-10-11
_clean-envoy-old: $(OSS_HOME)/envoy-build-container.txt.clean

_clean-envoy-old:
# 2020-02-20
	rm -f $(OSS_HOME)/cxx/envoy-static
	rm -f $(OSS_HOME)/bin_linux_amd64/envoy-static
	rm -f $(OSS_HOME)/bin_linux_amd64/envoy-static-stripped
# 2019-10-11
	rm -rf $(OSS_HOME)/envoy-bin
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(OSS_HOME)/envoy-src)
	rm -f $(OSS_HOME)/envoy-build-image.txt
# older than that
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(OSS_HOME)/envoy)
.PHONY: _clean-envoy-old
