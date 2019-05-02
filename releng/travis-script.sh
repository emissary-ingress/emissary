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

printf "== Begin: travis-script.sh ==\n"

# We start by figuring out the COMMIT_TYPE. Yes, this is kind of a hack.
eval $(make export-vars | grep COMMIT_TYPE)

printf "========\nCOMMIT_TYPE $COMMIT_TYPE; git status:\n"

git status

printf "========\n"

# Travis itself prevents launch on a nobuild branch _unless_ it's a PR from a
# nobuild branch.
# if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
#     printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
#     exit 0
# fi

# Basically everything for a GA commit happens from the deploy target.
if [ "${COMMIT_TYPE}" != "GA" ]; then
    # Set up the environment correctly, including the madness around
    # the ephemeral Docker registry.
    printf "========\nSetting up environment...\n"

    eval $(make USE_KUBERNAUT=true \
                DOCKER_EPHEMERAL_REGISTRY=true \
                DOCKER_EXTERNAL_REGISTRY=$DOCKER_REGISTRY \
                DOCKER_REGISTRY=localhost:31000 \
                export-vars)

    # Makes it much easier to actually debug when you see what the Makefile sees
    make print-vars

    printf "========\nStarting build...\n"

    make setup-develop cluster.yaml docker-registry
    make docker-push
    make KAT_REQ_LIMIT=900 test

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
    fi

    if [[ ${COMMIT_TYPE} == "RC" ]]; then
        # For RC builds, update AWS test keys.
		make VERSION="$VERSION" SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
    elif [[ ${COMMIT_TYPE} == "EA" ]]; then
        # For RC builds, update AWS EA keys.
		make VERSION="$VERSION" SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
    fi
else
    echo "GA commit, will retag in deployment"
fi

# All the artifact handling for GA builds happens in the deploy block
# in travis.yml, so we're done here.

printf "== End:   travis-script.sh ==\n"
