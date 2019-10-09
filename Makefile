OSS_HOME:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

include $(OSS_HOME)/builder/builder.mk

$(call module,ambassador,$(OSS_HOME))

compile: python/ambassador/VERSION.py

## I'm including necessary portions of the old Makefile and build-aux
## in order to get things to work. I would like to re-examine this
## stuff to a) make it match up with the new DX (e.g. the ifeq stuff
## is gonna happen *before* preflight checks, yet it uses stuff we
## should be checking for...), and b) I suspect the overal release
## machinery can be simplified given other changes.

## Stuff below here is from the old Makefile

# GIT_BRANCH on TravisCI needs to be set through some external custom logic. Default to a Git native mechanism or
# use what is defined.
#
# read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
GIT_DIRTY ?= $(shell test -z "$(shell git status --porcelain)" || printf "dirty")

GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

# This commands prints the tag of this commit or "undefined". Later we use GIT_TAG_SANITIZED and set it to "" if this
# string is "undefined" or blank.
GIT_TAG ?= $(shell git name-rev --tags --name-only $(GIT_COMMIT))

GIT_BRANCH_SANITIZED := $(shell printf $(GIT_BRANCH) | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-zA-Z0-9]/-/g' -e 's/-\{2,\}/-/g')
GIT_TAG_SANITIZED := $(shell \
	if [ "$(GIT_TAG)" = "undefined" -o -z "$(GIT_TAG)" ]; then \
		printf ""; \
	else \
		printf "$(GIT_TAG)" | sed -e 's/\^.*//g'; \
	fi \
)

# Trees get dirty sometimes by choice and sometimes accidently. If we are in a dirty tree then append "-dirty" to the
# GIT_COMMIT.
ifeq ($(GIT_DIRTY),dirty)
GIT_VERSION := $(GIT_BRANCH_SANITIZED)-$(GIT_COMMIT)-dirty
else
GIT_VERSION := $(GIT_BRANCH_SANITIZED)-$(GIT_COMMIT)
endif

# This gives the _previous_ tag, plus a git delta, like
# 0.36.0-436-g8b8c5d3
GIT_DESCRIPTION := $(shell git describe $(GIT_COMMIT))

# IS_PRIVATE: empty=false, nonempty=true
# Default is true if any of the git remotes have the string "private" in any of their URLs.
_git_remote_urls := $(shell git remote | xargs -n1 git remote get-url --all)
IS_PRIVATE ?= $(findstring private,$(_git_remote_urls))

# Note that for everything except RC builds, VERSION will be set to the version
# we'd use for a GA build. This is by design.
#
# Also note that we strip off the leading 'v' here -- that's just for the git tag.
ifneq ($(GIT_TAG_SANITIZED),)
VERSION = $(patsubst v%,%,$(firstword $(subst -, ,$(GIT_TAG_SANITIZED))))
else
VERSION = $(patsubst v%,%,$(firstword $(subst -, ,$(GIT_VERSION))))
endif

python/ambassador/VERSION.py: FORCE $(WRITE_IFCHANGED)
	$(call check_defined, VERSION, VERSION is not set)
	$(call check_defined, GIT_BRANCH, GIT_BRANCH is not set)
	$(call check_defined, GIT_COMMIT, GIT_COMMIT is not set)
	$(call check_defined, GIT_DESCRIPTION, GIT_DESCRIPTION is not set)
	@echo "Generating and templating version information -> $(VERSION)"
	sed \
		-e 's!{{VERSION}}!$(VERSION)!g' \
		-e 's!{{GITBRANCH}}!$(GIT_BRANCH)!g' \
		-e 's!{{GITDIRTY}}!$(GIT_DIRTY)!g' \
		-e 's!{{GITCOMMIT}}!$(GIT_COMMIT)!g' \
		-e 's!{{GITDESCRIPTION}}!$(GIT_DESCRIPTION)!g' \
		< VERSION-template.py | $(WRITE_IFCHANGED) $@

## Stuff below here is from build-aux

WRITE_IFCHANGED = $(BUILDER_HOME)/write-if-changed.sh

# Sometimes we have a file-target that we want Make to always try to
# re-generate (such as compiling a Go program; we would like to let
# `go install` decide whether it is up-to-date or not, rather than
# trying to teach Make how to do that).  We could mark it as .PHONY,
# but that tells Make that "this isn't a "this isn't a real file that
# I expect to ever exist", which has a several implications for Make,
# most of which we don't want.  Instead, we can have them *depend* on
# a .PHONY target (which we'll name "FORCE"), so that they are always
# considered out-of-date by Make, but without being .PHONY themselves.
.PHONY: FORCE
