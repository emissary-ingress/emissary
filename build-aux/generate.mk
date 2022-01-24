# -*- fill-column: 102 -*-

# This file deals with creating files that get checked in to Git.  This is all grouped together in to
# one file, rather than being closer to the "subject matter" because this is a heinous thing.  Output
# files should not get checked in to Git -- every entry added to to this file is an affront to all
# that is good and proper.  As an exception, some of the Envoy-related stuff is allowed to live in
# envoy.mk, because that's a whole other bag of gross.

#
# `go mod tidy`
#
# This `go mod tidy` business only belongs in generate.mk because for the moment we're checking
# 'vendor/' in to Git.

go-mod-tidy:
.PHONY: go-mod-tidy

go-mod-tidy: go-mod-tidy/main
go-mod-tidy/main:
	rm -f go.sum
	GOFLAGS=-mod=mod go mod tidy
.PHONY: go-mod-tidy/main

#
# The main `make generate` entrypoints and listings

# - Let $(generate/files) be a listing of all files or directories that `make generate` will create.
#
# - Let $(generate-fast/files) be the subset of $(generate/files) that can be generated "quickly".  A
#   file may NOT be considered fast if it uses the builder container, if it uses the network, or if it
#   needs to access the filesystem to evaluate the list of files (as the lines using `$(shell find
#   ...)` do).
#
# - Let $(generate/precious) be the subset of $(generate/files) that should not be deleted prior to
#   re-generation.

# Initialize
generate-fast/files  =
generate/files       = $(generate-fast/files)
generate/precious    =
# Whole directories with rules for each individual file in it
generate/files      += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/kat/              -name '*.proto')) $(OSS_HOME)/pkg/api/kat/
generate/files      += $(patsubst $(OSS_HOME)/api/%.proto,                   $(OSS_HOME)/pkg/api/%.pb.go                         , $(shell find $(OSS_HOME)/api/agent/            -name '*.proto')) $(OSS_HOME)/pkg/api/agent/
# Whole directories with one rule for the whole directory
generate/files      += $(OSS_HOME)/api/envoy/
generate/files      += $(OSS_HOME)/api/pb/
generate/files      += $(OSS_HOME)/pkg/api/envoy/
generate/files      += $(OSS_HOME)/pkg/api/pb/
generate/files      += $(OSS_HOME)/pkg/envoy-control-plane/
# Individual files: Misc
generate/files      += $(OSS_HOME)/docker/test-ratelimit/ratelimit.proto
generate/files      += $(OSS_HOME)/OPENSOURCE.md
generate/files      += $(OSS_HOME)/builder/requirements.txt
generate/precious   += $(OSS_HOME)/builder/requirements.txt
generate-fast/files += $(OSS_HOME)/CHANGELOG.md
generate-fast/files += $(OSS_HOME)/pkg/api/getambassador.io/v2/zz_generated.conversion.go
generate-fast/files += $(OSS_HOME)/pkg/api/getambassador.io/v2/zz_generated.conversion-spoke.go
generate-fast/files += $(OSS_HOME)/pkg/api/getambassador.io/v3alpha1/zz_generated.conversion-hub.go
# Individual files: YAML
generate-fast/files += $(OSS_HOME)/manifests/emissary/emissary-crds.yaml.in
generate-fast/files += $(OSS_HOME)/manifests/emissary/emissary-emissaryns.yaml.in
generate-fast/files += $(OSS_HOME)/manifests/emissary/emissary-defaultns.yaml.in
generate-fast/files += $(OSS_HOME)/pkg/api/getambassador.io/crds.yaml
generate-fast/files += $(OSS_HOME)/python/tests/integration/manifests/ambassador.yaml
generate-fast/files += $(OSS_HOME)/python/tests/integration/manifests/crds.yaml
generate-fast/files += $(OSS_HOME)/python/tests/integration/manifests/rbac_cluster_scope.yaml
generate-fast/files += $(OSS_HOME)/python/tests/integration/manifests/rbac_namespace_scope.yaml
# Individual files: Test TLS Certificates
generate-fast/files += $(OSS_HOME)/builder/server.crt
generate-fast/files += $(OSS_HOME)/builder/server.key
generate-fast/files += $(OSS_HOME)/docker/test-auth/authsvc.crt
generate-fast/files += $(OSS_HOME)/docker/test-auth/authsvc.key
generate-fast/files += $(OSS_HOME)/docker/test-ratelimit/ratelimit.crt
generate-fast/files += $(OSS_HOME)/docker/test-ratelimit/ratelimit.key
generate-fast/files += $(OSS_HOME)/docker/test-shadow/shadowsvc.crt
generate-fast/files += $(OSS_HOME)/docker/test-shadow/shadowsvc.key
generate-fast/files += $(OSS_HOME)/python/tests/selfsigned.py

generate: ## Update generated sources that get committed to Git
generate:
	$(MAKE) generate-clean
# This (generating specific targets early, then having a separate `_generate`) is a hack.  Because the
# full value of $(generate/files) is based on the listing of files in $(OSS_HOME)/api/, we need to
# make sure that those directories are fully populated before we evaluate the full $(generate/files).
	$(MAKE) $(OSS_HOME)/api/envoy $(OSS_HOME)/api/pb
	$(MAKE) _generate
_generate:
	@echo '$(MAKE) $$(generate/files)'; $(MAKE) $(patsubst %/,%,$(generate/files))
.PHONY: generate _generate

generate-clean: ## Delete generated sources that get committed to Git
	rm -rf $(filter-out $(generate/precious),$(generate/files))
	find $(OSS_HOME)/pkg/api/getambassador.io -name 'zz_generated.*.go' -print -delete # generated as a side-effect of other files
.PHONY: generate-clean

generate-fast: ## Update the subset of generated-sources-that-get-committed-to-Git that can be updated quickly
generate-fast:
	$(MAKE) generate-fast-clean
	$(MAKE) $(patsubst %/,%,$(generate-fast/files))
.PHONY: generate-fast

generate-fast-clean: ## Delete the subset of generated-sources-that-get-committed-to-Git that can be updated quickly
	rm -rf $(filter-out $(generate/precious),$(generate-fast/files))
	find $(OSS_HOME)/pkg/api/getambassador.io -name 'zz_generated.*.go' -print -delete # generated as a side-effect of other files
.PHONY: generate-fast-clean

#
# Helper Make functions and variables

# Usage: $(call joinlist,SEPARATOR,LIST)
# Example: $(call joinlist,/,foo bar baz) => foo/bar/baz
joinlist=$(if $(word 2,$2),$(firstword $2)$1$(call joinlist,$1,$(wordlist 2,$(words $2),$2)),$2)

comma=,

gomoddir = $(shell cd $(OSS_HOME); go list -mod=readonly $1/... >/dev/null 2>/dev/null; go list -mod=readonly -m -f='{{.Dir}}' $1)

#
# Rules for downloading ("vendoring") sources from elsewhere

# How to set ENVOY_GO_CONTROL_PLANE_COMMIT: In envoyproxy/go-control-plane.git, the majority of
# commits have a commit message of the form "Mirrored from envoyproxy/envoy @ ${envoy.git_commit}".
# Look for the most recent one that names a commit that is an ancestor of our ENVOY_COMMIT.  If there
# are commits not of that form immediately following that commit, you can take them in too (but that's
# pretty uncommon).  Since that's a simple sentence, but it can be tedious to go through and check
# which commits are ancestors, I added `make guess-envoy-go-control-plane-commit` to do that in an
# automated way!  Still look at the commit yourself to make sure it seems sane; blindly trusting
# machines is bad, mmkay?
ENVOY_GO_CONTROL_PLANE_COMMIT = v0.9.6

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
	    -e 's,github\.com/envoyproxy/go-control-plane/pkg,github.com/datawire/ambassador/v2/pkg/envoy-control-plane,g' \
	    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/datawire/ambassador/v2/pkg/api/envoy,g' \
	    -- {} +; \
	  find "$$tmpdir" -name '*.bak' -delete; \
	  mv "$$tmpdir" $(abspath $@); \
	}
	cd $(OSS_HOME) && gofmt -w -s ./pkg/envoy-control-plane/

$(OSS_HOME)/docker/test-ratelimit/ratelimit.proto:
	set -e; { \
	  url=https://raw.githubusercontent.com/envoyproxy/ratelimit/v1.3.0/proto/ratelimit/ratelimit.proto; \
	  echo "// Downloaded from $$url"; \
	  echo; \
	  curl --fail -L "$$url"; \
	} > $@

#
# `make generate` certificate generation

$(OSS_HOME)/builder/server.crt: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=$@ --out-key=/dev/null --hosts=kat-server.test.getambassador.io
$(OSS_HOME)/builder/server.key: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=/dev/null --out-key=$@ --hosts=kat-server.test.getambassador.io

$(OSS_HOME)/docker/test-auth/authsvc.crt: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=$@ --out-key=/dev/null --hosts=authsvc.datawire.io
$(OSS_HOME)/docker/test-auth/authsvc.key: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=/dev/null --out-key=$@ --hosts=authsvc.datawire.io

$(OSS_HOME)/docker/test-ratelimit/ratelimit.crt: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=$@ --out-key=/dev/null --hosts=ratelimit.datawire.io
$(OSS_HOME)/docker/test-ratelimit/ratelimit.key: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=/dev/null --out-key=$@ --hosts=ratelimit.datawire.io

$(OSS_HOME)/docker/test-shadow/shadowsvc.crt: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=$@ --out-key=/dev/null --hosts=demosvc.datawire.io
$(OSS_HOME)/docker/test-shadow/shadowsvc.key: $(tools/testcert-gen)
	$(tools/testcert-gen) --out-cert=/dev/null --out-key=$@ --hosts=demosvc.datawire.io

$(OSS_HOME)/python/tests/selfsigned.py: %: %.gen $(tools/testcert-gen)
	$@.gen $(tools/testcert-gen) >$@

#
# `make generate` protobuf rules

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

clean: _generate_clean
_generate_clean:
	rm -rf $(OSS_HOME)/_generate.tmp
.PHONY: _generate_clean

#
# `make generate` rules to update generated YAML files (and `zz_generated.*.go` Go files)

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
controller-gen/options/object      += headerFile=build-aux/copyright-boilerplate.go.txt
controller-gen/options/crd         += trivialVersions=false # Requires Kubernetes 1.13+
controller-gen/options/crd         += crdVersions=v1        # Requires Kubernetes 1.16+
controller-gen/output/crd           = dir=$@
$(OSS_HOME)/_generate.tmp/crds: $(tools/controller-gen) build-aux/copyright-boilerplate.go.txt FORCE
	@printf '  $(CYN)Running controller-gen$(END)\n'
	rm -rf $@
	mkdir -p $@
	cd $(OSS_HOME) && $(tools/controller-gen) \
	  $(foreach varname,$(sort $(filter controller-gen/options/%,$(.VARIABLES))), $(patsubst controller-gen/options/%,%,$(varname))$(if $(strip $($(varname))),:$(call joinlist,$(comma),$($(varname)))) ) \
	  $(foreach varname,$(sort $(filter controller-gen/output/%,$(.VARIABLES))), $(call joinlist,:,output $(patsubst controller-gen/output/%,%,$(varname)) $($(varname))) ) \
	  $(foreach p,$(wildcard ./pkg/api/getambassador.io/v*/),paths=$p...)

$(OSS_HOME)/%/zz_generated.conversion.go: $(tools/conversion-gen) build-aux/copyright-boilerplate.go.txt FORCE
	rm -f $@ $(@D)/*.scaffold.go
	GOPATH= GOFLAGS=-mod=mod $(tools/conversion-gen) \
	  --skip-unsafe \
	  --go-header-file=build-aux/copyright-boilerplate.go.txt \
	  --input-dirs=./$* \
	  --output-file-base=zz_generated.conversion

$(OSS_HOME)/%/handwritten.conversion.scaffold.go: $(OSS_HOME)/%/zz_generated.conversion.go
	{ \
	  gawk ' \
	    BEGIN { \
	      print("//+build scaffold"); \
	      print(""); \
	      print("package $(notdir $*)"); \
	      inFunc=0; \
	      curFunc=""; \
	    } \
	    match($$0, /^func auto(Convert_[^(]+)(\(.*)/, m) { \
	      if (inFunc) { \
	        print("  return nil"); \
	        print("}"); \
	        print(""); \
	        inFunc=0; \
	      } \
	      curFunc=\
	        "func " m[1] m[2] \
	        "  if err := auto" m[1] "(in, out, s); err != nil {" \
	        "    return err" \
	        "  }"; \
	    } \
	    /INFO|WARN/ { \
	      if (!inFunc) { \
	        print(curFunc); \
	        inFunc=1; \
	      } \
	      print; \
	    } \
	    END { \
	      if (inFunc) { \
	        print("  return nil"); \
	        print("}"); \
	      } \
	    }' | \
	  gofmt; \
	} <$< >$@

$(OSS_HOME)/%/zz_generated.conversion-hub.go: FORCE
	rm -f $@
	{ \
	  gawk ' \
	    BEGIN { \
	       print("package $(notdir $*)"); \
	       print(""); \
	       object=0; \
	    } \
	    /\/\/ \+kubebuilder:object:root=true/ { \
	       object=1; \
	    } \
	    /^type \S+ struct/ && object { \
	        if (!match($$2, /List$$/)) { \
	          print "func(*" $$2 ") Hub() {}"; \
	        } \
	        object=0; \
	    }' $(sort $(wildcard $(@D)/*.go)) | \
	  gofmt; \
	} >$@

$(OSS_HOME)/%/zz_generated.conversion-spoke.go: FORCE
	rm -f $@
	{ \
	  gawk ' \
	    BEGIN { \
	       print("package $(notdir $*)"); \
	       print(""); \
	       print("import ("); \
	       print("  \"k8s.io/apimachinery/pkg/runtime\""); \
	       print("  \"sigs.k8s.io/controller-runtime/pkg/conversion\""); \
	       print(")"); \
	       print(""); \
	       print("func convert(src, dst runtime.Object) error {"); \
	       print("  s, err := SchemeBuilder.Build()"); \
	       print("  if err != nil { return err }"); \
	       print("  return s.Convert(src, dst, nil)"); \
	       print("}"); \
	       print(""); \
	       object=0; \
	    } \
	    /\/\/ \+kubebuilder:object:root=true/ { \
	       object=1; \
	    } \
	    /^type \S+ struct/ && object { \
	        if (!match($$2, /List$$/)) { \
	          print "func(dst *" $$2 ") ConvertFrom(src conversion.Hub) error { return convert(src, dst) }"; \
	          print "func(src *" $$2 ") ConvertTo(dst conversion.Hub) error { return convert(src, dst) }"; \
	        } \
	        object=0; \
	    }' $(sort $(wildcard $(@D)/*.go)) | \
	  gofmt; \
	} >$@

$(OSS_HOME)/manifests/emissary/emissary-crds.yaml.in: $(OSS_HOME)/_generate.tmp/crds $(tools/fix-crds)
	$(tools/fix-crds) --target=apiserver-kubectl $(sort $(wildcard $</*.yaml)) >$@

$(OSS_HOME)/python/tests/integration/manifests/crds.yaml: $(OSS_HOME)/_generate.tmp/crds $(tools/fix-crds)
	$(tools/fix-crds) --target=apiserver-kat $(sort $(wildcard $</*.yaml)) >$@

$(OSS_HOME)/pkg/api/getambassador.io/crds.yaml: $(OSS_HOME)/_generate.tmp/crds $(tools/fix-crds)
	$(tools/fix-crds) --target=internal-validator $(sort $(wildcard $</*.yaml)) >$@

python-setup: create-venv
	$(OSS_HOME)/venv/bin/python -m pip install ruamel.yaml
.PHONY: python-setup

helm.name.emissary-emissaryns = emissary-ingress
helm.name.emissary-defaultns = emissary-ingress
helm.namespace.emissary-emissaryns = emissary
helm.namespace.emissary-defaultns = default
$(OSS_HOME)/k8s-config/%/helm-expanded.yaml: \
  $(OSS_HOME)/k8s-config/%/values.yaml \
  $(OSS_HOME)/charts/emissary-ingress/templates $(wildcard $(OSS_HOME)/charts/emissary-ingress/templates/*.yaml) \
  $(OSS_HOME)/charts/emissary-ingress/values.yaml \
  FORCE
	helm template --namespace=$(helm.namespace.$*) --values=$(@D)/values.yaml $(or $(helm.name.$*),$*) $(OSS_HOME)/charts/emissary-ingress >$@
$(OSS_HOME)/k8s-config/%/output.yaml: \
  $(OSS_HOME)/k8s-config/%/helm-expanded.yaml \
  $(OSS_HOME)/k8s-config/%/require.yaml \
  $(OSS_HOME)/k8s-config/create_yaml.py \
  python-setup
	. $(OSS_HOME)/venv/bin/activate && $(filter %.py,$^) $(filter %/helm-expanded.yaml,$^) $(filter %/require.yaml,$^) >$@
$(OSS_HOME)/manifests/emissary/%.yaml.in: $(OSS_HOME)/k8s-config/%/output.yaml
	cp $< $@
$(OSS_HOME)/python/tests/integration/manifests/%.yaml: $(OSS_HOME)/k8s-config/kat-%/output.yaml
	sed -e 's/«/{/g' -e 's/»/}/g' -e 's/♯.*//g' -e 's/- ←//g' <$< >$@
$(OSS_HOME)/python/tests/integration/manifests/rbac_cluster_scope.yaml: $(OSS_HOME)/k8s-config/kat-rbac-multinamespace/output.yaml
	sed -e 's/«/{/g' -e 's/»/}/g' -e 's/♯.*//g' -e 's/- ←//g' <$< >$@
$(OSS_HOME)/python/tests/integration/manifests/rbac_namespace_scope.yaml: $(OSS_HOME)/k8s-config/kat-rbac-singlenamespace/output.yaml
	sed -e 's/«/{/g' -e 's/»/}/g' -e 's/♯.*//g' -e 's/- ←//g' <$< >$@

#
# Generate report on dependencies

$(OSS_HOME)/build-aux/pip-show.txt: sync
	docker exec $$($(BUILDER)) sh -c 'pip freeze --exclude-editable | cut -d= -f1 | xargs pip show' > $@

$(OSS_HOME)/builder/requirements.txt: %.txt: %.in FORCE
	$(BUILDER) pip-compile
.PRECIOUS: $(OSS_HOME)/builder/requirements.txt

$(OSS_HOME)/build-aux/go-version.txt: $(OSS_HOME)/builder/Dockerfile.base
	sed -En 's,.*https://dl\.google\.com/go/go([0-9a-z.-]*)\.linux-amd64\.tar\.gz.*,\1,p' < $< > $@
$(OSS_HOME)/build-aux/py-version.txt: $(OSS_HOME)/builder/Dockerfile.base
	{ grep -o 'python3=\S*' | cut -d= -f2; } < $< > $@

$(OSS_HOME)/build-aux/go1%.src.tar.gz:
	curl -o $@ --fail -L https://dl.google.com/go/$(@F)

$(OSS_HOME)/OPENSOURCE.md: $(tools/go-mkopensource) $(tools/py-mkopensource) $(OSS_HOME)/build-aux/go-version.txt $(OSS_HOME)/build-aux/pip-show.txt
	$(MAKE) $(OSS_HOME)/build-aux/go$$(cat $(OSS_HOME)/build-aux/go-version.txt).src.tar.gz
	set -e; { \
		cd $(OSS_HOME); \
		$(tools/go-mkopensource) --output-format=txt --package=mod --gotar=build-aux/go$$(cat $(OSS_HOME)/build-aux/go-version.txt).src.tar.gz; \
		echo; \
		{ sed 's/^---$$//' $(OSS_HOME)/build-aux/pip-show.txt; echo; } | $(tools/py-mkopensource); \
	} > $@

#
# Misc. other `make generate` rules

$(OSS_HOME)/CHANGELOG.md: $(OSS_HOME)/docs/CHANGELOG.tpl $(OSS_HOME)/docs/releaseNotes.yml
	docker run --rm \
	  -v $(OSS_HOME)/docs/CHANGELOG.tpl:/tmp/CHANGELOG.tpl \
	  -v $(OSS_HOME)/docs/releaseNotes.yml:/tmp/releaseNotes.yml \
	  hairyhenderson/gomplate --verbose --file /tmp/CHANGELOG.tpl --datasource relnotes=/tmp/releaseNotes.yml > CHANGELOG.md
