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

update-aws() {
    if [ -z "${AWS_ACCESS_KEY_ID}" ]; then
        @echo 'AWS credentials not configured; not updating either https://s3.amazonaws.com/datawire-static-files/ambassador/$(STABLE_TXT_KEY) or the latest version in Scout'
        exit
    fi

    if [ -n "${STABLE_TXT_KEY}" ]; then
        printf "${RELEASE_VERSION}" > stable.txt
        echo "updating ${STABLE_TXT_KEY} with $(cat stable.txt)"
        aws s3api put-object \
            --bucket datawire-static-files \
            --key ambassador/${STABLE_TXT_KEY} \
            --body stable.txt
    fi

    if [ -n "${SCOUT_APP_KEY}" ]; then
        printf '{"application":"ambassador","latest_version":"${RELEASE_VERSION}","notices":[]}' > app.json
        echo "updating ${SCOUT_APP_KEY} with $(cat app.json)"
        aws s3api put-object \
            --bucket scout-datawire-io \
            --key ambassador/$(SCOUT_APP_KEY) \
            --body app.json
    fi
}


printf "== Begin: travis-script.sh ==\n"

if [[ -n "$TRAVIS_TAG" ]]; then
    if [[ "$TRAVIS_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        COMMIT_TYPE=GA
    elif [[ "$TRAVIS_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]+$ ]]; then
        COMMIT_TYPE=RC
    elif [[ "$TRAVIS_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-ea[0-9]+$ ]]; then
        COMMIT_TYPE=EA
    else
        echo "TRAVIS_TAG '$TRAVIS_TAG' is not in one of the recognized tag formats:" >&2
        echo " - 'vSEMVER'" >&2
        echo " - 'vSEMVER-rcN'" >&2
        echo " - 'vSEMVER-eaN'" >&2
        echo "Note that the tag name must start with a lowercase 'v'" >&2
        exit 1
    fi
elif [[ "$TRAVIS_PULL_REQUEST" != false ]]; then
    COMMIT_TYPE=PR
else
    COMMIT_TYPE=random
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

set -o xtrace

printf "========\nStarting build...\n"

case "$COMMIT_TYPE" in
    GA)
        : # We just re-tag the RC image as GA; nothing to build
        ;;
    *)
        # CI might have set DOCKER_BUILD_USERNAME and DOCKER_BUILD_PASSWORD
        # (in case BASE_DOCKER_REPO is private)
        docker login -u="${DOCKER_BUILD_USERNAME:-datawire-dev+ci}" --password-stdin "${DEV_REGISTRY}" <<<"${DOCKER_BUILD_PASSWORD:-CEAWVNREJHTOAHSOJFJHJZQYI7H9MELSU1RG1CD6XIFAURD5D7Y1N8F8MU0JO912}"

        [ "$TRAVIS_OS_NAME" = "linux" ] && make test
        ;;
esac

printf "========\nPublishing artifacts...\n"

case "$COMMIT_TYPE" in
    GA)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${RELEASE_REGISTRY}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make release
        # Promote for Edge Control binary (copy from latest.txt to stable.txt) on Linux.
        [ "$TRAVIS_OS_NAME" = "linux" ] && ./releng/build-cli.sh promote
        # XXX
	#SCOUT_APP_KEY=app.json STABLE_TXT_KEY=stable.txt update-aws
        ;;
    RC)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${RELEASE_REGISTRY}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make rc
        # Build/push Edge Control binary. Tag (set latest.txt) on Linux.
        ./releng/build-cli.sh build
        ./releng/build-cli.sh push
        [ "$TRAVIS_OS_NAME" = "linux" ] && ./releng/build-cli.sh tag
        # XXX
	#SCOUT_APP_KEY=testapp.json STABLE_TXT_KEY=teststable.txt update-aws
        ;;
    EA)
        if [[ -n "${DOCKER_RELEASE_USERNAME:-}" ]]; then
            docker login -u="$DOCKER_RELEASE_USERNAME" --password-stdin "${RELEASE_REGISTRY}" <<<"$DOCKER_RELEASE_PASSWORD"
        fi
        make rc
        # Build/push Edge Control binary. Don't tag.
        ./releng/build-cli.sh build
        ./releng/build-cli.sh push
        # XXX
        #SCOUT_APP_KEY=earlyapp.json STABLE_TXT_KEY=earlystable.txt update-aws
        ;;
    *)
        : # Nothing to do
        ;;
esac

printf "== End:   travis-script.sh ==\n"
