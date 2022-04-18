publish-docs-yaml: build-output/docs-yaml-$(patsubst v%,%,$(VERSION))
ifneq ($(IS_PRIVATE),)
	@echo "Private repo, not pushing chart" >&2
	@exit 1
else
	docs/publish_yaml_s3.sh $<
endif
.PHONY: publish-docs-yaml

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
