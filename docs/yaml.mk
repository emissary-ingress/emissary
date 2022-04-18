docs/yamk.mk/srcs = $(shell find docs/ -name '*.yaml' -type f)
docs/yaml.mk/dsts = $(patsubst docs/%.yaml,build/docs/%.yaml,$(docs/yaml.mk/srcs))

generate-docs-yaml:
	rm -rf build/docs/
	@echo '$(MAKE) $$(docs/yaml.mk/dsts)'; $(MAKE) $(docs/yaml.mk/dsts)
.PHONY: generate-docs-yaml

publish-docs-yaml: generate-docs-yaml
ifneq ($(IS_PRIVATE),)
	@echo "Private repo, not pushing chart" >&2
	@exit 1
else
	docs/publish_yaml_s3.sh build/docs/yaml/
	rm -rf build/docs/
endif
.PHONY: publish-docs-yaml

build/docs/%.yaml: docs/%.yaml FORCE | build/docs
	docs/template_versions.sh $< $@
build/docs:
	mkdir -p $@
