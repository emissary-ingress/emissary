#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o xtrace

# Makes it much easier to actually debug when you see what the Makefile sees
make print-vars
git status

# IMPORTANT: no custom logic about shell variables goes here. The Makefile 
# sets them all, because we want make to work when a developer runs it by 
# hand. 
#
# All we get to do here is to copy things that make understands.
eval $(make export-vars)
git status

printf "== Begin: travis-script.sh ==\n"

if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
    printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
    exit 0
fi

# Basically everything for a GA commit happens from the deploy target.
if [ "${COMMIT_TYPE}" != "GA" ]; then
    make test
    git status
    make docker-push
    git status
    make website
    git status
    make publish-website
    git status
fi

# E2E happens unless this is a random commit not on the main branch.
if [ \( "${GIT_BRANCH}" = "${MAIN_BRANCH}" \) -o \( "${COMMIT_TYPE}" != "random" \) ]; then
    make e2e
fi

# All the artifact handling for GA builds happens in the deploy block
# in travis.yml, so we're done here.

printf "== End:   travis-script.sh ==\n"
