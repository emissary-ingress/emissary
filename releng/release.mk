
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

release/rc/print-test-artifacts:
	@set -e; { \
		manifest_ver=$(RELEASE_VERSION) ; \
		manifest_ver=$${manifest_ver%"-dirty"} ; \
		echo "RC_TAG=v$$manifest_ver" ; \
		echo "AMBASSADOR_MANIFEST_URL=https://app.getambassador.io/yaml/emissary/$$manifest_ver" ; \
		echo "HELM_CHART_VERSION=`grep 'version' $(OSS_HOME)/charts/emissary-ingress/Chart.yaml | awk '{ print $$2 }'`" ; \
	}
.PHONY: release/print-test-artifacts

release/ga/changelog-update:
	$(OSS_HOME)/releng/release-go-changelog-update --quiet $(VERSION)
.PHONY: release/ga/changelog-update

release/ga/create-gh-release:
	@$(OSS_HOME)/releng/release-create-github $(VERSION)
.PHONY: release/ga/create-gh-release

release/ga/manifest-update:
	$(OSS_HOME)/releng/release-manifest-image-update --oss-version $(VERSION)
.PHONY: release/ga/manifest-update

