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

printf "========\nSetting up environment...\n"
case "$COMMIT_TYPE" in
    GA)
        eval $(make DOCKER_EXTERNAL_REGISTRY=$DOCKER_REGISTRY export-vars)
        ;;
    *)
        eval $(make USE_KUBERNAUT=true \
                    DOCKER_EPHEMERAL_REGISTRY=true \
                    DOCKER_EXTERNAL_REGISTRY=$DOCKER_REGISTRY \
                    DOCKER_REGISTRY=localhost:31000 \
                    export-vars)
        ;;
esac
set -o xtrace
make print-vars

printf "========\nStarting build...\n"

case "$COMMIT_TYPE" in
    GA)
        : # We just re-tag the RC image as GA; nothing to build
        ;;
    *)
        # CI might have set DOCKER_BUILD_USERNAME and DOCKER_BUILD_PASSWORD
        # (in case BASE_DOCKER_REPO is private)
        if [[ -n "${DOCKER_BUILD_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_BUILD_USERNAME" --password-stdin "${BASE_DOCKER_REPO%%/*}" <<<"$DOCKER_BUILD_PASSWORD"
        fi

        make setup-develop cluster.yaml docker-registry
        make docker-push DOCKER_PUSH_AS="$AMBASSADOR_DOCKER_IMAGE" # to the in-cluster registry
        # make KAT_REQ_LIMIT=1200 test
        make test
        ;;
esac

printf "========\nPublishing artifacts...\n"

case "$COMMIT_TYPE" in
    GA)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${AMBASSADOR_EXTERNAL_DOCKER_REPO%%/*}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make release
        ;;
    RC)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${AMBASSADOR_EXTERNAL_DOCKER_REPO%%/*}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${GIT_TAG_SANITIZED}" # public X.Y.Z-rcA
        make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${LATEST_RC}"         # public X.Y.Z-rc-latest
        make VERSION="$VERSION" SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
        ;;
    EA)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${AMBASSADOR_EXTERNAL_DOCKER_REPO%%/*}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make docker-push DOCKER_PUSH_AS="${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${GIT_TAG_SANITIZED}" # public X.Y.Z-eaA
        make VERSION="$VERSION" SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
        ;;
    *)
        : # Nothing to do
        ;;
esac

printf "== End:   travis-script.sh ==\n"
