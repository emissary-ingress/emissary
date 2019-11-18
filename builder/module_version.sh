#!/usr/bin/env bash

# FIXME: This is *copied* from builder.sh. Duplicate code!

module_version() (
    shopt -s extglob
    set -o nounset
    set -o errexit

    # shellcheck disable=SC2030
    local \
        MODULE \
        GIT_BRANCH \
        GIT_COMMIT \
        GIT_DIRTY \
        GIT_DESCRIPTION \
        RELEASE_VERSION \
        BUILD_VERSION

    MODULE="$1"

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
    GIT_BRANCH="${TRAVIS_PULL_REQUEST_BRANCH:-${TRAVIS_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}}"

    # The short git commit hash
    GIT_COMMIT="$(git rev-parse --short HEAD)"

    # Whether `git add . && git commit` would commit anything (empty=false, nonempty=true)
    if [ -n "$(git status --porcelain)" ]; then
        GIT_DIRTY='dirty'
    else
        GIT_DIRTY=''
    fi

    # The _previous_ tag, plus a git delta, like 0.36.0-436-g8b8c5d3
    GIT_DESCRIPTION="$(git describe --tags)"

    # RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
    # will upload/release the image as.  It does NOT include a leading 'v'
    # (trimming the 'v' from the git tag is what the 'patsubst' is for).
    # If this is an RC or EA, then it includes the '-rcN' or '-eaN'
    # suffix.
    RELEASE_VERSION="${TRAVIS_TAG:-$(git describe --tags --always)}"
    RELEASE_VERSION="${RELEASE_VERSION#v}"
    RELEASE_VERSION+="${GIT_DIRTY:+-dirty}"

    # BUILD_VERSION is of the same format, but is the version number that
    # we build into the image.  Because an image built as a "release
    # candidate" will ideally get promoted to be the GA image, we trim off
    # the '-rcN' suffix.
    BUILD_VERSION="${RELEASE_VERSION%%-rc*([0-9])}"

    printf '%s=%q\n' \
           MODULE "$MODULE" \
           GIT_BRANCH "$GIT_BRANCH" \
           GIT_COMMIT "$GIT_COMMIT" \
           GIT_DIRTY "$GIT_DIRTY" \
           GIT_DESCRIPTION "$GIT_DESCRIPTION" \
           RELEASE_VERSION "$RELEASE_VERSION" \
           BUILD_VERSION "$BUILD_VERSION"
)

module_version "$@"
