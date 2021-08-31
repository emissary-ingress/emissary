
########################################################################
# Manual commands
# These are the commands that are currently run manually in the normal
# release process
########################################################################
release/start:
	@test -n "$(VERSION)" || (printf "VERSION is required\n"; exit 1)
	@$(OSS_HOME)/releng/00-release-start --next-version $(VERSION)
.PHONY: release/start

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

