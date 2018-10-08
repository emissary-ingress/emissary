#!/usr/bin/env bash

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

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
    make docker-push
    make test

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

        echo "making stable docs for $VERSION"
        make VERSION="$VERSION" DOC_RELEASE_TYPE=stable website
    else
        # Anything else, push staging.

        echo "making draft docs for $VERSION"
        make website
    fi        

    #### RUN END-TO-END TESTS ONLY ON RC BUILDS
    ####
    #### The end-to-end tests, running sequentially, take around half an hour 
    #### to run. Allowing them to run while collecting things for release is
    #### just shredding velocity right now.
    ####
    #### Work is underway to parallelize the end-to-end tests (or to more
    #### majorly refactor around needing them) but for now, we're only going to
    #### run them on RC builds -- which is to say, do all the testing you can
    #### in development, collect things for a release, and the RC build will run
    #### the final E2E pass.
    
    SKIP_E2E=

    if [[ ${COMMIT_TYPE} != "RC" ]]; then
        SKIP_E2E=yes
    fi

    # # Run E2E if this isn't a nobuild branch, nor a doc branch, nor a random commit not on the main branch.
    # if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
    #     SKIP_E2E=yes
    # fi

    # if [[ ${GIT_BRANCH} =~ ^doc.* ]]; then
    #     SKIP_E2E=yes
    # fi

    # if [[ ( ${GIT_BRANCH} != ${MAIN_BRANCH} ) && ( ${COMMIT_TYPE} == "random" ) ]]; then
    #     SKIP_E2E=yes
    # fi

    if [ -z "$SKIP_E2E" ]; then
        make e2e
    fi

    # For RC builds, update AWS test keys.
    if [[ ${COMMIT_TYPE} == "RC" ]]; then
		make VERSION="$VERSION" SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
    fi
else
    echo "GA commit, will retag in deployment"
fi

# All the artifact handling for GA builds happens in the deploy block
# in travis.yml, so we're done here.

printf "== End:   travis-script.sh ==\n"
