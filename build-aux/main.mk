include build-aux/tools.mk
include build-aux/var.mk

#
# Utility rules

# For files that should only-maybe update when the rule runs, put ".stamp" on
# the left-side of the ":", and just go ahead and update it within the rule.
#
# ".stamp" should NEVER appear in a dependency list (that is, it
# should never be on the right-side of the ":"), save for in this rule
# itself.
%: %.stamp $(tools/copy-ifchanged)
	@$(tools/copy-ifchanged) $< $@
docker/%: docker/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@

# Load ocibuild files in to dockerd.
_ocibuild-images  = base
_ocibuild-images += kat-client
_ocibuild-images += kat-server
$(foreach img,$(_ocibuild-images),docker/.$(img).docker.stamp): docker/.%.docker.stamp: docker/%.img.tar
	docker load < $<
	docker inspect $$(bsdtar xfO $< manifest.json|jq -r '.[0].RepoTags[0]') --format='{{.Id}}' > $@

#
# Specific rules

# For images we can either write rules for
#  - `docker/.NAME.img.tar.stamp` for ocibuild-oriented images, or
#  - `docker/.NAME.docker.stamp` for `docker build`-oriented images.
#
# Note that there are a few images used by the test suite that are
# defined in check.mk, rather than here.

# base: Base OS; none of our specific stuff.  Used for auxiliar test images
# that don't need Emissary-specif stuff.
docker/.base.img.tar.stamp: FORCE $(tools/crane) docker/base-python/Dockerfile
	$(tools/crane) pull $(shell gawk '$$1 == "FROM" { print $$2; quit; }' < docker/base-python/Dockerfile) $@ || test -e $@

# base-python: Base OS, plus some Emissary-specific setup of
# low-level/expensive pieces of the Python environment.  This does NOT
# include the packages installed by `requirements.txt`.
#
# At the moment, it also includes some other stuff too (kubectl...),
# but including those things at such an early stage should be
# understood to be debt from a previous build system, and not
# something we're actually happy with.
#
# In the long-run, this will likely always be a `docker build` rather
# than an `ocibuild`, in order to do truly base-OS-specific setup
# (`apk add`, libc-specific compilation...).
docker/.base-python.docker.stamp: FORCE docker/base-python/Dockerfile docker/base-python.docker.gen
	docker/base-python.docker.gen >$@

# base-pip: base-python, but with requirements.txt installed.
#
# Mixed feelings about this one; it kinda wants to not be a separate
# image and just be part of the main emissary Dockerfile.  But that
# would create problems for generate.mk's `pip freeze` step.  Perhaps
# it will get to go away with `ocibuild`.
#
# TODO(lukeshu): Figure out a `py-list-deps`-based workflow for
# updating requirements-dev.txt.
#python/requirements-dev.txt: $(tools/py-list-deps) $(tools/write-ifchanged) FORCE
#	$(tools/py-list-deps) --include-dev python/ | $(tools/write-ifchanged) $@
python/requirements.in: $(tools/py-list-deps) $(tools/write-ifchanged) FORCE
	set -o pipefail; $(tools/py-list-deps) --no-include-dev python/ | $(tools/write-ifchanged) $@
python/.requirements.txt.stamp: python/requirements.in docker/base-python.docker.tag.local
# The --interactive is so that stdin gets passed through; otherwise Docker closes stdin.
	set -ex -o pipefail; { \
	  docker run --platform="$(BUILD_ARCH)" --rm --interactive "$$(cat docker/base-python.docker)" sh -c 'tar xf - && find ~/.cache/pip -name "maturin-*.whl" -exec pip install --no-deps {} + >&2 && pip-compile --allow-unsafe --no-build-isolation -q >&2 && cat requirements.txt' \
	    < <(bsdtar -cf - -C $(@D) requirements.in requirements.txt) \
	    > $@; }
python/requirements.txt: python/%: python/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
.PRECIOUS: python/requirements.txt
docker/base-pip/requirements.txt: python/requirements.txt $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
docker/.base-pip.docker.stamp: docker/.%.docker.stamp: docker/%/Dockerfile docker/%/requirements.txt docker/base-python.docker.tag.local
	docker build --platform="$(BUILD_ARCH)" --build-arg=from="$$(sed -n 2p docker/base-python.docker.tag.local)" --iidfile=$@ $(<D)

# The Helm chart
build-output/chart-%.d: \
  $(shell find charts/emissary-ingress) \
  $(var.)DEV_REGISTRY $(var.)RELEASE_REGISTRY \
  $(tools/chart-doc-gen)
ifeq ($(CI),)
	rm -rf $@
else
	@if test -d $@; then \
	  echo 'This should not happen in CI: $@ should not need to change' >&2; \
	  echo 'Files triggering the change are: $?' >&2; \
	  exit 1; \
	fi
endif
	mkdir -p $(@D)
	cp -a $< $@
	@PS4=; set -ex -o pipefail; { \
	  if [[ '$(word 1,$(subst _, ,$*))' =~ ^[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+|-ea)?$$ ]]; then \
	    registry=$(RELEASE_REGISTRY); \
	  else \
	    registry=$(DEV_REGISTRY); \
	  fi; \
	  for file in Chart.yaml values.yaml; do \
	    sed \
	      -e 's/@version@/$(word 1,$(subst _, ,$*))/g' \
	      -e 's/@chartVersion@/$(word 2,$(subst _, ,$*))/g' \
	      -e "s,@imageRepo@,$${registry}/emissary,g" \
	      <'$<'/"$${file}.in" \
	      >'$@'/"$${file}"; \
	  done; \
	}
	$(tools/chart-doc-gen) -d $</doc.yaml -t $</readme.tpl -v $@/values.yaml >$@/README.md
build-output/chart-%.tgz: build-output/chart-%.d
	helm package --destination=$< $<
	mv $</emissary-ingress-$(word 2,$(subst _, ,$*)).tgz $@

# Convenience aliases for the Helm chart
chart_dir = build-output/chart-$(patsubst v%,%,$(VERSION))_$(patsubst v%,%,$(CHART_VERSION)).d
chart_tgz = $(patsubst %.d,%.tgz,$(chart_dir))
chart: $(chart_tgz)
PHONY: chart

boguschart_dir = build-output/chart-2.0.0-bogus_7.0.0-bogus.d
boguschart_tgz = $(patsubst %.d,%.tgz,$(boguschart_dir))

# YAML manifests
build-output/yaml-%: $(shell find $(CURDIR)/manifests/emissary/ -type d -o -name '*.yaml.in') $(var.)DEV_REGISTRY $(var.)RELEASE_REGISTRY
ifeq ($(CI),)
	rm -rf $@
else
	@if test -d $@; then \
	  echo 'This should not happen in CI: $@ should not need to change' >&2; \
	  echo 'Files triggering the change are: $?' >&2; \
	  exit 1; \
	fi
endif
	mkdir -p $@
	$(foreach src,$(filter %.yaml.in,$^),$(foreach dst,$(patsubst $(CURDIR)/manifests/emissary/%.yaml.in,$@/%.yaml,$(src)),\
	  { \
	    if [[ '$*' =~ ^[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+|-ea)?$$ ]]; then \
	      registry=$(RELEASE_REGISTRY); \
	    else \
	      registry=$(DEV_REGISTRY); \
	    fi; \
	    sed -e 's/\$$version\$$/$*/g' -e 's,\$$imageRepo\$$,'"$${registry}"'/emissary,g' <$(src) >$(dst); \
	  }$(NL)))

build-output/docs-yaml-%: $(shell find docs/yaml)
ifeq ($(CI),)
	rm -rf $@
else
	@if test -d $@; then \
	  echo 'This should not happen in CI: $@ should not need to change' >&2; \
	  echo 'Files triggering the change are: $?' >&2; \
	  exit 1; \
	fi
endif
	$(foreach src,$(filter %.yaml,$^),$(foreach dst,$(patsubst docs/yaml/%,$@/%,$(src)),\
	  mkdir -p $(dir $(dst))$(NL)\
	  sed -e 's/\$$version\$$/$*/g' -e 's/\$$quoteVersion$$/0.4.1/g' <$(src) >$(dst)$(NL)))
