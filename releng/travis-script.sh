#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o xtrace

# Makes it much easier to actually debug when you see what the Makefile sees
make print-vars

# IMPORTANT: no custom logic about shell variables goes here. The Makefile 
# sets them all, because we want make to work when a developer runs it by 
# hand. 
#
# All we get to do here is to copy things that make understands.
MAIN_BRANCH="$(make print-MAIN_BRANCH)"
COMMIT_TYPE="$(make print-COMMIT_TYPE)"

GIT_TAG="$(make print-GIT_TAG_SANITIZED)"
GIT_BRANCH="$(make print-GIT_BRANCH)"

printf "== Begin: travis-script.sh ==\n"

if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
    printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
    exit 0
fi

# Don't rebuild a GA commit. All we need to do here is to do doc stuff.
if [ "${COMMIT_TYPE}" != "GA" ]; then
    make test
    make docker-push
fi

# In all cases, do the doc stuff.
make website
make publish-website

# E2E happens unless this is a random commit not on the main branch.
if [ \( "${GIT_BRANCH}" = "${MAIN_BRANCH}" \) -o \( "${COMMIT_TYPE}" != "random" \) ]; then
    make e2e
fi

# All the artifact handling for GA builds happens in the deploy block
# in travis.yml, so we're done here.

printf "== End:   travis-script.sh ==\n"
