#!/usr/bin/env bash

# FIXME: This is *copied* from builder.sh. Duplicate code!

module_version() {
    echo MODULE="\"$1\""
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
    for VAR in "${TRAVIS_PULL_REQUEST_BRANCH}" "${TRAVIS_BRANCH}" $(git rev-parse --abbrev-ref HEAD); do
        if [ -n "${VAR}" ]; then
            echo GIT_BRANCH="\"${VAR}\""
            break
        fi
    done
    # The short git commit hash
    echo GIT_COMMIT="\"$(git rev-parse --short HEAD)\""
    # Whether `git add . && git commit` would commit anything (empty=false, nonempty=true)
    if [ -n "$(git status --porcelain)" ]; then
        echo GIT_DIRTY="\"dirty\""
        dirty="yes"
    else
        echo GIT_DIRTY="\"\""
        dirty=""
    fi
    # The _previous_ tag, plus a git delta, like 0.36.0-436-g8b8c5d3
    echo GIT_DESCRIPTION="\"$(git describe --tags)\""

    # RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
    # will upload/release the image as.  It does NOT include a leading 'v'
    # (trimming the 'v' from the git tag is what the 'patsubst' is for).
    # If this is an RC or EA, then it includes the '-rc.N' or '-ea.N'
    # suffix.
    #
    # BUILD_VERSION is of the same format, but is the version number that
    # we build into the image.  Because an image built as a "release
    # candidate" will ideally get promoted to be the GA image, we trim off
    # the '-rcN' suffix.
    for VAR in "${TRAVIS_TAG}" "$(git describe --tags --always)"; do
        if [ -n "${VAR}" ]; then
            RELEASE_VERSION="${VAR}"
            break
        fi
    done

    if [[ ${RELEASE_VERSION} =~ ^v[0-9]+.*$ ]]; then
        RELEASE_VERSION=${RELEASE_VERSION:1}
    fi

    if [ -n "${dirty}" ]; then
        RELEASE_VERSION="${RELEASE_VERSION}-dirty"
    fi

    echo RELEASE_VERSION="\"${RELEASE_VERSION}\""
    echo BUILD_VERSION="\"$(echo "${RELEASE_VERSION}" | sed 's/-rc\.[0-9]*$//')\""
}

module_version "$@"
