
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

release/rc/print-test-artifacts:
	@set -e; { \
		manifest_ver=$(patsubst v%,%,$(VERSION)) ; \
		manifest_ver=$${manifest_ver%"-dirty"} ; \
		echo "RC_TAG=v$$manifest_ver" ; \
		echo "AMBASSADOR_MANIFEST_URL=https://app.getambassador.io/yaml/emissary/$$manifest_ver" ; \
		echo "HELM_CHART_VERSION=$$(gawk '$$1 == "version:" { print $$2 }' <charts/emissary-ingress/Chart.yaml)" ; \
	}
.PHONY: release/print-test-artifacts

release/rc/check:
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+rc\.[0-9]+$$ ]] || (printf '$(RED)ERROR: VERSION must be set to an RC "v2.Y.Z-rc.N" value; it is set to "%s"$(END)\n' "$(VERSION)"; exit 1)
	{ $(OSS_HOME)/releng/release-rc-check \
	  --rc-version=$(patsubst v%,%,$(VERSION)) \
	  --s3-bucket=$(AWS_S3_BUCKET) \
	  --s3-key=charts-dev \
	  --helm-version=$$(gawk '$$1 == "version:" { gsub("-", " "); print $$2; }' <charts/emissary-ingress/Chart.yaml)$$(sed 's/^[^-]*//' <<<'$(VERSION)') \
	  --docker-image=$(RELEASE_REGISTRY)/$(LCNAME):$(patsubst v%,%,$(VERSION)); }
.PHONY: release/rc/check

release/ga/create-gh-release:
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]] || (printf '$(RED)ERROR: VERSION must be set to a GA "v2.Y.Z" value; it is set to "%s"$(END)\n' "$(VERSION)"; exit 1)
	@$(OSS_HOME)/releng/release-create-github $(patsubst v%,%,$(VERSION))
.PHONY: release/ga/create-gh-release

release/chart-create-gh-release:
	$(OSS_HOME)/releng/chart-create-gh-release
.PHONY: release/chart-create-gh-release
