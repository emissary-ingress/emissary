# This is only "kinda" the git branch name:
#
#  - if checked out is the synthetic merge-commit for a PR, then use
#    the PR's branch name (even though the merge commit we have
#    checked out isn't part of the branch")
#  - if this is a CI run for a tag (not a branch or PR), then use the
#    tag name
#  - if none of the above, then use the actual git branch name
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_BRANCH := $(or $(TRAVIS_PULL_REQUEST_BRANCH),$(TRAVIS_BRANCH),$(shell git rev-parse --abbrev-ref HEAD))
# The short git commit hash
GIT_COMMIT := $(shell git rev-parse --short HEAD)
# Whether `git add . && git commit` would commit anything (empty=false, nonempty=true)
GIT_DIRTY := $(if $(shell git status --porcelain),dirty)
# The _previous_ tag, plus a git delta, like 0.36.0-436-g8b8c5d3
GIT_DESCRIPTION := $(shell git describe --tags)

ifneq ($(CI),)
ifneq ($(GIT_DIRTY),)
$(warning Build is dirty:)
$(shell git add . >&2)
$(shell PAGER= git diff --cached >&2)
$(error This should not happen in CI: the build should not be dirty)
endif
endif

# RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
# will upload/release the image as.  It does NOT include a leading 'v'
# (trimming the 'v' from the git tag is what the 'patsubst' is for).
# If this is an RC or EA, then it includes the '-rcN' or '-eaN'
# suffix.
#
# BUILD_VERSION is of the same format, but is the version number that
# we build into the image.  Because an image built as a "release
# candidate" will ideally get promoted to be the GA image, we trim off
# the '-rcN' suffix.
RELEASE_VERSION = $(patsubst v%,%,$(or $(TRAVIS_TAG),$(shell git describe --tags --always)))$(if $(GIT_DIRTY),-dirty)
BUILD_VERSION = $(shell echo '$(RELEASE_VERSION)' | sed 's/-rc[0-9]*$$//')

# TODO: validate version is conformant to some set of rules might be a good idea to add here
python/ambassador/VERSION.py: python/VERSION-template.py $(var.)BUILD_VERSION $(var.)GIT_BRANCH $(var.)GIT_DIRTY $(var.)GIT_COMMIT $(var.GIT_DESCRIPTION)
	$(call check_defined, BUILD_VERSION, BUILD_VERSION is not set)
	$(call check_defined, GIT_BRANCH, GIT_BRANCH is not set)
	$(call check_defined, GIT_COMMIT, GIT_COMMIT is not set)
	$(call check_defined, GIT_DESCRIPTION, GIT_DESCRIPTION is not set)
	@echo "Generating and templating version information -> $(BUILD_VERSION)"
	sed \
		-e 's!{{VERSION}}!$(BUILD_VERSION)!g' \
		-e 's!{{GITBRANCH}}!$(GIT_BRANCH)!g' \
		-e 's!{{GITDIRTY}}!$(GIT_DIRTY)!g' \
		-e 's!{{GITCOMMIT}}!$(GIT_COMMIT)!g' \
		-e 's!{{GITDESCRIPTION}}!$(GIT_DESCRIPTION)!g' \
		< python/VERSION-template.py > $@

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = $(strip $(foreach 1,$1, $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),, $(error Undefined $1$(if $2, ($2))))
