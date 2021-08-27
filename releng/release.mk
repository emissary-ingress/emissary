
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
