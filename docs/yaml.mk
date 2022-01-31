GENERATED_YAML_DIR := $(OSS_HOME)/build/docs/

generate-docs-yaml/files += $(patsubst $(OSS_HOME)/docs/%.yaml, $(GENERATED_YAML_DIR)/%.yaml, $(shell find $(OSS_HOME)/docs/ -name '*.yaml' -type f))

generate-docs-yaml:
	@rm -rf $(GENERATED_YAML_DIR)
	@mkdir -p $(GENERATED_YAML_DIR)
	@echo '$(MAKE) $$(generate-docs-yaml/files)'; $(MAKE) $(GENERATED_YAML_DIR) $(generate-docs-yaml/files)
.PHONY: generate-docs-yaml

publish-docs-yaml: generate-docs-yaml
	if [ ! -z $(IS_PRIVATE) ]; then \
		echo "Private repo, not pushing chart" ;\
		exit 1 ;\
	fi;
	@$(OSS_HOME)/docs/publish_yaml_s3.sh $(GENERATED_YAML_DIR)yaml/ $(generate-docs-yaml/files)
	@rm -rf $(GENERATED_YAML_DIR)
.PHONY: publish-docs-yaml

$(GENERATED_YAML_DIR)/%.yaml: FORCE
	$(OSS_HOME)/docs/template_versions.sh $(patsubst $(GENERATED_YAML_DIR)/%.yaml, $(OSS_HOME)/docs/%.yaml, $@) $@
