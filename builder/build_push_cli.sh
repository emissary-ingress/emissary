#!/usr/bin/env bash

# Copyright 2019 Datawire. All rights reserved.
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

# Where is this script?
SRCDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

# Set BUILD_VERSION and RELEASE_VERSION (and other junk)
eval "$("${SRCDIR}/module_version.sh" unused)"

case "$(go env GOOS)" in
windows)
    EXE_NAME=edgectl.exe
    ;;
*)
    EXE_NAME=edgectl
    ;;
esac

DIST=~/bin
EXE_PATH=${DIST}/${EXE_NAME}

case "$1" in
    build)
        cd "${SRCDIR}/.." && go build -trimpath -ldflags "-X main.Version=$BUILD_VERSION" -o "${EXE_PATH}" ./cmd/edgectl
        ;;
    push)
        # Push this OS/arch binary
	    aws s3 cp --acl public-read \
            "${EXE_PATH}" \
            "s3://datawire-static-files/edgectl/${RELEASE_VERSION}/$(go env GOOS)/$(go env GOARCH)/${EXE_NAME}"
        ;;
    tag)
        # Update latest.txt
        echo "$RELEASE_VERSION" | aws s3 cp --acl public-read - s3://datawire-static-files/edgectl/latest.txt
        ;;
    promote)
        # Replace stable.txt with the contents of latest.txt
        aws s3 cp --acl public-read \
            s3://datawire-static-files/edgectl/latest.txt \
            s3://datawire-static-files/edgectl/stable.txt
        ;;
    *)
        echo "usage: $0 {build|push|tag|promote}"
        exit 1
        ;;
esac
