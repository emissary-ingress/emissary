
########################################################################
# Manual commands
# These are the commands that are currently run manually in the normal
# release process
########################################################################
release/start:
	@test -n "$(VERSION)" || (printf "VERSION is required\n"; exit 1)
	@$(OSS_HOME)/releng/00-release-start --next-version $(VERSION)
.PHONY: release/start

release/rc/tag:
	@set -e; { \
		$(OSS_HOME)/releng/release-wait-for-commit --commit $$(git rev-parse HEAD) --s3-key dev-builds ; \
		rc_num=$$(PAGER= git tag --sort=-version:refname -l 'v$(VERSIONS_YAML_VER)-rc.*' | wc -l) ; \
		rc_tag=v$(VERSIONS_YAML_VER)-rc.$$rc_num ; \
		echo "Tagging $$rc_tag" ; \
		git tag -m $$rc_tag -a $$rc_tag ; \
		git push origin $$rc_tag ; \
	}
.PHONY: release/rc/tag

release/ga/tag:
	@[[ "$(VERSIONS_YAML_VER)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: RELEASE_VERSION=%s does not look like a GA tag\n' "$(VERSIONS_YAML_VER)"; exit 1)
	@[[ -z "$(IS_DIRTY)" ]] || (printf '$(RED)ERROR: tree must be clean\n'; exit 1)
	$(OSS_HOME)/releng/release-wait-for-commit --commit $$(git rev-parse HEAD) --s3-key passed-builds
	git tag -m v$(VERSIONS_YAML_VER) -a v$(VERSIONS_YAML_VER)
	git push origin v$(VERSIONS_YAML_VER)
.PHONY: release/ga/tag

########################################################################
# CI commands
# These commands are run in CI in a normal release process
########################################################################

release/ga/changelog-update:
	$(OSS_HOME)/releng/release-go-changelog-update --quiet $(VERSIONS_YAML_VER)
.PHONY: release/ga/changelog-update

release/ga/create-gh-release:
	@$(OSS_HOME)/releng/release-create-github $(VERSIONS_YAML_VER)
.PHONY: release/ga/create-gh-release

release/ga/manifest-update:
	$(OSS_HOME)/release-manifest-image-update --oss-version $(VERSIONS_YAML_VER)
.PHONY: release/ga/manifest-update

