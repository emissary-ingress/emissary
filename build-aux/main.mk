include build-aux/tools.mk

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

docker/.base.img.tar.stamp: FORCE $(tools/crane) builder/Dockerfile
	$(tools/crane) pull $(shell sed -n 's,ARG base=,,p' < builder/Dockerfile) $@ || test -e $@
