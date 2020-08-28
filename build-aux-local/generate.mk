crds_yaml_dir = $(OSS_HOME)/../ambassador-chart/crds

generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/agent/            -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/getambassador.io/%.proto,  $(OSS_HOME)/python/ambassador/proto/%_pb2.py        , $(shell find $(OSS_HOME)/api/getambassador.io/ -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_pb.js          , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_grpc_web_pb.js , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(OSS_HOME)/pkg/api/envoy
generate/files += $(OSS_HOME)/pkg/api/pb
generate/files += $(OSS_HOME)/pkg/envoy-control-plane
generate/files += $(OSS_HOME)/docker/test-ratelimit/ratelimit.proto
generate/files += $(OSS_HOME)/OPENSOURCE.md
generate/files += $(OSS_HOME)/builder/requirements.txt
generate: ## Update generated sources that get committed to git
generate:
	$(MAKE) generate-clean
	$(MAKE) $(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb
	$(MAKE) _generate
_generate:
	@echo '$(MAKE) $$(generate/files)'; $(MAKE) $(generate/files)
generate-clean: ## Delete generated sources that get committed to git
generate-clean:
	rm -rf $(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb
	rm -rf $(OSS_HOME)/pkg/api/envoy $(OSS_HOME)/pkg/api/pb
	rm -rf $(OSS_HOME)/_cxx/envoy/build_go
	rm -rf $(OSS_HOME)/pkg/api/kat
	rm -f $(OSS_HOME)/pkg/api/agent/*.pb.go
	rm -rf $(OSS_HOME)/python/ambassador/proto
	rm -f $(OSS_HOME)/tools/sandbox/grpc_web/*_pb.js
	rm -rf $(OSS_HOME)/pkg/envoy-control-plane
	rm -f $(OSS_HOME)/docker/test-ratelimit/ratelimit.proto
	rm -f $(OSS_HOME)/OPENSOURCE.md
.PHONY: generate _generate generate-clean

go-mod-tidy/oss:
	rm -f $(OSS_HOME)/go.sum
	cd $(OSS_HOME) && go mod tidy
	cd $(OSS_HOME) && go mod vendor # make sure go.mod's complete, re-gen go.sum
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

# PROTOC_VERSION must be at least 3.8.0 in order to contain the fix so that it doesn't generate
# invalid Python if you name an Enum member the same as a Python keyword.
PROTOC_VERSION            = 3.8.0
PROTOC_PLATFORM           = $(patsubst darwin,osx,$(GOHOSTOS))-$(patsubst amd64,x86_64,$(patsubst 386,x86_32,$(GOHOSTARCH)))
tools/protoc              = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/bin/protoc
$(tools/protoc): $(OSS_HOME)/build-aux-local/generate.mk
	mkdir -p $(dir $(@D))
	set -o pipefail; curl --fail -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_PLATFORM).zip | bsdtar -C $(dir $(@D)) -xf -
	chmod 755 $@

# The version number of protoc-gen-go is controlled by `./go.mod`.  Additionally, the package name is
# mentioned in `./pkg/ignore/pin.go`, so that `go mod tidy` won't make the `go.mod` file forget about
# it.
tools/protoc-gen-go = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-go
$(tools/protoc-gen-go): $(OSS_HOME)/go.mod
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/golang/protobuf/protoc-gen-go

GRPC_WEB_VERSION          = 1.0.3
GRPC_WEB_PLATFORM         = $(GOHOSTOS)-x86_64
tools/protoc-gen-grpc-web = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web
$(tools/protoc-gen-grpc-web): $(OSS_HOME)/build-aux-local/generate.mk
	mkdir -p $(@D)
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@

# The version number of protoc-gen-validate is controlled by `./go.mod`.  Additionally, the package
# name is mentioned in `./pkg/ignore/pin.go`, so that `go mod tidy` won't make the `go.mod` file
# forget about it.
tools/controller-gen = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/controller-gen
$(tools/controller-gen): $(OSS_HOME)/go.mod
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

tools/fix-crds = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/fix-crds
$(tools/fix-crds): FORCE
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/datawire/ambassador/cmd/fix-crds

tools/go-mkopensource = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/go-mkopensource
$(tools/go-mkopensource): FORCE
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/datawire/ambassador/cmd/go-mkopensource

tools/py-mkopensource = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/py-mkopensource
$(tools/py-mkopensource): FORCE
	mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ github.com/datawire/ambassador/cmd/py-mkopensource

#
# `make generate` vendor rules

# How to set ENVOY_GO_CONTROL_PLANE_COMMIT: In envoyproxy/go-control-plane.git, the majority of
# commits have a commit message of the form "Mirrored from envoyproxy/envoy @ ${envoy.git_commit}".
# Look for the most recent one that names a commit that is an ancestor of our ENVOY_COMMIT.  If there
# are commits not of that form immediately following that commit, you can take them in too (but that's
# pretty uncommon).  Since that's a simple sentence, can be tedious to go through and check which
# commits are ancestors, I added `make guess-envoy-go-control-plane-commit` to do that in an automated
# way!  Still look at the commit yourself to make sure it seems sane; blindly trusting machines is
# bad, mmkay?
ENVOY_GO_CONTROL_PLANE_COMMIT = v0.9.6

guess-envoy-go-control-plane-commit: $(OSS_HOME)/_cxx/envoy $(OSS_HOME)/_cxx/go-control-plane
	@echo
	@echo '######################################################################'
	@echo
	@set -e; { \
	  (cd $(OSS_HOME)/_cxx/go-control-plane && git log --format='%H %s' origin/master) | sed -n 's, Mirrored from envoyproxy/envoy @ , ,p' | \
	  while read -r go_commit cxx_commit; do \
	    if (cd $(OSS_HOME)/_cxx/envoy && git merge-base --is-ancestor "$$cxx_commit" $(ENVOY_COMMIT) 2>/dev/null); then \
	      echo "ENVOY_GO_CONTROL_PLANE_COMMIT = $$go_commit"; \
	      break; \
	    fi; \
	  done; \
	}
.PHONY: guess-envoy-go-control-plane-commit

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
	cd $(OSS_HOME) && go fmt ./pkg/envoy-control-plane/...

#
# `make generate` protobuf rules

$(OSS_HOME)/docker/test-ratelimit/ratelimit.proto:
	set -e; { \
	  url=https://raw.githubusercontent.com/envoyproxy/ratelimit/v1.3.0/proto/ratelimit/ratelimit.proto; \
	  echo "// Downloaded from $$url"; \
	  echo; \
	  curl --fail -L "$$url"; \
	} > $@

# proto_path is a list of where to look for .proto files.
_proto_path += $(OSS_HOME)/api # input files must be within the path
_proto_path += $(OSS_HOME)/vendor # for "k8s.io/..."
proto_path = $(call lazyonce,proto_path,$(_proto_path))

# Usage: $(call protoc,output_module,output_basedir[,plugin_files])
protoc = @echo PROTOC --$1_out=$2 $<; mkdir -p $2 && $(tools/protoc) \
  $(addprefix --proto_path=,$(proto_path)) \
  $(addprefix --plugin=,$3) \
  --$1_out=$(if $(proto_options/$(strip $1)),$(call joinlist,$(comma),$(proto_options/$(strip $1))):)$2 \
  $<

# The "M{FOO}={BAR}" options map from .proto files to Go package names.
_proto_options/go += plugins=grpc
#_proto_options/go += Mgoogle/protobuf/duration.proto=github.com/golang/protobuf/ptypes/duration
proto_options/go = $(call lazyonce,proto_options/go,$(_proto_options/go))
$(OSS_HOME)/pkg/api/%.pb.go: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-go) | $(OSS_HOME)/vendor
	$(call protoc,go,$(OSS_HOME)/pkg/api,\
	    $(tools/protoc-gen-go))

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

$(OSS_HOME)/python/ambassador/proto/%.py: $(OSS_HOME)/generate.tmp/getambassador.io/%.py
	mkdir -p $(@D)
	cp $< $@

$(OSS_HOME)/tools/sandbox/grpc_web/%.js: $(OSS_HOME)/generate.tmp/kat/%.js
	cp $< $@

$(OSS_HOME)/vendor: FORCE
	set -e; { \
	  cd $(@D); \
	  GOPATH=/bogus GO111MODULE=off go list -f='{{ range .Imports }}{{ . }}{{ "\n" }}{{ end }}' ./... | \
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

#
# `make generate`/`make update-yaml` rules to update generated YAML files (and `zz_generated.*.go` Go files)

update-yaml-preflight:
	@printf "$(CYN)==> $(GRN)Updating YAML$(END)\n"
.PHONY: update-yaml-preflight

# Use `controller-gen` to generate Go & YAML
#
# - Enable a generator by setting the
#   `controller-gen/options/GENERATOR_NAME` variable (even to an empty
#   value).
# - Setting `controller-gen/output/GENERATOR_NAME` for an enabled
#   generator is optional; the default output for each enabled
#   generator is `dir=config/GENERATOR_NAME`.
# - It is invalid to set `controller-gen/output/GENERATOR_NAME` for a
#   generator that is not enabled.
#
#controller-gen/options/webhook     +=
#controller-gen/options/schemapatch += manifests=foo
#controller-gen/options/rbac        += roleName=ambassador
controller-gen/options/object      += # headerFile=hack/boilerplate.go.txt
controller-gen/options/crd         += trivialVersions=true # change this to "false" once we're OK with requiring Kubernetes 1.13+
controller-gen/options/crd         += crdVersions=v1beta1 # change this to "v1" once we're OK with requiring Kubernetes 1.16+
controller-gen/output/crd           = dir=$(crds_yaml_dir)
_generate_controller_gen: $(tools/controller-gen) $(tools/fix-crds) update-yaml-preflight
	@printf '  $(CYN)Running controller-gen$(END)\n'
	rm -f $(crds_yaml_dir)/getambassador.io_*
	cd $(OSS_HOME) && $(tools/controller-gen) \
	  $(foreach varname,$(filter controller-gen/options/%,$(.VARIABLES)), $(patsubst controller-gen/options/%,%,$(varname))$(if $(strip $($(varname))),:$(call joinlist,$(comma),$($(varname)))) ) \
	  $(foreach varname,$(filter controller-gen/output/%,$(.VARIABLES)), $(call joinlist,:,output $(patsubst controller-gen/output/%,%,$(varname)) $($(varname))) ) \
	  paths="./pkg/api/getambassador.io/..."
	@PS4=; set -ex; for file in $(crds_yaml_dir)/getambassador.io_*.yaml; do $(tools/fix-crds) helm 1.11 "$$file" > "$$file.tmp"; mv "$$file.tmp" "$$file"; done
.PHONY: _generate_controller_gen

$(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml: _generate_controller_gen $(tools/fix-crds) update-yaml-preflight
	@printf '  $(CYN)$@$(END)\n'
	$(tools/fix-crds) oss 1.11 $(sort $(wildcard $(crds_yaml_dir)/*.yaml)) > $@
$(OSS_HOME)/python/tests/manifests/crds.yaml: $(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml $(tools/fix-crds) update-yaml-preflight
	@printf '  $(CYN)$@$(END)\n'
	$(tools/fix-crds) oss 1.10 $< > $@
$(OSS_HOME)/docs/yaml/ambassador/%.yaml: $(OSS_HOME)/docs/yaml/ambassador/%.yaml.m4 $(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml update-yaml-preflight
	@printf '  $(CYN)$@$(END)\n'
	cd $(@D) && m4 < $(<F) > $(@F)

update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml
update-yaml/files += $(OSS_HOME)/python/tests/manifests/crds.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-rbac-prometheus.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-knative.yaml

generate/files += $(update-yaml/files)
update-yaml:
	$(MAKE) update-yaml-clean
	@echo '$(MAKE) $$(update-yaml/files)'; $(MAKE) $(update-yaml/files)
.PHONY: update-yaml

update-yaml-clean:
	find $(OSS_HOME)/pkg/api/getambassador.io -name 'zz_generated.*.go' -delete
	rm -f $(crds_yaml_dir)/getambassador.io_*
	rm -f $(update-yaml/files)
generate-clean: update-yaml-clean
.PHONY: update-yaml-clean

#
# Generate report on dependencies

$(OSS_HOME)/build-aux-local/pip-show.txt: sync
	docker exec $$($(BUILDER)) sh -c 'pip freeze --exclude-editable | cut -d= -f1 | xargs pip show' > $@

$(OSS_HOME)/builder/requirements.txt: %.txt: %.in FORCE
	$(BUILDER) pip-compile
.PRECIOUS: $(OSS_HOME)/builder/requirements.txt

$(OSS_HOME)/build-aux-local/go-version.txt: $(OSS_HOME)/builder/Dockerfile.base
	sed -En 's,.*https://dl\.google\.com/go/go([0-9a-z.-]*)\.linux-amd64\.tar\.gz.*,\1,p' < $< > $@

$(OSS_HOME)/build-aux/go1%.src.tar.gz:
	curl -o $@ --fail -L https://dl.google.com/go/$(@F)

$(OSS_HOME)/OPENSOURCE.md: $(tools/go-mkopensource) $(tools/py-mkopensource) $(OSS_HOME)/build-aux-local/go-version.txt $(OSS_HOME)/build-aux-local/pip-show.txt $(OSS_HOME)/vendor
	$(MAKE) $(OSS_HOME)/build-aux/go$$(cat $(OSS_HOME)/build-aux-local/go-version.txt).src.tar.gz
	set -e; { \
		cd $(OSS_HOME); \
		$(tools/go-mkopensource) --output-format=txt --package=mod --gotar=build-aux/go$$(cat $(OSS_HOME)/build-aux-local/go-version.txt).src.tar.gz; \
		echo; \
		{ sed 's/^---$$//' $(OSS_HOME)/build-aux-local/pip-show.txt; echo; } | $(tools/py-mkopensource); \
	} > $@
