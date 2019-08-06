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

printf "== Begin: travis-script.sh ==\n"

if [[ "$GIT_BRANCH" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    COMMIT_TYPE=GA
elif [[ "$GIT_BRANCH" =~ -rc[0-9]+$ ]]; then
    COMMIT_TYPE=RC
elif [[ "$GIT_BRANCH" =~ -ea[0-9]+$ ]]; then
    COMMIT_TYPE=EA
elif [[ "$TRAVIS_PULL_REQUEST" != false ]]; then
    COMMIT_TYPE=PR
else
    COMMIT_TYPE=random
fi

printf "========\nCOMMIT_TYPE $COMMIT_TYPE; git status:\n"

git status

printf "========\n"

# Travis itself prevents launch on a nobuild branch _unless_ it's a PR from a
# nobuild branch.
# if [[ ${GIT_BRANCH} =~ ^nobuild.* ]]; then
#     printf "!! Branch is 'nobuild', therefore, no work will be performed.\n"
#     exit 0
# fi

if [ "${COMMIT_TYPE}" != "GA" ]; then
    # Set up the environment correctly, including the madness around
    # the ephemeral Docker registry.
    printf "========\nSetting up environment...\n"

    eval $(make USE_KUBERNAUT=true \
                DOCKER_EPHEMERAL_REGISTRY=true \
                DOCKER_EXTERNAL_REGISTRY=$DOCKER_REGISTRY \
                DOCKER_REGISTRY=localhost:31000 \
                export-vars)
    set -o xtrace

    # Makes it much easier to actually debug when you see what the Makefile sees
    make print-vars

    printf "========\nStarting build...\n"

    make setup-develop cluster.yaml docker-registry
    make docker-push DOCKER_PUSH_AS="$AMBASSADOR_DOCKER_IMAGE" # to the in-cluster registry
    case "$COMMIT_TYPE" in
        RC)
            make docker-login
            make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${GIT_TAG_SANITIZED}" # public X.Y.Z-rcA
            make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${LATEST_RC}"         # public X.Y.Z-rc-latest
            ;;
        EA)
            make docker-login
            make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${GIT_TAG_SANITIZED}" # public X.Y.Z-eaA
            ;;
    esac

    printf "========\nkubectl version...\n"
    kubectl version

    # make KAT_REQ_LIMIT=1200 test
    make test

    if [[ ${COMMIT_TYPE} == "RC" ]]; then
        # For RC builds, update AWS test keys.
        make VERSION="$VERSION" SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
    elif [[ ${COMMIT_TYPE} == "EA" ]]; then
        # For RC builds, update AWS EA keys.
        make VERSION="$VERSION" SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
    fi
else
    eval $(make DOCKER_EXTERNAL_REGISTRY=$DOCKER_REGISTRY export-vars)
    set -o xtrace
    make print-vars

    # retag
    make docker-login
    make release
fi

printf "== End:   travis-script.sh ==\n"
