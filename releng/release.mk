
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

# `make release/ga/changelog-update` is meant to be run by the human
# maintainer when preparing the final version of the `rel/vX.Y.Z`
# branch.
release/ga/changelog-update:
	$(OSS_HOME)/releng/release-go-changelog-update --quiet $(VERSIONS_YAML_VER)
.PHONY: release/ga/changelog-update

########################################################################
# CI commands
# These commands are run in CI in a normal release process
########################################################################

release/rc/print-test-artifacts:
	@set -e; { \
		manifest_ver=$(RELEASE_VERSION) ; \
		manifest_ver=$${manifest_ver%"-dirty"} ; \
		echo "RC_TAG=v$$manifest_ver" ; \
		echo "AMBASSADOR_MANIFEST_URL=https://app.getambassador.io/yaml/emissary/$$manifest_ver" ; \
		echo "HELM_CHART_VERSION=`grep 'version' $(OSS_HOME)/charts/emissary-ingress/Chart.yaml | awk '{ print $$2 }'`" ; \
	}
.PHONY: release/print-test-artifacts

release/rc/check:
	@set -ex; { \
		rc_num=$$(PAGER= git tag --sort=-version:refname -l 'v$(VERSIONS_YAML_VER_STRIPPED)-rc.*' | head -n -1 | wc -l) ; \
		rc_tag=$(VERSIONS_YAML_VER_STRIPPED)-rc.$$rc_num ; \
		chart_version=$$(grep 'version:' $(OSS_HOME)/charts/emissary-ingress/Chart.yaml | awk '{ print $$2 }' | sed 's/-ea//g') ; \
		$(OSS_HOME)/releng/release-rc-check \
			--rc-version $$rc_tag --s3-bucket $(AWS_S3_BUCKET) --s3-key charts-dev \
			--helm-version $$chart_version-rc.$$rc_num \
			--docker-image $(RELEASE_REGISTRY)/$(LCNAME):$$rc_tag ; \
	}
.PHONY: release/rc/check

release/ga/create-gh-release:
	@$(OSS_HOME)/releng/release-create-github $(VERSIONS_YAML_VER)
.PHONY: release/ga/create-gh-release

release/chart-create-gh-release:
	$(OSS_HOME)/releng/chart-create-gh-release
.PHONY: release/chart-create-gh-release
