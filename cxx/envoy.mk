srcdir := $(patsubst %/,%,$(dir $(lastword $(MAKEFILE_LIST))))
topsrcdir := $(patsubst %/,%,$(dir $(firstword $(MAKEFILE_LIST))))

YES_I_AM_OK_WITH_COMPILING_ENVOY ?=
ENVOY_FILE ?= $(topsrcdir)/bin_linux_amd64/envoy-static-stripped

# IF YOU MESS WITH ANY OF THESE VALUES, YOU MUST RUN `make update-base`.
  ENVOY_REPO ?= $(if $(IS_PRIVATE),git@github.com:datawire/envoy-private.git,git://github.com/datawire/envoy.git)
  ENVOY_COMMIT ?= 6e6ae35f214b040f76666d86b30a6ad3ceb67046
  ENVOY_COMPILATION_MODE ?= opt

  # Increment BASE_ENVOY_RELVER on changes to `docker/base-envoy/Dockerfile`, or Envoy recipes
  BASE_ENVOY_RELVER ?= 6

  BASE_VERSION.envoy ?= $(BASE_ENVOY_RELVER).$(ENVOY_COMMIT).$(ENVOY_COMPILATION_MODE)
# END LIST OF VARIABLES REQUIRING `make update-base`.

#
# Envoy build

$(srcdir)/envoy: FORCE
	@echo "Getting Envoy sources..."
# Migrate from old layouts
	@set -e; { \
	    if ! test -d $@; then \
	        if test -d $(topsrcdir)/envoy; then \
	            set -x; \
	            mv $(topsrcdir)/envoy $@; \
	        elif test -d $(topsrcdir)/envoy-src; then \
	            set -x; \
	            mv $(topsrcdir)/envoy-src $@; \
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
	    . envoy_build_sha.sh; \
	    popd; \
	    echo docker.io/envoyproxy/envoy-build-ubuntu:$$ENVOY_BUILD_SHA | $(WRITE_IFCHANGED) $@; \
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

$(topsrcdir)/bin_linux_amd64/envoy-static: $(ENVOY_BASH.deps) FORCE
	mkdir -p $(@D)
	@PS4=; set -ex; { \
	    if docker run --rm --entrypoint=true $(BASE_IMAGE.envoy); then \
	        rsync -Pav --blocking-io -e 'docker run --rm -i' $$(docker image inspect $(BASE_IMAGE.envoy) --format='{{.Id}}' | sed 's/^sha256://'):/usr/local/bin/envoy $@; \
	    else \
	        if [ -z '$(YES_I_AM_UPDATING_THE_BASE_IMAGES)' ]; then \
	            { set +x; } &>/dev/null; \
	            echo 'error: failed to pull $(BASE_IMAGE.envoy), but $$YES_I_AM_UPDATING_THE_BASE_IMAGES is not set'; \
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
	            docker exec --workdir=/root/envoy $$(cat $(srcdir)/envoy-build-container.txt) bazel build --verbose_failures -c $(ENVOY_COMPILATION_MODE) //source/exe:envoy-static; \
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
	$(call ENVOY_BASH.cmd, \
	    docker exec --workdir=/root/envoy $$(cat $(srcdir)/envoy-build-container.txt) bazel test --verbose_failures -c dbg --test_env=ENVOY_IP_TEST_VERSIONS=v4only //test/...; \
	)
.PHONY: check-envoy

envoy-shell: ## Run a shell in the Envoy build container
envoy-shell: $(ENVOY_BASH.deps)
	$(call ENVOY_BASH.cmd, \
	    docker exec -it $$(cat $(srcdir)/envoy-build-container.txt) || true; \
	)
.PHONY: envoy-shell

base-envoy.docker.stamp: $(srcdir)/envoy-build-image.txt $(topsrcdir)/bin_linux_amd64/envoy-static
base-envoy.docker.stamp.DOCKER_OPTS = --build-arg=ENVOY_BUILD_IMAGE=$$(cat $(srcdir)/envoy-build-image.txt)
base-envoy.docker.stamp.DOCKER_DIR = $(topsrcdir)/bin_linux_amd64

#
# Envoy generate

generate: pkg/api/envoy
generate: api/envoy

# The version numbers of `protoc` (in this Makefile),
# `protoc-gen-gogofast` (in go.mod), and `protoc-gen-validate` (in
# go.mod) are based on
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/Dockerfile.ci
# (since that commit most closely corresponds to our ENVOY_COMMIT).
# Additionally, the package names of those programs are mentioned in
# ./go/pin.go, so that `go mod tidy` won't make the go.mod file forget
# about them.

PROTOC_VERSION = 3.5.1
PROTOC_PLATFORM = $(patsubst darwin,osx,$(GOHOSTOS))-$(patsubst amd64,x86_64,$(patsubst 386,x86_32,$(GOHOSTARCH)))

$(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc: $(var.)PROTOC_VERSION $(var.)PROTOC_PLATFORM
	mkdir -p $(@D)
	set -o pipefail; curl --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip | bsdtar -x -f - -O bin/protoc > $@
	chmod 755 $@

$(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast: go.mod $(FLOCK)
	mkdir -p $(@D)
	$(FLOCK) go.mod go build -o $@ github.com/gogo/protobuf/protoc-gen-gogofast

$(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-validate: go.mod $(FLOCK)
	mkdir -p $(@D)
	$(FLOCK) go.mod go build -o $@ github.com/envoyproxy/protoc-gen-validate

api/envoy: $(srcdir)/envoy
	rsync --recursive --delete --delete-excluded --prune-empty-dirs --include='*/' --include='*.proto' --exclude='*' $</api/envoy/ $@

# Search path for .proto files
gomoddir = $(shell $(FLOCK) go.mod go list $1/... >/dev/null 2>/dev/null; $(FLOCK) go.mod go list -m -f='{{.Dir}}' $1)
# This list is based on 'imports=()' in https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/build/generate_protos.sh
# (since that commit most closely corresponds to our ENVOY_COMMIT).
#
# However, we make the following edits:
#  - "github.com/gogo/protobuf/protobuf" instead of "github.com/gogo/protobuf" (we add an
#    extra "/protobuf" at the end).  I have no idea why.  I have no idea how the
#    go-control-plane version works without the extra "/protobuf" at the end; it looks to
#    me like they would need it too.  It makes no sense.
#  - Mess with the paths under "istio.io/gogo-genproto", since in 929161c and ee07f27 they
#    moved the .proto files all around.  The reason this affects us and not
#    go-control-plane is that our newer Envoy needs googleapis'
#    "google/api/expr/v1alpha1/", which was added in 32e3935 (.pb.go files) and ee07f27
#    (.proto files).
imports += $(CURDIR)/api
imports += $(call gomoddir,github.com/envoyproxy/protoc-gen-validate)
imports += $(call gomoddir,github.com/gogo/protobuf)/protobuf
imports += $(call gomoddir,istio.io/gogo-genproto)/common-protos
imports += $(call gomoddir,istio.io/gogo-genproto)/common-protos/github.com/prometheus/client_model
imports += $(call gomoddir,istio.io/gogo-genproto)/common-protos/github.com/census-instrumentation/opencensus-proto/src

# Map from .proto files to Go package names
# This list is based on 'mappings=()' in https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/build/generate_protos.sh
# (since that commit most closely corresponds to our ENVOY_COMMIT).
#
# However, we make the following edits:
#  - Add an entry for "google/api/expr/v1alpha1/syntax.proto", which didn't exist yet in
#    the version that go-control-plane uses (see the comment around "imports" above).
mappings += gogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto
mappings += google/api/annotations.proto=istio.io/gogo-genproto/googleapis/google/api
mappings += google/api/expr/v1alpha1/syntax.proto=istio.io/gogo-genproto/googleapis/google/api/expr/v1alpha1
mappings += google/api/http.proto=istio.io/gogo-genproto/googleapis/google/api
mappings += google/protobuf/any.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/duration.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/empty.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/struct.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/timestamp.proto=github.com/gogo/protobuf/types
mappings += google/protobuf/wrappers.proto=github.com/gogo/protobuf/types
mappings += google/rpc/code.proto=istio.io/gogo-genproto/googleapis/google/rpc
mappings += google/rpc/error_details.proto=istio.io/gogo-genproto/googleapis/google/rpc
mappings += google/rpc/status.proto=istio.io/gogo-genproto/googleapis/google/rpc
mappings += metrics.proto=istio.io/gogo-genproto/prometheus
mappings += opencensus/proto/trace/v1/trace.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
mappings += opencensus/proto/trace/v1/trace_config.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
mappings += validate/validate.proto=github.com/envoyproxy/protoc-gen-validate/validate
mappings += $(shell find $(CURDIR)/api/envoy -type f -name '*.proto' | sed -E 's,^$(CURDIR)/api/((.*)/[^/]*),\1=github.com/datawire/ambassador/pkg/api/\2,')

_imports = $(call lazyonce,_imports,$(imports))
_mappings = $(call lazyonce,_mappings,$(mappings))
pkg/api/envoy: api/envoy $(FLOCK) $(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc $(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast $(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-validate $(var.)_imports $(var.)_mappings
	rm -rf $@ $(@D).envoy.tmp
	mkdir -p $(@D).envoy.tmp
# go-control-plane `make generate`
	@set -e; find $(CURDIR)/api/envoy -type f -name '*.proto' | sed 's,/[^/]*$$,,' | uniq | while read -r dir; do \
		echo "Generating $$dir"; \
		./$(topsrcdir)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
			$(addprefix --proto_path=,$(_imports))  \
			--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast --gogofast_out='$(call joinlist,$(COMMA),plugins=grpc $(addprefix M,$(_mappings))):$(@D).envoy.tmp' \
			--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-validate --validate_out='lang=gogo:$(@D).envoy.tmp' \
			"$$dir"/*.proto; \
	done
# go-control-plane `make generate-patch`
# https://github.com/envoyproxy/go-control-plane/issues/173
	find $(@D).envoy.tmp -name '*.validate.go' -exec sed -E -i.bak 's,"(envoy/.*)"$$,"github.com/datawire/ambassador/pkg/api/\1",' {} +
	find $(@D).envoy.tmp -name '*.bak' -delete
# move things in to place
	mkdir -p $(@D)
	mv $(@D).envoy.tmp/envoy $@
	rmdir $(@D).envoy.tmp

#
# Envoy clean

clean: _clean-envoy
clobber: _clobber-envoy
generate-clean: _generate-clean-envoy

_clean-envoy: _clean-envoy-old
_clean-envoy: $(srcdir)/envoy-build-container.txt.clean
	rm -f $(srcdir)/envoy-build-image.txt
	rm -rf $(topsrcdir)/pkg/api.envoy.tmp/
_clobber-envoy: _clean-envoy
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(srcdir)/envoy)
_generate-clean-envoy: _clobber-envoy
	rm -rf $(topsrcdir)/api/envoy
.PHONY: _clean-envoy _clobber-envoy _generate-clean-envoy

# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
_clean-envoy-old:
# 2019-10-11
_clean-envoy-old: $(topsrcdir)/envoy-build-container.txt.clean
# 2019-10-11
	rm -rf $(topsrcdir)/envoy-bin
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(topsrcdir)/envoy-src)
	rm -f $(topsrcdir)/envoy-build-image.txt
# older than that
	$(if $(filter-out -,$(ENVOY_COMMIT)),rm -rf $(topsrcdir)/envoy)
.PHONY: _clean-envoy-old
