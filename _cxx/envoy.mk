# yo
include $(OSS_HOME)/build-aux/prelude.mk

YES_I_AM_OK_WITH_COMPILING_ENVOY ?=
ENVOY_TEST_LABEL ?= //test/...

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
  ENVOY_REPO ?= $(if $(IS_PRIVATE),git@github.com:datawire/envoy-private.git,git://github.com/datawire/envoy.git)
  ENVOY_COMMIT ?= 7a33e53fd3d3c4befa53030797f344fcacaa61f4
  ENVOY_COMPILATION_MODE ?= opt
  # Increment BASE_ENVOY_RELVER on changes to `docker/base-envoy/Dockerfile`, or Envoy recipes.
  # You may reset BASE_ENVOY_RELVER when adjusting ENVOY_COMMIT.
  BASE_ENVOY_RELVER ?= 0

  ENVOY_DOCKER_REPO ?= $(if $(IS_PRIVATE),quay.io/datawire-private/ambassador-base,docker.io/datawire/ambassador-base)
  ENVOY_DOCKER_VERSION ?= $(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)
  ENVOY_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-$(ENVOY_DOCKER_VERSION)
  ENVOY_FULL_DOCKER_TAG ?= $(ENVOY_DOCKER_REPO):envoy-full-$(ENVOY_DOCKER_VERSION)
# END LIST OF VARIABLES REQUIRING `make update-base`.

# for builder.mk...
export ENVOY_DOCKER_TAG

#
# Envoy build

$(OSS_HOME)/_cxx/envoy: FORCE
	@echo "Getting Envoy sources..."
# Migrate from old layouts
	@set -e; { \
	    if ! test -d $@; then \
	        for old in $(OSS_HOME)/envoy $(OSS_HOME)/envoy-src $(OSS_HOME)/cxx/envoy; do \
	            if test -d $$old; then \
	                set -x; \
	                mv $$old $@; \
	                { set +x; } >&/dev/null; \
	                break; \
	            fi; \
	        done; \
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

$(OSS_HOME)/_cxx/go-control-plane: FORCE
	@echo "Getting Envoy go-control-plane sources..."
# Migrate from old layouts
	@set -e; { \
	    if ! test -d $@; then \
	        for old in $(OSS_HOME)/cxx/go-control-plane; do \
	            if test -d $$old; then \
	                set -x; \
	                mv $$old $@; \
	                { set +x; } >&/dev/null; \
	                break; \
	            fi; \
	        done; \
	    fi; \
	}
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

$(OSS_HOME)/_cxx/envoy-build-image.txt: $(OSS_HOME)/_cxx/envoy $(WRITE_IFCHANGED) FORCE
	@PS4=; set -ex -o pipefail; { \
	    pushd $</ci; \
	    echo "$$(pwd)"; \
	    . envoy_build_sha.sh; \
	    popd; \
	    echo docker.io/envoyproxy/envoy-build-ubuntu:$$ENVOY_BUILD_SHA | $(WRITE_IFCHANGED) $@; \
	}

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

%-container.txt.clean:
	@PS4=; set -ex; { \
	    if [ -e $*-container.txt ]; then \
	        docker kill $$(cat $*-container.txt) || true; \
	    fi; \
	}
	rm -f $*-container.txt
.PHONY: %-container.txt.clean

RSYNC_EXTRAS=
# RSYNC_EXTRAS=Pv

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
%-stripped: % FORCE
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
# Envoy generate

$(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb: $(OSS_HOME)/api/%: $(OSS_HOME)/_cxx/envoy
	rsync --recursive --delete --delete-excluded --prune-empty-dirs --include='*/' --include='*.proto' --exclude='*' $</api/$*/ $@

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

#
# Envoy clean

clean: _clean-envoy
clobber: _clobber-envoy

_clean-envoy: _clean-envoy-old
_clean-envoy: $(OSS_HOME)/_cxx/envoy-build-container.txt.clean
	@PS4=; set -e; { \
		if docker volume inspect envoy-build >/dev/null 2>&1; then \
			set -x ;\
			docker volume rm envoy-build >/dev/null ;\
		fi ;\
	}
	rm -f $(OSS_HOME)/_cxx/envoy-build-image.txt
_clobber-envoy: _clean-envoy
	rm -f $(OSS_HOME)/docker/base-envoy/envoy-static
	rm -f $(OSS_HOME)/docker/base-envoy/envoy-static-stripped
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(OSS_HOME)/_cxx/envoy)
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
