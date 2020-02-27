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

# NOTE WELL: at the moment, this code needs to run outside the builder
# container, because it uses module_version.sh, which needs the git 
# tree, which the build container doesn't have. This is a bug that needs 
# fixing.

usage () {
    echo "usage: $0 (build|push|push-private|tag|promote) cli_name" >&2
    echo "" >&2
    echo "e.g. $0 build edgectl" >&2
    echo "     $0 promote apictl-key" >&2
}

set -o errexit

cmd="$1"
cli_name="$2"

if [ -z "$cmd" -o -z "$cli_name" ]; then
    usage
    exit 1
fi

# Where is this script?
SRCDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

CMD_DIR="${SRCDIR}/.."
if [ -d "${SRCDIR}/../cmd/${cli_name}" ]; then
    # we're good, do nothing
    :
elif [ -d "${SRCDIR}/../ambassador/cmd/${cli_name}" ]; then
    CMD_DIR="${SRCDIR}/../ambassador"
else
    echo "could not find cmd/${cli_name} in ${SRCDIR}/.. or ${SRCDIR}/../ambassador" >&2
    exit 1
fi

# Set BUILD_VERSION and RELEASE_VERSION (and other junk)
eval "$("${SRCDIR}/module_version.sh" unused)"

case "$(go env GOOS)" in
windows)
    EXE_NAME=${cli_name}.exe
    ;;
*)
    EXE_NAME=${cli_name}
    ;;
esac

DIST=~/bin
EXE_PATH=${DIST}/${EXE_NAME}

case "$cmd" in
    build)
        cd "${CMD_DIR}" && go build -trimpath -ldflags "-X main.Version=$BUILD_VERSION" -o "${EXE_PATH}" ./cmd/${cli_name}
        ;;
    push)
        # Push this OS/arch binary
        aws s3 cp --acl public-read \
            "${EXE_PATH}" \
            "s3://datawire-static-files/${cli_name}/${RELEASE_VERSION}/$(go env GOOS)/$(go env GOARCH)/${EXE_NAME}"
        ;;
    push-private)
        # Push this OS/arch binary, but don't override the ACL: the defaults for our S3 bucket are 
        # private access only.
        aws s3 cp \
            "${EXE_PATH}" \
            "s3://datawire-static-files/${cli_name}/${RELEASE_VERSION}/$(go env GOOS)/$(go env GOARCH)/${EXE_NAME}"
        ;;
    tag)
        # Update latest.txt
        echo "$RELEASE_VERSION" | aws s3 cp --acl public-read - s3://datawire-static-files/${cli_name}/latest.txt
        ;;
    promote)
        # Replace stable.txt with the contents of latest.txt
        aws s3 cp --acl public-read \
            s3://datawire-static-files/${cli_name}/latest.txt \
            s3://datawire-static-files/${cli_name}/stable.txt
        ;;
    *)
        usage
        exit 1
        ;;
esac
