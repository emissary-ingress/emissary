#!/usr/bin/env bash
set -o errexit
set -o nounset

# Makes it much easier to actually debug when you see what the Makefile sees
make print-vars

MAIN_BRANCH="$(make print-MAIN_BRANCH)"

GIT_TAG="$(make print-GIT_TAG_SANITIZED)"
GIT_BRANCH="$(make print-GIT_BRANCH)"

printf "== Begin: travis-script.sh ==\n"

if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
    printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
    exit 0
fi

if [[ -z ${GIT_TAG} ]] ; then
    make test
    make docker-images docker-push
    make website
fi

if [[ ${GIT_BRANCH} == ${MAIN_BRANCH} ]] || \
   [[ $(make print-IS_PULL_REQUEST) == "true" ]]; then
    make e2e
fi

printf "== End:   travis-script.sh ==\n"
