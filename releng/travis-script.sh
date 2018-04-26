#!/usr/bin/env bash
set -o errexit
set -o nounset

BLESSED_BRANCH="${BLESSED_BRANCH:?BLESSED_BRANCH is not set or empty}"
GIT_BRANCH="${GIT_BRANCH:?GIT_BRANCH is not set or empty}"

TRAVIS_TAG="${TRAVIS_TAG:-''}"
TRAVIS_PULL_REQUEST="${TRAVIS_PULL_REQUEST:?TRAVIS_PULL_REQUEST is not set or empty}"

VERSION="${VERSION:?VERSION is not set or empty}"

printf "== Begin: travis-script.sh ==\n"

make print-vars

if [[ "$TRAVIS_TAG" == "$VERSION" && ! "${TRAVIS_PULL_REQUEST_BRANCH}" =~ ^nobuild.* ]]; then
    printf "== Begin: execute tests\n"

    make test

    printf "== End:   execute tests\n"

    printf "== Begin: build and push docker image\n"

    make docker-images
    make docker-push

    printf "== End:   build and push docker image\n"

    printf "== Begin: generate documentation\n"

    make website

    printf "== End:   generate documentation\n"
fi

if [[ ("${GIT_BRANCH}" == "${BLESSED_BRANCH}") || ("${TRAVIS_PULL_REQUEST}" == "true" && ! "${GIT_BRANCH}" =~ ^nobuild.*) ]]; then
    printf "== Begin: e2e test execution\n"
    #make e2e
    printf "== End:   e2e test execution\n"
fi

printf "== End:   travis-script.sh ==\n"
