include build-aux/tools.mk
include build-aux/var.mk

#
# Utility rules

# Assume that any rule ending with '.clean' is phony.
.PHONY: %.clean

# Also provide a basic *.clean implementation... well, it'd be.  But
# because of what I'm convinced is a bug in Make, it is confusing this
# %.clean rule with the %.docker.clean rule.  So I named this one
# `%.rm`.  But I'd have liked to name it `%.clean`.
%.rm:
	rm -f $*
.PHONY: %.rm
%.rm-r:
	rm -rf $*
.PHONY: %.rm-r

# For files that should only-maybe update when the rule runs, put ".stamp" on
# the left-side of the ":", and just go ahead and update it within the rule.
#
# ".stamp" should NEVER appear in a dependency list (that is, it
# should never be on the right-side of the ":"), save for in this rule
# itself.
%: %.stamp $(tools/copy-ifchanged)
	@$(tools/copy-ifchanged) $< $@

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
	mv $</emissary-chart-$(word 2,$(subst _, ,$*)).tgz $@

# Convenience aliases for the Helm chart
chart_dir = build-output/chart-$(patsubst v%,%,$(VERSION))_$(patsubst v%,%,$(CHART_VERSION)).d
chart_tgz = $(patsubst %.d,%.tgz,$(chart_dir))
chart: $(chart_tgz)
PHONY: chart

_major_version = $(firstword $(subst ., ,$(patsubst v%,%,$(VERSION))))
_chart_major_version = $(firstword $(subst ., ,$(patsubst v%,%,$(CHART_VERSION))))
boguschart_dir = build-output/chart-$(_major_version).0.0-bogus_$(_chart_major_version).0.0-bogus.d
boguschart_tgz = $(patsubst %.d,%.tgz,$(boguschart_dir))

# YAML manifests
build-output/yaml-%: $(shell find $(CURDIR)/manifests/emissary -type d -o -name '*.yaml.in') $(var.)DEV_REGISTRY $(var.)RELEASE_REGISTRY
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

#
# Destructive rules

clobber: clean

clean: build-output.rm-r

python.clean releng.clean: %.clean: python/ambassador.egg-info.rm-r
	find $* -name __pycache__ -exec rm -rf -- {} +
clean: python.clean releng.clean

cmd.clean pkg.clean: %.clean:
	find $* -name '*.yaml.o' -delete
clean: cmd.clean pkg.clean
