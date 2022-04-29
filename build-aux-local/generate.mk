crds_yaml_dir = $(OSS_HOME)/charts/ambassador/crds

generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/agent/            -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/edgectl/          -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/getambassador.io/%.proto,  $(OSS_HOME)/python/ambassador/proto/%_pb2.py        , $(shell find $(OSS_HOME)/api/getambassador.io/ -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_pb.js          , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(patsubst $(OSS_HOME)/api/kat/%.proto,               $(OSS_HOME)/tools/sandbox/grpc_web/%_grpc_web_pb.js , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto'))
generate/files += $(OSS_HOME)/api/envoy               # recipe in _cxx/envoy.mk
generate/files += $(OSS_HOME)/api/pb                  # recipe in _cxx/envoy.mk
generate/files += $(OSS_HOME)/pkg/api/envoy           # recipe in _cxx/envoy.mk
generate/files += $(OSS_HOME)/pkg/api/pb              # recipe in _cxx/envoy.mk
generate/files += $(OSS_HOME)/pkg/envoy-control-plane # recipe in _cxx/envoy.mk
generate/files += $(OSS_HOME)/docker/test-ratelimit/ratelimit.proto
generate/files += $(OSS_HOME)/OPENSOURCE.md
generate/files += $(OSS_HOME)/builder/requirements.txt
generate/files += $(OSS_HOME)/CHANGELOG.md

generate: ## Update generated sources that get committed to git
generate:
	$(MAKE) generate-clean
	$(MAKE) $(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb
	$(MAKE) _generate
	cd .circleci && ./generate --always-make
_generate:
	@echo '$(MAKE) $$(generate/files)'; $(MAKE) $(generate/files)
generate-clean: ## Delete generated sources that get committed to git
generate-clean:
	rm -rf $(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb
	rm -rf $(OSS_HOME)/pkg/api/envoy $(OSS_HOME)/pkg/api/pb
	rm -rf $(OSS_HOME)/_cxx/envoy/build_go
	rm -rf $(OSS_HOME)/pkg/api/kat
	rm -f $(OSS_HOME)/pkg/api/agent/*.pb.go
	rm -f $(OSS_HOME)/pkg/api/edgectl/rpc/*.pb.go
	rm -rf $(OSS_HOME)/python/ambassador/proto
	rm -f $(OSS_HOME)/tools/sandbox/grpc_web/*_pb.js
	rm -rf $(OSS_HOME)/pkg/envoy-control-plane
	rm -f $(OSS_HOME)/docker/test-ratelimit/ratelimit.proto
	rm -f $(OSS_HOME)/OPENSOURCE.md
.PHONY: generate _generate generate-clean

go-mod-tidy/oss:
	rm -f $(OSS_HOME)/go.sum
	cd $(OSS_HOME) && GOFLAGS=-mod=mod go mod tidy
	cd $(OSS_HOME) && GOFLAGS=-mod=mod go mod vendor # make sure go.mod is complete, and re-gen go.sum
	$(MAKE) go-mod-tidy/oss-evaluate
go-mod-tidy/oss-evaluate:
	@echo '# evaluate $$(proto_path)'; # $(proto_path) # cause Make to call `go list STUFF`, which will maybe edit go.mod or go.sum
go-mod-tidy: go-mod-tidy/oss
.PHONY: go-mod-tidy/oss go-mod-tidy

$(OSS_HOME)/CHANGELOG.md: $(OSS_HOME)/docs/CHANGELOG.tpl $(OSS_HOME)/docs/releaseNotes.yml
	docker run --rm \
	  -v $(OSS_HOME)/docs/CHANGELOG.tpl:/tmp/CHANGELOG.tpl \
	  -v $(OSS_HOME)/docs/releaseNotes.yml:/tmp/releaseNotes.yml \
	  hairyhenderson/gomplate --verbose --file /tmp/CHANGELOG.tpl --datasource relnotes=/tmp/releaseNotes.yml > CHANGELOG.md

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

#
# `make generate` protobuf rules

$(OSS_HOME)/docker/test-ratelimit/ratelimit.proto:
	set -e; { \
	  url=https://raw.githubusercontent.com/envoyproxy/ratelimit/v1.3.0/proto/ratelimit/ratelimit.proto; \
	  echo "// Downloaded from $$url"; \
	  echo; \
	  curl --fail -L "$$url"; \
	} > $@

# Usage: $(call protoc,output_module,output_basedir[,plugin_files])
#
# Using the $(call protoc,...) macro will execute the `protoc` program
# to generate the single output file $@ from $< using the
# 'output_module' argument.
#
# Nomenclature:
#   The `protoc` program uses "plugins" that add support for new "output
#   modules" to the protoc.
#
# Arguments:
#   - output_module: The protoc module to run.
#   - output_basedir: Where the protobuf "namespace" starts; such that
#     $@ is "{output_basedir}/{protobuf_packagename}/{filename}"
#   - plugin_files: A whitespace-separated list of plugin files to
#     load (necessary if output_module isn't built-in to protoc)
#
# Configuration:
#   This macro takes most of its configuration from global variables:
#
#    - proto_path: A whitespace-separated list of directories to look
#      for .proto files in.  Input files must be within this path.
#    - proto_options/$(output_module): A whitespace-separated list of
#      configuration options specific to this output module.
#
#   Having these as global variables instead of arguments makes it a
#   lot easier to wrangle having large tables of options that some
#   modules require.
#
# Example:
#
#    The Make snippet
#
#        proto_path  = $(CURDIR)/input_dir
#        proto_path += $(CURDIR)/vendor/lib
#        proto_options/example  = key1=val1
#        proto_options/example += key2=val2
#
#        $(CURDIR)/output_dir/mypkg/myfile.pb.example: $(CURDIR)/input_dir/mypkg/myfile.proto /usr/bin/protoc-gen-example
#                $(call protoc,example,$(CURDIR)/output_dir,\
#                    /usr/bin/protoc-gen-example)
#
#    would run the command
#
#        $(tools/protoc) \
#            --proto_path=$(CURDIR)/input_dir,$(CURDIR)/vendor/lib \
#            --plugin=/usr/bin/protoc-gen-example \
#            --example_out=key1=val1,key2=val2:$(CURDIR)/output_dir
protoc = @echo PROTOC --$1_out=$2 $<; mkdir -p $2 && $(tools/protoc) \
  $(addprefix --proto_path=,$(proto_path)) \
  $(addprefix --plugin=,$3) \
  --$1_out=$(if $(proto_options/$(strip $1)),$(call joinlist,$(comma),$(proto_options/$(strip $1))):)$2 \
  $<

# proto_path is a list of where to look for .proto files.
proto_path += $(OSS_HOME)/api # input files must be within the path
proto_path += $(OSS_HOME)/vendor # for "k8s.io/..."

# The "M{FOO}={BAR}" options map from .proto files to Go package names.
proto_options/go += plugins=grpc
#proto_options/go += Mgoogle/protobuf/duration.proto=github.com/golang/protobuf/ptypes/duration
$(OSS_HOME)/pkg/api/%.pb.go: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-go)
	$(call protoc,go,$(OSS_HOME)/pkg/api,\
	    $(tools/protoc-gen-go))

proto_options/python +=
$(OSS_HOME)/generate.tmp/%_pb2.py: $(OSS_HOME)/api/%.proto $(tools/protoc)
	mkdir -p $(OSS_HOME)/generate.tmp/getambassador.io
	mkdir -p $(OSS_HOME)/generate.tmp/getambassador
	ln -sf ../getambassador.io/ $(OSS_HOME)/generate.tmp/getambassador/io
	$(call protoc,python,$(OSS_HOME)/generate.tmp)

proto_options/js += import_style=commonjs
$(OSS_HOME)/generate.tmp/%_pb.js: $(OSS_HOME)/api/%.proto $(tools/protoc)
	$(call protoc,js,$(OSS_HOME)/generate.tmp)

proto_options/grpc-web += import_style=commonjs
proto_options/grpc-web += mode=grpcwebtext
$(OSS_HOME)/generate.tmp/%_grpc_web_pb.js: $(OSS_HOME)/api/%.proto $(tools/protoc) $(tools/protoc-gen-grpc-web)
	$(call protoc,grpc-web,$(OSS_HOME)/generate.tmp,\
	    $(tools/protoc-gen-grpc-web))

$(OSS_HOME)/python/ambassador/proto/%.py: $(OSS_HOME)/generate.tmp/getambassador.io/%.py
	mkdir -p $(@D)
	cp $< $@

$(OSS_HOME)/tools/sandbox/grpc_web/%.js: $(OSS_HOME)/generate.tmp/kat/%.js
	cp $< $@

clean: _makefile_clean
_makefile_clean:
	rm -rf $(OSS_HOME)/generate.tmp
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
	  $(foreach varname,$(sort $(filter controller-gen/options/%,$(.VARIABLES))), $(patsubst controller-gen/options/%,%,$(varname))$(if $(strip $($(varname))),:$(call joinlist,$(comma),$($(varname)))) ) \
	  $(foreach varname,$(sort $(filter controller-gen/output/%,$(.VARIABLES))), $(call joinlist,:,output $(patsubst controller-gen/output/%,%,$(varname)) $($(varname))) ) \
	  paths="./pkg/api/getambassador.io/..."
	@PS4=; set -ex; for file in $(crds_yaml_dir)/getambassador.io_*.yaml; do $(tools/fix-crds) helm 1.11 "$$file" > "$$file.tmp"; mv "$$file.tmp" "$$file"; done
.PHONY: _generate_controller_gen

$(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml: $(OSS_HOME)/manifests/ambassador/ambassador-crds.yaml
	@printf '  $(CYN)$@$(END)\n'
	cp $(OSS_HOME)/manifests/ambassador/ambassador-crds.yaml $@

$(OSS_HOME)/manifests/ambassador/ambassador-crds.yaml: _generate_controller_gen $(tools/fix-crds) update-yaml-preflight
	@printf '  $(CYN)$@$(END)\n'
	$(tools/fix-crds) oss 1.11 $(sort $(wildcard $(crds_yaml_dir)/getambassador.io_*.yaml)) > $@

$(OSS_HOME)/docs/yaml/ambassador/%.yaml: $(OSS_HOME)/docs/yaml/ambassador/%.yaml.m4 $(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml update-yaml-preflight
	@printf '  $(CYN)$@$(END)\n'
	cd $(@D) && m4 < $(<F) > $(@F)

update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-crds.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-rbac-prometheus.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador/ambassador-rbac.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/oss-migration.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/resources-migration.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/projects.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/aes.yaml
update-yaml/files += $(OSS_HOME)/docs/yaml/ambassador-agent.yaml
update-yaml/files += $(OSS_HOME)/manifests/ambassador/ambassador-crds.yaml
update-yaml/files += $(OSS_HOME)/manifests/ambassador/ambassador.yaml

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

$(OSS_HOME)/OPENSOURCE.md: $(tools/go-mkopensource) $(tools/py-mkopensource) $(OSS_HOME)/build-aux-local/go-version.txt $(OSS_HOME)/build-aux-local/pip-show.txt
	$(MAKE) $(OSS_HOME)/build-aux/go$$(cat $(OSS_HOME)/build-aux-local/go-version.txt).src.tar.gz
	set -e; { \
		cd $(OSS_HOME); \
		$(tools/go-mkopensource) --output-format=txt --package=mod --gotar=build-aux/go$$(cat $(OSS_HOME)/build-aux-local/go-version.txt).src.tar.gz; \
		echo; \
		{ sed 's/^---$$//' $(OSS_HOME)/build-aux-local/pip-show.txt; echo; } | $(tools/py-mkopensource); \
	} > $@

python-setup: create-venv
	$(OSS_HOME)/venv/bin/python -m pip install ruamel.yaml
.PHONY: python-setup

define generate_yaml_from_helm
	mkdir -p $(OSS_HOME)/build/yaml/$(1) && \
		helm template ambassador -n $(2) \
		-f $(OSS_HOME)/k8s-config/$(1)/values.yaml \
		$(OSS_HOME)/charts/ambassador > $(OSS_HOME)/build/yaml/$(1)/helm-expanded.yaml
	$(OSS_HOME)/venv/bin/python $(OSS_HOME)/k8s-config/create_yaml.py \
		$(OSS_HOME)/build/yaml/$(1)/helm-expanded.yaml $(OSS_HOME)/k8s-config/$(1)/require.yaml > $(3)
endef

$(OSS_HOME)/docs/yaml/ambassador/ambassador-rbac.yaml: $(OSS_HOME)/manifests/ambassador/ambassador.yaml
	@printf '  $(CYN)$@$(END)\n'
	cp $(OSS_HOME)/manifests/ambassador/ambassador.yaml $@

$(OSS_HOME)/manifests/ambassador/ambassador.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/ambassador-rbac/require.yaml $(OSS_HOME)/k8s-config/ambassador-rbac/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,ambassador-rbac,default,$@)

$(OSS_HOME)/docs/yaml/oss-migration.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/oss-migration/require.yaml $(OSS_HOME)/k8s-config/oss-migration/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,oss-migration,default,$@)

$(OSS_HOME)/docs/yaml/resources-migration.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/resources-migration/require.yaml $(OSS_HOME)/k8s-config/resources-migration/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,resources-migration,default,$@)

$(OSS_HOME)/docs/yaml/projects.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/projects/require.yaml $(OSS_HOME)/k8s-config/projects/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,projects,ambassador,$@)

$(OSS_HOME)/docs/yaml/aes.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/aes/require.yaml $(OSS_HOME)/k8s-config/aes/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,aes,ambassador,$@)

$(OSS_HOME)/docs/yaml/ambassador-agent.yaml: $(OSS_HOME)/k8s-config/create_yaml.py $(OSS_HOME)/k8s-config/ambassador-agent/require.yaml $(OSS_HOME)/k8s-config/ambassador-agent/values.yaml $(OSS_HOME)/charts/ambassador/templates/*.yaml $(OSS_HOME)/charts/ambassador/values.yaml python-setup
	@printf '  $(CYN)$@$(END)\n'
	$(call generate_yaml_from_helm,ambassador-agent,ambassador,$@)
