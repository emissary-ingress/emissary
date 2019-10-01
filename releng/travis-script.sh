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

NEEDS_V=

if [[ "$GIT_BRANCH" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    COMMIT_TYPE=GA
    NEEDS_V=yes
elif [[ "$GIT_BRANCH" =~ -rc[0-9]+$ ]]; then
    COMMIT_TYPE=RC
    NEEDS_V=yes
elif [[ "$GIT_BRANCH" =~ -ea[0-9]+$ ]]; then
    COMMIT_TYPE=EA
    NEEDS_V=yes
elif [[ "$TRAVIS_PULL_REQUEST" != false ]]; then
    COMMIT_TYPE=PR
else
    COMMIT_TYPE=random
fi

if [ -n "$NEEDS_V" ]; then
    # GIT_BRANCH must start with a 'v' for consistency here. The Makefile yanks off
    # the 'v' in the version number.
    if ! [[ "$GIT_BRANCH" =~ ^[vV] ]]; then
        echo "GIT_BRANCH '$GIT_BRANCH' does not start with a 'v'" >&2
        exit 1
    fi
fi

# If downstream, don't re-run release machinery for tags that are an
# existing upstream release.
if [[ "$TRAVIS_REPO_SLUG" != datawire/ambassador ]] &&
   [[ -n "${TRAVIS_TAG:-}" ]] &&
   git fetch https://github.com/datawire/ambassador.git "refs/tags/${TRAVIS_TAG}:refs/upstream-tag" &&
   [[ "$(git rev-parse refs/upstream-tag)" == "$(git rev-parse "refs/tags/${TRAVIS_TAG}")" ]]
then
    COMMIT_TYPE=random
fi
git update-ref -d refs/upstream-tag

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

case "$TRAVIS_EVENT_TYPE" in
    cron)
        printf "========\nRunning Envoy tests...\n"

        gcloud beta compute --project=datawireio instances create envoy-tests-ambassador --zone=us-east1-b --machine-type=n1-highcpu-32 --image=ubuntu-1904-disco-v20190918 --image-project=ubuntu-os-cloud --boot-disk-size=200GB --boot-disk-type=pd-ssd --boot-disk-device-name=envoy-tests-ambassador

        gcloud beta compute --project "datawireio" ssh --zone "us-east1-b" "envoy-tests-ambassador" << EOF

        sudo apt-get update
        sudo apt-get install -y docker.io git make golang

        git clone https://github.com/datawire/ambassador
        cd ambassador
        git fetch origin pull/1873/head:1873
        git checkout 1873

        sudo usermod -aG docker $USER
EOF

        gcloud beta compute --project "datawireio" ssh --zone "us-east1-b" "envoy-tests-ambassador" << EOF
        export DOCKER_REGISTRY=-
        cd ambassador
        make envoy-tests
EOF

        gcloud beta compute instances delete envoy-tests-ambassador --zone us-east1-b --quiet
    ;;
    *)
        printf "Skipping Envoy tests"
    ;;
esac

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
        make docker-push docker-push-kat-client docker-push-kat-server # to the in-cluster registry (DOCKER_REGISTRY)
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
        tags=(
            "${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${VERSION}" # public X.Y.Z-rcA
            "${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${LATEST_RC}"         # public X.Y.Z-rc-latest
        )
        for tag in "${tags[@]}"; do
            docker tag "$AMBASSADOR_DOCKER_IMAGE" "$tag"
            docker push "$tag"
        done
        make VERSION="$VERSION" SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
        ;;
    EA)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${AMBASSADOR_EXTERNAL_DOCKER_REPO%%/*}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        tags=(
            "${AMBASSADOR_EXTERNAL_DOCKER_REPO}:${VERSION}" # public X.Y.Z-eaA
        )
        for tag in "${tags[@]}"; do
            docker tag "$AMBASSADOR_DOCKER_IMAGE" "$tag"
            docker push "$tag"
        done
        make VERSION="$VERSION" SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
        ;;
    *)
        : # Nothing to do
        ;;
esac

printf "== End:   travis-script.sh ==\n"
