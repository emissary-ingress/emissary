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

# Travis itself prevents launch on a nobuild branch _unless_ it's a PR from a
# nobuild branch.
# if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
#     printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
#     exit 0
# fi

# Basically everything for a GA commit happens from the deploy target.
if [ "${COMMIT_TYPE}" != "GA" ]; then
    make test
    make docker-push

    make website

    if [[ ${GIT_BRANCH} = ${MAIN_BRANCH} ]]; then
        # By fiat, _any commit_ on the main branch pushes production docs.
        # This is to allow simple doc fixes. So. Grab the most recent proper
        # version...
        VERSION=$(git describe --tags --abbrev=0 --exclude='*-*')

        if [ -z "$VERSION" ]; then
            # Uh WTF.
            echo "No tagged version found at $GIT_COMMIT" >&2
            exit 1
        fi

        if [[ $VERSION =~ '^v' ]]; then
            VERSION=$(echo "$VERSION" | cut -c2-)
        fi

        echo make VERSION=$(VERSION) DOC_RELEASE_TYPE=stable publish-website
    else
        # Anything else, push staging.
        make publish-website
    fi        

    # Run E2E if this isn't a nobuild branch, nor a doc branch, nor a random commit not on the main branch.
    SKIP_E2E=

    if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
        SKIP_E2E=yes
    fi

    if [[ ${GIT_BRANCH} =~ ^doc.* ]]; then
        SKIP_E2E=yes
    fi

    if [[ ( ${GIT_BRANCH} != ${MAIN_BRANCH} ) && ( ${COMMIT_TYPE} == "random" ) ]]; then
        SKIP_E2E=yes
    fi

    if [ -z "$SKIP_E2E" ]; then
        make e2e
    fi
else
    echo "GA commit, will retag in deployment"
fi

# All the artifact handling for GA builds happens in the deploy block
# in travis.yml, so we're done here.

printf "== End:   travis-script.sh ==\n"
