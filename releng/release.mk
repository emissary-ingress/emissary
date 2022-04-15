
########################################################################
# Manual commands
# These are the commands that are currently run manually in the normal
# release process
########################################################################

# `make release/start START_VERSION=X.Y.0` is meant to be run by the
# human maintainer when work on a new X.Y.0 starts.
release/start:
	@[[ "$(START_VERSION)" =~ ^[0-9]+\.[0-9]+\.0$$ ]] || (printf '$(RED)ERROR: START_VERSION must be set to a GA "2.Y.0" value; it is set to "%s"$(END)\n' "$(START_VERSION)"; exit 1)
	@$(OSS_HOME)/releng/00-release-start --next-version $(START_VERSION)
.PHONY: release/start

# `make release/ga/changelog-update CHANGELOG_VERSION=X.Y.Z` is meant
# to be run by the human maintainer when preparing the final version
# of the `rel/vX.Y.Z` branch.
release/ga/changelog-update:
	$(OSS_HOME)/releng/release-go-changelog-update --quiet $(CHANGELOG_VERSION)
.PHONY: release/ga/changelog-update

########################################################################
# CI commands
# These commands are run in CI in a normal release process
########################################################################

release/ga/create-gh-release:
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]] || (printf '$(RED)ERROR: VERSION must be set to a GA "v2.Y.Z" value; it is set to "%s"$(END)\n' "$(VERSION)"; exit 1)
	@$(OSS_HOME)/releng/release-create-github $(patsubst v%,%,$(VERSION))
.PHONY: release/ga/create-gh-release

release/chart-create-gh-release:
	$(OSS_HOME)/releng/chart-create-gh-release
.PHONY: release/chart-create-gh-release
