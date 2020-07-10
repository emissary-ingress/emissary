generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/                  -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.validate.go                , $(shell find $(OSS_HOME)/api/envoy/            -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.json.go                    , $(shell find $(OSS_HOME)/api/getambassador.io/ -name '*.proto' -not -name '*_nojson.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/getambassador.io/%.proto,  $(OSS_HOME)/python/ambassador/proto/%_pb2.py        , $(shell find $(OSS_HOME)/api/getambassador.io/ -name '*.proto' -not -name '*_nojson.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_pb.js          , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_grpc_web_pb.js , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(OSS_HOME)/pkg/envoy-control-plane
generate: ## Update generated sources that get committed to git
generate:
	$(MAKE) generate-clean
	$(MAKE) $(OSS_HOME)/api/envoy
	$(MAKE) _generate
_generate:
	@echo '$(MAKE) $$(generate/files)'; $(MAKE) $(generate/files)
generate-clean: ## Delete generated sources that get committed to git
generate-clean:
	rm -rf $(OSS_HOME)/api/envoy
	rm -rf $(OSS_HOME)/pkg/api $(OSS_HOME)/python/ambassador/proto
	rm -f $(OSS_HOME)/tools/sandbox/grpc_web/*_pb.js
	rm -rf $(OSS_HOME)/pkg/envoy-control-plane
.PHONY: generate _generate generate-clean

go-mod-tidy/oss:
	rm -f $(OSS_HOME)/go.sum
	cd $(OSS_HOME) && go mod tidy
	cd $(OSS_HOME) && go mod edit -require=$$(go list -m github.com/cncf/udpa/go | sed 's,/go ,@,')
	cd $(OSS_HOME) && go mod vendor # adds "// indirect" to the udpa line
	$(MAKE) go-mod-tidy/oss-evaluate
go-mod-tidy/oss-evaluate:
	@echo '# evaluate $$(proto_path)'; # $(proto_path) # cause Make to call `go list STUFF`, which will maybe edit go.mod or go.sum
go-mod-tidy: go-mod-tidy/oss
.PHONY: go-mod-tidy/oss go-mod-tidy

#
# Helper Make functions and variables

# Usage: VAR = $(call lazyonce,VAR,EXPR)
#
# Caches the value of EXPR (in case it's expensive/slow) once it is
# evaluated, but doesn't eager-evaluate it either.
lazyonce = $(eval $(strip $1) := $2)$2

# Usage: $(call joinlist,SEPARATOR,LIST)
# Example: $(call joinlist,/,foo bar baz) => foo/bar/baz
joinlist=$(if $(word 2,$2),$(firstword $2)$1$(call joinlist,$1,$(wordlist 2,$(words $2),$2)),$2)

comma=,

gomoddir = $(shell cd $(OSS_HOME); go list $1/... >/dev/null 2>/dev/null; go list -m -f='{{.Dir}}' $1)

#
# Tools we need to install for `make generate`

clobber: _makefile_clobber
_makefile_clobber:
	rm -rf $(OSS_HOME)/bin_*/
.PHONY: _makefile_clobber

GOHOSTOS=$(call lazyonce,GOHOSTOS,$(shell go env GOHOSTOS))
GOHOSTARCH=$(call lazyonce,GOHOSTARCH,$(shell go env GOHOSTARCH))

# PROTOC_VERSION is based on
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/Dockerfile.ci
# (since that commit most closely corresponds to our ENVOY_COMMIT).  That file says 3.6.1, so we're
# going to try to be as close as that to possible; but go ahead and upgrade to 3.8.0, which is the
# closest version that contains the fix so that it doesn't generate invalid Python if you name an
# Enum member the same as a Python keyword.
PROTOC_VERSION            = 3.8.0
PROTOC_PLATFORM           = $(patsubst darwin,osx,$(GOHOSTOS))-$(patsubst amd64,x86_64,$(patsubst 386,x86_32,$(GOHOSTARCH)))
tools/protoc              = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc
$(tools/protoc):
	mkdir -p $(@D)
	set -o pipefail; curl --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip | bsdtar -x -f - -O bin/protoc > $@
	chmod 755 $@

# The version number of protoc-gen-gogofast is controlled by `./go.mod`, and is based on
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/Dockerfile.ci
# (since that commit most closely corresponds to our ENVOY_COMMIT).  Additionally, the package name
# is mentioned in `./pkg/ignore/pin.go`, so that `go mod tidy` won't make the `go.mod` file forget
# about it.
tools/protoc-gen-gogofast = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast
$(tools/protoc-gen-gogofast): $(OSS_HOME)/go.mod
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/gogo/protobuf/protoc-gen-gogofast

# The version number of protoc-gen-validate is controlled by `./go.mod`, and is based on
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/Dockerfile.ci
# (since that commit most closely corresponds to our ENVOY_COMMIT).  Additionally, the package name
# is mentioned in `./pkg/ignore/pin.go`, so that `go mod tidy` won't make the `go.mod` file forget
# about it.
tools/protoc-gen-validate = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-validate
$(tools/protoc-gen-validate): $(OSS_HOME)/go.mod
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/envoyproxy/protoc-gen-validate

GRPC_WEB_VERSION          = 1.0.3
GRPC_WEB_PLATFORM         = $(GOHOSTOS)-x86_64
tools/protoc-gen-grpc-web = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web
$(tools/protoc-gen-grpc-web):
	mkdir -p $(@D)
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@

# The version number of protoc-gen-validate is controlled by `./go.mod`.  Additionally, the package
# name is mentioned in `./pkg/ignore/pin.go`, so that `go mod tidy` won't make the `go.mod` file
# forget about it.
tools/protoc-gen-go-json = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-go-json
$(tools/protoc-gen-go-json): $(OSS_HOME)/go.mod
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/mitchellh/protoc-gen-go-json

#
# `make generate` vendor rules

# TODO(lukeshu): Figure out a sane way of selecting an appropriate ENVOY_GO_CONTROL_PLANE_COMMIT for
# our version of Envoy.
#
# I was tempted to use "v0.9.0^" because it's the last version that used gogo/protobuf instead of
# golang/protobuf.  However, because the Envoy 1.11 -> 1.12 upgrade included
# https://github.com/envoyproxy/envoy/pull/8163 continuing to use the gogo/protobuf-based version is
# very difficult.  To the point that using the golang/protobuf version and editing it to work with
# gogo/protobuf is easier than getting the gogo/protobuf version to work with the newer proto files.
#
# Also, note that we disable all calls to SetDeterministic since it's totally broken in gogo/protobuf
# 1.3.0 and 1.3.1 (the latest version at the time of this writing), because gogo cherry-picked
# https://github.com/golang/protobuf/pull/650 and https://github.com/golang/protobuf/pull/656 but
# not https://github.com/golang/protobuf/pull/658 ; and is even more broken than it was in pre-#658
# golang/protobuf because protoc-gen-gogofast always generates a `Marshal` method, meaning that it
# is 100% impossible to use SetDeterministic with gogofast.

ENVOY_GO_CONTROL_PLANE_COMMIT = 3a8210324ccf55ef9fd7eeeed6fd24d59d6aefd9
$(OSS_HOME)/pkg/envoy-control-plane: FORCE
	rm -rf $@
	@PS4=; set -ex; { \
	  unset GIT_DIR GIT_WORK_TREE; \
	  tmpdir=$$(mktemp -d); \
	  trap 'rm -rf "$$tmpdir"' EXIT; \
	  cd "$$tmpdir"; \
	  git init .; \
	  git remote add origin https://github.com/envoyproxy/go-control-plane; \
	  git fetch --tags --all; \
	  git checkout $(ENVOY_GO_CONTROL_PLANE_COMMIT); \
	  find pkg -name '*.go' -exec sed -E -i.bak \
	    -e 's,github\.com/envoyproxy/go-control-plane/pkg,github.com/datawire/ambassador/pkg/envoy-control-plane,g' \
	    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/datawire/ambassador/pkg/api/envoy,g' \
	    -e 's,^[[:space:]]*"github.com/datawire/ambassador/pkg/api/[^"]*/([^/"]*)",\1 &,' \
	    \
	    -e 's,^[[:space:]]*"github\.com/golang/protobuf/ptypes",ptypes "github.com/gogo/protobuf/types",g' \
	    -e 's,^[[:space:]]*"github\.com/golang/protobuf/ptypes/any",any "github.com/gogo/protobuf/types",g' \
	    -e 's,^[[:space:]]*"github\.com/golang/protobuf/ptypes/struct",struct "github.com/gogo/protobuf/types",g' \
	    -e 's,"github\.com/golang/protobuf/ptypes(/any|/struct)?","github.com/gogo/protobuf/types",g' \
	    -e 's,github\.com/golang/protobuf/,github.com/gogo/protobuf/,g' \
	    -e '/SetDeterministic/d' \
	    -- {} +; \
	  find pkg -name '*.bak' -delete; \
	  mv $$(git ls-files ':[A-Z]*' ':!Dockerfile*' ':!Makefile') pkg; \
	  mv pkg $(abspath $@); \
	}
	cd $(OSS_HOME) && go fmt ./pkg/envoy-control-plane/...

#
# `make generate` protobuf rules

# TODO(lukeshu): Bring this in-line with
#   https://github.com/envoyproxy/envoy/pull/8155 /
#   https://github.com/envoyproxy/go-control-plane/pull/226
# instead of the old
#   https://github.com/envoyproxy/go-control-plane/blob/v0.9.0%5E/build/generate_protos.sh

# This proto_path list is largely based on 'imports=()' in
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/build/generate_protos.sh
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
#  - We use the $(call gomoddir) trick instead of the `go mod vendor` trick, because (1) the vendor
#    trick only works if the `.proto` and the `.go` live in the same directory together.  This is no
#    longer true of istio.io/gogo-genproto, and because (2) the vendor trick assumes that everything
#    we need vendored is mentioned in non-generates sources, which doesn't seem to be true for us.
#
# ... except now all that info lives in various Bazel BUILD files in envoy.git.  IDK what to tell
# you; if `make generate && go build ./pkg/api/...` breaks, blindly grub about in envoy.git/api/ and
# hope you figure out something that seems reasonable.
_proto_path += $(OSS_HOME)/api
_proto_path += $(OSS_HOME)/vendor
_proto_path += $(call gomoddir,github.com/envoyproxy/protoc-gen-validate)
_proto_path += $(call gomoddir,github.com/gogo/protobuf)/protobuf
_proto_path += $(call gomoddir,istio.io/gogo-genproto)/common-protos
_proto_path += $(call gomoddir,istio.io/gogo-genproto)/common-protos/github.com/prometheus/client_model
_proto_path += $(call gomoddir,istio.io/gogo-genproto)/common-protos/github.com/census-instrumentation/opencensus-proto/src
_proto_path += $(call gomoddir,github.com/cncf/udpa)
proto_path = $(call lazyonce,proto_path,$(_proto_path))

# Usage: $(call protoc,output_module,output_basedir[,plugin_files])
protoc = @echo PROTOC --$1_out=$2 $<; mkdir -p $2 && $(tools/protoc) \
  $(addprefix --proto_path=,$(proto_path)) \
  $(addprefix --plugin=,$3) \
  --$1_out=$(if $(proto_options/$(strip $1)),$(call joinlist,$(comma),$(proto_options/$(strip $1))):)$2 \
  $<

# The "M{FOO}={BAR}" options map from .proto files to Go package names.  This list of mappings is
# largely based on 'mappings=()' in
# https://github.com/envoyproxy/go-control-plane/blob/0e75602d5e36e96eafbe053999c0569edec9fe07/build/generate_protos.sh
# (since that commit most closely corresponds to our ENVOY_COMMIT).
#
# However, we make the following edits:
#  - Add an entry for "google/api/expr/v1alpha1/syntax.proto", which didn't exist yet in the version
#    that go-control-plane uses (see the comment around "proto_path" above).
#
# ... except now all that info lives in various Bazel BUILD files in envoy.git.  IDK what to tell
# you; if `make generate && go build ./pkg/api/...` breaks, blindly grub about in envoy.git/api/ and
# hope you figure out something that seems reasonable.
_proto_options/gogofast += plugins=grpc
_proto_options/gogofast += Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto
_proto_options/gogofast += Mgoogle/api/annotations.proto=istio.io/gogo-genproto/googleapis/google/api
_proto_options/gogofast += Mgoogle/api/expr/v1alpha1/syntax.proto=istio.io/gogo-genproto/googleapis/google/api/expr/v1alpha1
_proto_options/gogofast += Mgoogle/api/http.proto=istio.io/gogo-genproto/googleapis/google/api
_proto_options/gogofast += Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor
_proto_options/gogofast += Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types
_proto_options/gogofast += Mgoogle/rpc/code.proto=istio.io/gogo-genproto/googleapis/google/rpc
_proto_options/gogofast += Mgoogle/rpc/error_details.proto=istio.io/gogo-genproto/googleapis/google/rpc
_proto_options/gogofast += Mgoogle/rpc/status.proto=istio.io/gogo-genproto/googleapis/google/rpc
_proto_options/gogofast += Mmetrics.proto=istio.io/gogo-genproto/prometheus
_proto_options/gogofast += Mopencensus/proto/trace/v1/trace.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
_proto_options/gogofast += Mopencensus/proto/trace/v1/trace_config.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1
_proto_options/gogofast += Mvalidate/validate.proto=github.com/envoyproxy/protoc-gen-validate/validate
_proto_options/gogofast += Mudpa/annotations/migrate.proto=github.com/cncf/udpa/go/udpa/annotations
_proto_options/gogofast += Mudpa/annotations/sensitive.proto=github.com/cncf/udpa/go/udpa/annotations
_proto_options/gogofast += Mudpa/annotations/status.proto=github.com/cncf/udpa/go/udpa/annotations
_proto_options/gogofast += Mudpa/annotations/versioning.proto=github.com/cncf/udpa/go/udpa/annotations
_proto_options/gogofast += $(shell find $(OSS_HOME)/api/envoy -type f -name '*.proto' | sed -E 's,^$(OSS_HOME)/api/((.*)/[^/]*),M\1=github.com/datawire/ambassador/pkg/api/\2,')
proto_options/gogofast = $(call lazyonce,proto_options/gogofast,$(_proto_options/gogofast))
$(OSS_HOME)/pkg/api/%.pb.go: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-gogofast) | $(OSS_HOME)/vendor
	$(call protoc,gogofast,$(OSS_HOME)/pkg/api,\
	    $(tools/protoc-gen-gogofast))

proto_options/validate += lang=gogo
$(OSS_HOME)/pkg/api/%.pb.validate.go: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-validate) | $(OSS_HOME)/vendor
	$(call protoc,validate,$(OSS_HOME)/pkg/api,\
	    $(tools/protoc-gen-validate))
	sed -E -i.bak 's,"(envoy/.*)"$$,"github.com/datawire/ambassador/pkg/api/\1",' $@
	rm -f $@.bak

proto_options/go-json +=
$(OSS_HOME)/pkg/api/%.pb.json.go: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-go-json) | $(OSS_HOME)/vendor
	$(call protoc,go-json,$(OSS_HOME)/pkg/api,\
	    $(tools/protoc-gen-go-json))
	sed -E -i.bak 's,golang/protobuf,gogo/protobuf,g' $@
	rm -f $@.bak

proto_options/python +=
$(OSS_HOME)/generate.tmp/%_pb2.py: $(OSS_HOME)/api/%.proto $(tools/protoc) | $(OSS_HOME)/vendor
	mkdir -p $(OSS_HOME)/generate.tmp/getambassador.io
	mkdir -p $(OSS_HOME)/generate.tmp/getambassador
	ln -sf ../getambassador.io/ $(OSS_HOME)/generate.tmp/getambassador/io
	$(call protoc,python,$(OSS_HOME)/generate.tmp)

proto_options/js += import_style=commonjs
$(OSS_HOME)/generate.tmp/%_pb.js: $(OSS_HOME)/api/%.proto $(tools/protoc) | $(OSS_HOME)/vendor
	$(call protoc,js,$(OSS_HOME)/generate.tmp)

proto_options/grpc-web += import_style=commonjs
proto_options/grpc-web += mode=grpcwebtext
$(OSS_HOME)/generate.tmp/%_grpc_web_pb.js: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-grpc-web) | $(OSS_HOME)/vendor
	$(call protoc,grpc-web,$(OSS_HOME)/generate.tmp,\
	    $(tools/protoc-gen-grpc-web))

# This madness with sed is because protoc likes to insert broken imports when generating
# Python code, and my attempts to sort out how to fix the protoc invocation are taking 
# longer than I have right now.
# (Previous we just did cp $< $@ instead of the sed call.)

$(OSS_HOME)/python/ambassador/proto/%.py: $(OSS_HOME)/generate.tmp/getambassador.io/%.py
	mkdir -p $(@D)
	sed \
		-e 's/github_dot_com_dot_gogo_dot_protobuf_dot_gogoproto_dot_gogo__pb2.DESCRIPTOR,//' \
		-e '/from github.com.gogo.protobuf.gogoproto import/d' \
		< $< > $@

$(OSS_HOME)/tools/sandbox/grpc_web/%.js: $(OSS_HOME)/generate.tmp/kat/%.js
	cp $< $@

$(OSS_HOME)/vendor: FORCE
	set -e; { \
	  cd $(@D); \
	  GO111MODULE=off go list -f='{{ range .Imports }}{{ . }}{{ "\n" }}{{ end }}' ./... | \
	    sort -u | \
	    sed -E -n 's,^github\.com/datawire/ambassador/pkg/(api|envoy-control-plane),pkg/\1,p' | \
	      while read -r dir; do \
	        mkdir -p "$$dir"; \
	        echo "$$dir" | sed 's,.*/,package ,' > "$${dir}/vendor_bootstrap_hack.go"; \
	     done; \
	}
	cp -a $(@D)/go.mod $(@D)/go.mod.vendor-hack.bak
	cd $(@D) && go mod vendor
	find $(@D) -name vendor_bootstrap_hack.go -delete
	mv -f $(@D)/go.mod.vendor-hack.bak $(@D)/go.mod

clean: _makefile_clean
_makefile_clean:
	rm -rf $(OSS_HOME)/generate.tmp $(OSS_HOME)/vendor
.PHONY: _makefile_clean
