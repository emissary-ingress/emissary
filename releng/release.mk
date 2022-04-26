
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

CHART_S3_BUCKET = $(or $(AWS_S3_BUCKET),datawire-static-files)
CHART_S3_PREFIX = $(if $(findstring -,$(CHART_VERSION)),charts-dev,charts)
release/push-chart: $(chart_tgz)
ifneq ($(IS_PRIVATE),)
	echo "Private repo, not pushing chart" >&2
else
	@if curl -k -L https://s3.amazonaws.com/$(CHART_S3_BUCKET)/$(CHART_S3_PREFIX)/index.yaml | grep -F emissary-ingress-$(patsubst v%,%,$(CHART_VERSION)).tgz; then \
	  printf 'Chart version %s is already in the index\n' '$(CHART_VERSION)' >&2; \
	  exit 1; \
	fi
	{ aws s3api put-object \
	  --bucket $(CHART_S3_BUCKET) \
	  --key $(CHART_S3_PREFIX)/emissary-ingress-$(patsubst v%,%,$(CHART_VERSION)).tgz \
	  --body $<; }
endif
.PHONY: release/push-chart

push-manifests: build-output/yaml-$(patsubst v%,%,$(VERSION))
ifneq ($(IS_PRIVATE),)
	@echo "Private repo, not pushing chart" >&2
	@exit 1
else
	manifests/push_manifests.sh $<
endif
.PHONY: push-manifests

publish-docs-yaml: build-output/docs-yaml-$(patsubst v%,%,$(VERSION))
ifneq ($(IS_PRIVATE),)
	@echo "Private repo, not pushing chart" >&2
	@exit 1
else
	docs/publish_yaml_s3.sh $<
endif
.PHONY: publish-docs-yaml
