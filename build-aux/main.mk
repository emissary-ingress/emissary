include build-aux/tools.mk

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
	  docker run --rm --interactive "$$(cat docker/base-python.docker)" sh -c 'tar xf - && find ~/.cache/pip -name "maturin-*.whl" -exec pip install --no-deps {} + >&@ && pip-compile --allow-unsafe --no-build-isolation -q >&2 && cat requirements.txt' \
	    < <(bsdtar -cf - -C $(@D) requirements.in requirements.txt) \
	    > $@; }
python/requirements.txt: python/%: python/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
.PRECIOUS: python/requirements.txt
docker/base-pip/requirements.txt: python/requirements.txt $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@
docker/.base-pip.docker.stamp: docker/.%.docker.stamp: docker/%/Dockerfile docker/%/requirements.txt docker/base-python.docker.tag.local
	docker build --build-arg=from="$$(sed -n 2p docker/base-python.docker.tag.local)" --iidfile=$@ $(<D)

# The Helm chart
build-output/charts/emissary-ingress-$(patsubst v%,%,$(CHART_VERSION)).tgz: \
  charts/emissary-ingress/Chart.yaml \
  charts/emissary-ingress/values.yaml \
  charts/emissary-ingress/README.md
	mkdir -p $(@D)
	helm package --destination=$(@D) $(<D)

# Convience alias for the Helm chart
chart: build-output/charts/emissary-ingress-$(patsubst v%,%,$(CHART_VERSION)).tgz
PHONY: chart
